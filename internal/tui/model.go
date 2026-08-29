package tui

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"charm.land/bubbles/v2/textinput"
	"charm.land/bubbles/v2/viewport"
	tea "charm.land/bubbletea/v2"

	"github.com/lum1n/dotr/internal/backup"
	"github.com/lum1n/dotr/internal/config"
	"github.com/lum1n/dotr/internal/gitstatus"
	"github.com/lum1n/dotr/internal/ignore"
	"github.com/lum1n/dotr/internal/load"
	"github.com/lum1n/dotr/internal/preview"
	"github.com/lum1n/dotr/internal/scan"
	"github.com/lum1n/dotr/internal/secret"
	"github.com/lum1n/dotr/internal/stow"
	"github.com/lum1n/dotr/internal/symlink"
	"github.com/lum1n/dotr/internal/watch"
)

type focusPane int

const (
	focusList focusPane = iota
	focusPreview
)

type mode int

const (
	modeBrowse mode = iota
	modeIgnores
	modeFilter
	modePaste
	modeRestore
	modeHelp
	modeNew
	modeConfirm
	modeStow
	modeLinkPath
	modeLinkTarget
	modeRetarget
)

type confirmKind int

const (
	confirmNone confirmKind = iota
	confirmYankContents
	confirmBackup
	confirmUnstow
	confirmDeleteSymlink
)

type scanDoneMsg struct {
	entries   []scan.Entry
	ignores   *ignore.List
	cfg       config.Config
	git       map[string]gitstatus.Kind
	truncated bool
	err       error
}

type previewDoneMsg struct {
	id     int
	result preview.Result
}

type previewTickMsg struct {
	id    int
	path  string
	width int
}

type ignoreSavedMsg struct {
	pattern string
	err     error
}

type backupDoneMsg struct {
	snap backup.Snapshot
	err  error
}

type restoreDoneMsg struct {
	path string
	err  error
}

type pasteDoneMsg struct {
	path string
	err  error
}

type newFileDoneMsg struct {
	path string
	err  error
}

type symlinkDoneMsg struct {
	op   string
	path string
	err  error
}

type fsEventMsg struct {
	path string
	op   string
}

type watcherReadyMsg struct {
	w *watch.Watcher
}

type gitDoneMsg struct {
	git map[string]gitstatus.Kind
}

type stowDoneMsg struct {
	opts   stow.Options
	pkgs   []stow.Package
	owners map[string]string
	err    error
}

type stowOpDoneMsg struct {
	action stow.Action
	pkgs   []string
	out    string
	err    error
}

type rescanTickMsg struct{}

// Model is the dual-pane dotr TUI.
type Model struct {
	styles styles
	cfg    config.Config

	width  int
	height int
	ready  bool
	mode   mode

	entries  []scan.Entry
	cursor   int
	offset   int
	filter   string
	filtered []int

	ignores      *ignore.List
	ignoreCursor int
	ignoreOffset int

	input textinput.Model

	yankPath string

	snapshots []backup.Snapshot
	snapCur   int
	snapOff   int

	confirm     confirmKind
	confirmPath string
	confirmPkg  string

	linkPath string // pending symlink path while prompting for target

	git map[string]gitstatus.Kind

	stowOpts   stow.Options
	stowPkgs   []stow.Package
	stowCur    int
	stowOff    int
	stowOwners map[string]string
	truncated  bool

	watcher *watch.Watcher

	viewport viewport.Model
	focus    focusPane

	previewID    int
	previewPath  string
	previewLang  string
	previewBadge string
	previewParse preview.ParseStatus
	previewErr   string
	status       string
	scanning     bool
	err          error

	junkBuf string
	junkID  int
}

// New creates the initial model.
func New() Model {
	vp := viewport.New(viewport.WithWidth(20), viewport.WithHeight(10))
	vp.SoftWrap = false
	vp.MouseWheelEnabled = true

	ti := textinput.New()
	ti.CharLimit = 256
	ti.Prompt = "/ "
	ti.Placeholder = "search apps & paths…"

	return Model{
		styles:   newStyles(),
		cfg:      config.Default(),
		viewport: vp,
		input:    ti,
		focus:    focusList,
		scanning: true,
		status:   "scanning…",
		ignores:    &ignore.List{},
		git:        map[string]gitstatus.Kind{},
		stowOwners: map[string]string{},
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(scanCmd(), startWatcherCmd())
}

func scanCmd() tea.Cmd {
	return func() tea.Msg {
		res, err := load.All()
		if err != nil {
			return scanDoneMsg{err: err}
		}
		// Git marks load in the background after the list is shown.
		return scanDoneMsg{
			entries:   res.Entries,
			ignores:   res.Ignores,
			cfg:       res.Config,
			truncated: res.Truncated,
		}
	}
}

func startWatcherCmd() tea.Cmd {
	return func() tea.Msg {
		home, configHome, err := scan.Roots()
		if err != nil {
			return watcherReadyMsg{}
		}
		w, err := watch.New(home, configHome)
		if err != nil {
			return watcherReadyMsg{}
		}
		return watcherReadyMsg{w: w}
	}
}

func waitWatcherCmd(w *watch.Watcher) tea.Cmd {
	if w == nil {
		return nil
	}
	return func() tea.Msg {
		ev, ok := <-w.Events()
		if !ok {
			return nil
		}
		return fsEventMsg{path: ev.Path, op: ev.Op}
	}
}

func refreshGitCmd(paths []string) tea.Cmd {
	return func() tea.Msg {
		return gitDoneMsg{git: gitstatus.Map(paths)}
	}
}

func refreshStowCmd(cfg config.Config, entries []scan.Entry) tea.Cmd {
	return func() tea.Msg {
		opts, err := stow.Resolve(cfg.StowDir, cfg.StowTarget)
		if err != nil {
			return stowDoneMsg{err: err}
		}
		pkgs, err := stow.AnalyzeAll(opts)
		if err != nil {
			return stowDoneMsg{opts: opts, err: err}
		}
		paths := make([]string, len(entries))
		for i, e := range entries {
			paths[i] = e.AbsPath
		}
		return stowDoneMsg{
			opts:   opts,
			pkgs:   pkgs,
			owners: stow.OwnershipMap(opts, paths),
		}
	}
}

func stowOpCmd(opts stow.Options, action stow.Action, pkgs []string) tea.Cmd {
	return func() tea.Msg {
		out, err := stow.Run(opts, action, pkgs, false)
		return stowOpDoneMsg{action: action, pkgs: pkgs, out: out, err: err}
	}
}

func previewCmd(id int, path string, width int) tea.Cmd {
	return func() tea.Msg {
		return previewDoneMsg{id: id, result: preview.Render(path, width)}
	}
}

func addIgnoreCmd(list *ignore.List, pattern string) tea.Cmd {
	return func() tea.Msg {
		if list == nil {
			list = &ignore.List{}
		}
		if !list.Add(pattern) {
			return ignoreSavedMsg{pattern: pattern, err: fmt.Errorf("already ignored")}
		}
		if err := list.Save(); err != nil {
			return ignoreSavedMsg{pattern: pattern, err: err}
		}
		return ignoreSavedMsg{pattern: pattern}
	}
}

func removeIgnoreCmd(list *ignore.List, index int) tea.Cmd {
	return func() tea.Msg {
		if list == nil || index < 0 || index >= len(list.Patterns) {
			return ignoreSavedMsg{err: fmt.Errorf("nothing to remove")}
		}
		pattern := list.Patterns[index]
		list.RemoveAt(index)
		if err := list.Save(); err != nil {
			return ignoreSavedMsg{pattern: pattern, err: err}
		}
		return ignoreSavedMsg{pattern: "removed " + pattern}
	}
}

func backupCmd(path string, keep int) tea.Cmd {
	return func() tea.Msg {
		snap, err := backup.Create(path, keep)
		return backupDoneMsg{snap: snap, err: err}
	}
}

func restoreCmd(snap backup.Snapshot) tea.Cmd {
	return func() tea.Msg {
		err := backup.Restore(snap)
		return restoreDoneMsg{path: snap.Source, err: err}
	}
}

func pasteCmd(src, dest string) tea.Cmd {
	return func() tea.Msg {
		if err := copyFile(src, dest); err != nil {
			return pasteDoneMsg{path: dest, err: err}
		}
		return pasteDoneMsg{path: dest}
	}
}

func newFileCmd(path string) tea.Cmd {
	return func() tea.Msg {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return newFileDoneMsg{path: path, err: err}
		}
		if _, err := os.Stat(path); err == nil {
			return newFileDoneMsg{path: path, err: fmt.Errorf("already exists")}
		}
		if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
			return newFileDoneMsg{path: path, err: err}
		}
		return newFileDoneMsg{path: path}
	}
}

func symlinkCreateCmd(link, target string) tea.Cmd {
	return func() tea.Msg {
		err := symlink.Create(link, target)
		return symlinkDoneMsg{op: "created", path: link, err: err}
	}
}

func symlinkRetargetCmd(link, target string) tea.Cmd {
	return func() tea.Msg {
		err := symlink.Retarget(link, target)
		return symlinkDoneMsg{op: "retargeted", path: link, err: err}
	}
}

func symlinkRemoveCmd(link string) tea.Cmd {
	return func() tea.Msg {
		err := symlink.Remove(link)
		return symlinkDoneMsg{op: "removed", path: link, err: err}
	}
}

func copyFile(src, dest string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(dest); err == nil {
		return fmt.Errorf("already exists: %s", dest)
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

func (m *Model) rebuildFilter() {
	m.filtered = m.filtered[:0]
	q := normalizeQuery(m.filter)
	if q == "" {
		for i := range m.entries {
			m.filtered = append(m.filtered, i)
		}
		if m.cursor >= len(m.filtered) {
			m.cursor = max(0, len(m.filtered)-1)
		}
		m.ensureVisible()
		return
	}

	type hit struct {
		idx   int
		score int
	}
	hits := make([]hit, 0, len(m.entries))
	for i, e := range m.entries {
		hay := searchHaystack(e.App, e.RelPath, e.AbsPath)
		score := fuzzyScore(q, hay)
		if score < 0 {
			continue
		}
		hits = append(hits, hit{idx: i, score: score})
	}
	sort.SliceStable(hits, func(i, j int) bool {
		if hits[i].score != hits[j].score {
			return hits[i].score > hits[j].score
		}
		return hits[i].idx < hits[j].idx
	})
	for _, h := range hits {
		m.filtered = append(m.filtered, h.idx)
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = max(0, len(m.filtered)-1)
	}
	m.ensureVisible()
}

func (m Model) selected() (scan.Entry, bool) {
	if len(m.filtered) == 0 {
		return scan.Entry{}, false
	}
	if m.cursor < 0 || m.cursor >= len(m.filtered) {
		return scan.Entry{}, false
	}
	return m.entries[m.filtered[m.cursor]], true
}

func (m *Model) ensureVisible() {
	listH := m.listInnerHeight()
	if listH <= 0 {
		return
	}
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+listH {
		m.offset = m.cursor - listH + 1
	}
}

func (m *Model) ensureIgnoreVisible() {
	h := m.listInnerHeight()
	if h <= 0 || m.ignores == nil {
		return
	}
	if m.ignoreCursor < m.ignoreOffset {
		m.ignoreOffset = m.ignoreCursor
	}
	if m.ignoreCursor >= m.ignoreOffset+h {
		m.ignoreOffset = m.ignoreCursor - h + 1
	}
}

func (m *Model) ensureSnapVisible() {
	h := m.listInnerHeight()
	if h <= 0 {
		return
	}
	if m.snapCur < m.snapOff {
		m.snapOff = m.snapCur
	}
	if m.snapCur >= m.snapOff+h {
		m.snapOff = m.snapCur - h + 1
	}
}

func (m Model) listInnerHeight() int {
	_, h := m.contentSize()
	return h
}

func (m Model) contentSize() (w, h int) {
	h = m.height - 1 - 1 - 2
	if m.mode == modeFilter || m.mode == modePaste || m.mode == modeNew ||
		m.mode == modeLinkPath || m.mode == modeLinkTarget || m.mode == modeRetarget {
		h--
	}
	w = m.width
	if h < 1 {
		h = 1
	}
	return w, h
}

func (m Model) paneWidths() (left, right int) {
	w, _ := m.contentSize()
	inner := w
	if inner < 10 {
		inner = 10
	}
	left = inner * 2 / 5
	if left < 24 {
		left = min(24, inner/2)
	}
	right = inner - left
	return left, right
}

func (m Model) contentOrigin() (x, y int) {
	return 2, 2
}

func (m *Model) updateFocusWatch() {
	if m.watcher == nil || !m.cfg.Watch {
		return
	}
	if e, ok := m.selected(); ok {
		m.watcher.SetFocus(filepath.Dir(e.AbsPath))
	}
}

const previewSettle = 70 * time.Millisecond

func (m *Model) requestPreview() tea.Cmd {
	e, ok := m.selected()
	if !ok {
		m.previewPath = ""
		m.previewLang = ""
		m.previewBadge = ""
		m.viewport.SetContent("no selection")
		return nil
	}
	m.updateFocusWatch()
	if e.AbsPath == m.previewPath && m.previewErr == "" && m.viewport.TotalLineCount() > 0 {
		return nil
	}
	_, right := m.paneWidths()
	innerW := max(20, right-6)
	if hit, ok := preview.Lookup(e.AbsPath, innerW); ok {
		m.previewID++
		id := m.previewID
		m.previewPath = e.AbsPath
		return func() tea.Msg {
			return previewDoneMsg{id: id, result: hit}
		}
	}
	m.previewID++
	id := m.previewID
	// Wait until navigation settles so hold-j / paging does not chroma-highlight
	// every file (and so markdown preview never races the TTY for stdin).
	return tea.Tick(previewSettle, func(time.Time) tea.Msg {
		return previewTickMsg{id: id, path: e.AbsPath, width: innerW}
	})
}

func (m Model) moveCursor(delta int) (Model, tea.Cmd) {
	if len(m.filtered) == 0 {
		return m, nil
	}
	m.cursor += delta
	if m.cursor < 0 {
		m.cursor = 0
	}
	if m.cursor >= len(m.filtered) {
		m.cursor = len(m.filtered) - 1
	}
	m.ensureVisible()
	return m, m.requestPreview()
}

func (m Model) moveIgnoreCursor(delta int) Model {
	if m.ignores == nil || len(m.ignores.Patterns) == 0 {
		m.ignoreCursor = 0
		return m
	}
	m.ignoreCursor += delta
	if m.ignoreCursor < 0 {
		m.ignoreCursor = 0
	}
	if m.ignoreCursor >= len(m.ignores.Patterns) {
		m.ignoreCursor = len(m.ignores.Patterns) - 1
	}
	m.ensureIgnoreVisible()
	return m
}

func (m Model) moveSnapCursor(delta int) Model {
	if len(m.snapshots) == 0 {
		m.snapCur = 0
		return m
	}
	m.snapCur += delta
	if m.snapCur < 0 {
		m.snapCur = 0
	}
	if m.snapCur >= len(m.snapshots) {
		m.snapCur = len(m.snapshots) - 1
	}
	m.ensureSnapVisible()
	return m
}

func (m Model) moveStowCursor(delta int) Model {
	if len(m.stowPkgs) == 0 {
		m.stowCur = 0
		return m
	}
	m.stowCur += delta
	if m.stowCur < 0 {
		m.stowCur = 0
	}
	if m.stowCur >= len(m.stowPkgs) {
		m.stowCur = len(m.stowPkgs) - 1
	}
	m.ensureStowVisible()
	return m
}

func (m *Model) ensureStowVisible() {
	h := m.listInnerHeight()
	if h <= 0 {
		return
	}
	if m.stowCur < m.stowOff {
		m.stowOff = m.stowCur
	}
	if m.stowCur >= m.stowOff+h {
		m.stowOff = m.stowCur - h + 1
	}
}

func (m *Model) enterStow() tea.Cmd {
	m.mode = modeStow
	m.focus = focusList
	m.stowCur = 0
	m.stowOff = 0
	m.status = "stow — enter/l link  u unlink  R restow  esc back"
	return refreshStowCmd(m.cfg, m.entries)
}

func (m *Model) enterFilter() tea.Cmd {
	m.mode = modeFilter
	m.focus = focusList
	m.input.Prompt = "/ "
	m.input.Placeholder = "search apps & paths…  (j/k move · enter keep · esc clear)"
	m.input.SetValue(m.filter)
	m.input.CursorEnd()
	m.status = fmt.Sprintf("%d/%d matches", len(m.filtered), len(m.entries))
	return m.input.Focus()
}

func (m *Model) enterPaste() tea.Cmd {
	if m.yankPath == "" {
		m.status = "nothing yanked — press y on a file first"
		return nil
	}
	m.mode = modePaste
	base := filepath.Base(m.yankPath)
	ext := filepath.Ext(base)
	stem := strings.TrimSuffix(base, ext)
	def := stem + ".copy" + ext
	m.input.Prompt = "paste as: "
	m.input.Placeholder = def
	m.input.SetValue(def)
	m.input.CursorEnd()
	m.status = "paste into " + filepath.Dir(m.yankPath)
	return m.input.Focus()
}

func (m *Model) enterNew() tea.Cmd {
	m.mode = modeNew
	def := "app/config.toml"
	if e, ok := m.selected(); ok {
		switch e.App {
		case "home":
			def = e.RelPath + ".new"
		case "config":
			def = e.RelPath
		default:
			def = e.App + "/new.toml"
		}
	}
	m.input.Prompt = "new: "
	m.input.Placeholder = "app/file.toml or ~/.zshrc"
	m.input.SetValue(def)
	m.input.CursorEnd()
	m.status = "path under ~/.config, or ~/./home file, or absolute"
	return m.input.Focus()
}

func (m *Model) enterLinkPath() tea.Cmd {
	m.mode = modeLinkPath
	m.linkPath = ""
	def := "app/link"
	if e, ok := m.selected(); ok {
		switch e.App {
		case "home":
			def = e.RelPath + ".link"
		case "config":
			def = e.RelPath
		default:
			def = e.App + "/link"
		}
	}
	m.input.Prompt = "link: "
	m.input.Placeholder = "path for new symlink"
	m.input.SetValue(def)
	m.input.CursorEnd()
	m.status = "where the symlink will live"
	return m.input.Focus()
}

func (m *Model) enterLinkTarget(linkPath string) tea.Cmd {
	m.mode = modeLinkTarget
	m.linkPath = linkPath
	def := m.yankPath
	if def == "" {
		if e, ok := m.selected(); ok {
			def = e.AbsPath
		}
	}
	m.input.Prompt = "→ "
	m.input.Placeholder = "target path (absolute, relative, or ~/…)"
	m.input.SetValue(def)
	m.input.CursorEnd()
	m.status = "target for " + linkPath
	return m.input.Focus()
}

func (m *Model) enterRetarget() tea.Cmd {
	e, ok := m.selected()
	if !ok {
		m.status = "no selection"
		return nil
	}
	info, err := symlink.Read(e.AbsPath)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	m.mode = modeRetarget
	m.linkPath = e.AbsPath
	m.input.Prompt = "→ "
	m.input.Placeholder = "new symlink target"
	m.input.SetValue(info.Target)
	m.input.CursorEnd()
	m.status = "retarget " + e.AbsPath
	return m.input.Focus()
}

func (m *Model) enterRestore() tea.Cmd {
	e, ok := m.selected()
	if !ok {
		m.status = "no selection"
		return nil
	}
	snaps, err := backup.List(e.AbsPath)
	if err != nil {
		m.status = err.Error()
		return nil
	}
	m.snapshots = snaps
	m.snapCur = 0
	m.snapOff = 0
	m.mode = modeRestore
	if len(snaps) == 0 {
		m.status = "no backups for " + filepath.Base(e.AbsPath)
	} else {
		m.status = fmt.Sprintf("%d backups — enter restore  esc back", len(snaps))
	}
	return nil
}

func (m *Model) askConfirm(kind confirmKind, path, msg string) {
	m.mode = modeConfirm
	m.confirm = kind
	m.confirmPath = path
	m.confirmPkg = ""
	m.status = msg
}

func (m *Model) askConfirmUnstow(pkg, msg string) {
	m.mode = modeConfirm
	m.confirm = confirmUnstow
	m.confirmPath = pkg
	m.confirmPkg = pkg
	m.status = msg
}

func resolveNewPath(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty path")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}

	if strings.HasPrefix(raw, "~/") {
		return filepath.Join(home, raw[2:]), nil
	}
	if strings.HasPrefix(raw, "/") {
		return raw, nil
	}
	if strings.HasPrefix(raw, "./") || (strings.HasPrefix(raw, ".") && !strings.Contains(raw[1:], "/")) {
		name := strings.TrimPrefix(raw, "./")
		return filepath.Join(home, name), nil
	}
	return filepath.Join(configHome, raw), nil
}

func isSecret(path string) bool {
	return secret.Path(path)
}

// samePath reports whether two paths refer to the same file.
func samePath(a, b string) bool {
	if a == "" || b == "" {
		return false
	}
	if a == b {
		return true
	}
	ra, err1 := filepath.EvalSymlinks(a)
	rb, err2 := filepath.EvalSymlinks(b)
	if err1 == nil && err2 == nil {
		return ra == rb
	}
	return false
}

// delayedRescan avoids stampeding on bursty editor writes.
func delayedRescan() tea.Cmd {
	return tea.Tick(400*time.Millisecond, func(time.Time) tea.Msg {
		return rescanTickMsg{}
	})
}
