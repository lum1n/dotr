package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/lum1n/dotr/internal/config"
	"github.com/lum1n/dotr/internal/gitstatus"
	"github.com/lum1n/dotr/internal/ignore"
	"github.com/lum1n/dotr/internal/preview"
	"github.com/lum1n/dotr/internal/scan"
	"github.com/lum1n/dotr/internal/stow"
	"github.com/lum1n/dotr/internal/symlink"
)

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.ready = true
		m.applySizes()
		m.ensureVisible()
		if m.mode == modeBrowse && m.previewPath == "" {
			cmds = append(cmds, m.requestPreview())
		}

	case scanDoneMsg:
		m.scanning = false
		if msg.err != nil {
			m.err = msg.err
			m.status = msg.err.Error()
			break
		}
		m.cfg = msg.cfg
		m.entries = msg.entries
		m.truncated = msg.truncated
		if msg.ignores != nil {
			m.ignores = msg.ignores
		}
		if msg.git != nil {
			m.git = msg.git
		} else if !m.cfg.GitStatus {
			m.git = map[string]gitstatus.Kind{}
		}
		m.rebuildFilter()
		ignN := 0
		if m.ignores != nil {
			ignN = len(m.ignores.Patterns)
		}
		m.status = fmt.Sprintf("%d configs · %d ignores", len(m.entries), ignN)
		if m.truncated {
			m.status += fmt.Sprintf(" · truncated at %d", scan.MaxFiles())
		}
		if !m.cfg.Watch && m.watcher != nil {
			_ = m.watcher.Close()
			m.watcher = nil
		}
		m.updateFocusWatch()
		cmds = append(cmds, refreshStowCmd(m.cfg, m.entries))
		if m.cfg.GitStatus {
			paths := make([]string, len(m.entries))
			for i, e := range m.entries {
				paths[i] = e.AbsPath
			}
			cmds = append(cmds, refreshGitCmd(paths))
		}
		if m.mode == modeBrowse || m.mode == modeFilter {
			m.previewPath = ""
			cmds = append(cmds, m.requestPreview())
		}

	case watcherReadyMsg:
		if msg.w != nil && m.cfg.Watch {
			m.watcher = msg.w
			m.updateFocusWatch()
			cmds = append(cmds, waitWatcherCmd(m.watcher))
		} else if msg.w != nil {
			_ = msg.w.Close()
		}

	case fsEventMsg:
		if !m.cfg.Watch {
			break
		}
		cmds = append(cmds, waitWatcherCmd(m.watcher))
		// Preview only — list rescans are manual (`r`), after $EDITOR, or
		// after mutating actions. Watcher noise (and leaked rgb: keystrokes)
		// used to stampede a full scan.
		if e, ok := m.selected(); ok && samePath(msg.path, e.AbsPath) && msg.op == "write" {
			preview.Invalidate(e.AbsPath)
			m.previewPath = ""
			cmds = append(cmds, m.requestPreview())
			if m.cfg.GitStatus {
				cmds = append(cmds, refreshGitCmd([]string{e.AbsPath}))
			}
			m.status = "reloaded " + filepath.Base(msg.path)
		}

	case rescanTickMsg:
		m.scanning = true
		cmds = append(cmds, scanCmd())

	case gitDoneMsg:
		if msg.git == nil {
			break
		}
		if m.git == nil {
			m.git = msg.git
		} else {
			for k, v := range msg.git {
				m.git[k] = v
			}
		}

	case stowDoneMsg:
		if msg.err != nil {
			if m.mode == modeStow {
				m.status = "stow: " + msg.err.Error()
			}
			break
		}
		m.stowOpts = msg.opts
		m.stowPkgs = msg.pkgs
		if msg.owners != nil {
			m.stowOwners = msg.owners
		}
		if m.stowCur >= len(m.stowPkgs) {
			m.stowCur = max(0, len(m.stowPkgs)-1)
		}
		m.ensureStowVisible()
		if m.mode == modeStow {
			m.status = fmt.Sprintf("%d packages · %s → %s",
				len(m.stowPkgs), m.stowOpts.Dir, m.stowOpts.Target)
		}

	case stowOpDoneMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
		} else {
			m.status = fmt.Sprintf("%s %s", msg.action, strings.Join(msg.pkgs, " "))
			if msg.out != "" {
				// keep status short; full stow verbose stays in CLI
				lines := strings.Split(msg.out, "\n")
				m.status = fmt.Sprintf("%s · %d lines", m.status, len(lines))
			}
		}
		cmds = append(cmds, refreshStowCmd(m.cfg, m.entries), delayedRescan())

	case previewTickMsg:
		if msg.id != m.previewID {
			break
		}
		e, ok := m.selected()
		if !ok || e.AbsPath != msg.path {
			break
		}
		m.previewPath = msg.path
		cmds = append(cmds, previewCmd(msg.id, msg.path, msg.width))

	case previewDoneMsg:
		if msg.id != m.previewID {
			break
		}
		r := msg.result
		m.previewLang = r.Language
		m.previewBadge = r.Badge()
		m.previewParse = r.Parse
		if r.Err != nil {
			m.previewErr = r.Err.Error()
			m.viewport.SetContent(r.Content)
			m.status = r.Err.Error()
			break
		}
		m.previewErr = ""
		content := r.Content
		if isSecret(r.Path) {
			warn := m.styles.secret.Render("⚠ secret-looking file — yank contents requires confirm")
			content = warn + "\n\n" + content
		}
		if r.Parse == preview.ParseFail && r.ParseErr != "" {
			content = m.styles.parseFail.Render("parse error: "+r.ParseErr) + "\n\n" + content
		}
		m.viewport.SetContent(content)
		m.viewport.GotoTop()
		label := filepath.Base(r.Path)
		if isSecret(r.Path) {
			label = "🔒 " + label
		}
		if r.Language != "" {
			label += " · " + r.Language
		}
		if badge := r.Badge(); badge != "" {
			label += " " + badge
		}
		if r.Binary {
			label += " · binary"
		}
		if r.Truncated {
			label += " · truncated"
		}
		m.status = label

	case ignoreSavedMsg:
		if msg.err != nil {
			m.status = msg.err.Error()
			break
		}
		m.status = msg.pattern
		if !strings.HasPrefix(msg.pattern, "removed ") {
			m.status = "ignored " + msg.pattern
		}
		if m.mode == modeIgnores {
			if m.ignores != nil && m.ignoreCursor >= len(m.ignores.Patterns) {
				m.ignoreCursor = max(0, len(m.ignores.Patterns)-1)
			}
			m.ensureIgnoreVisible()
		}
		m.scanning = true
		cmds = append(cmds, scanCmd())

	case backupDoneMsg:
		if msg.err != nil {
			m.status = "backup: " + msg.err.Error()
			break
		}
		m.status = "backed up " + filepath.Base(msg.snap.Source)

	case restoreDoneMsg:
		if msg.err != nil {
			m.status = "restore: " + msg.err.Error()
			break
		}
		m.mode = modeBrowse
		m.status = "restored " + filepath.Base(msg.path)
		m.previewPath = ""
		cmds = append(cmds, m.requestPreview())

	case pasteDoneMsg:
		if msg.err != nil {
			m.status = "paste: " + msg.err.Error()
			m.mode = modeBrowse
			m.input.Blur()
			break
		}
		m.mode = modeBrowse
		m.input.Blur()
		m.status = "pasted " + msg.path
		m.scanning = true
		cmds = append(cmds, scanCmd())

	case newFileDoneMsg:
		if msg.err != nil {
			m.status = "new: " + msg.err.Error()
			m.mode = modeBrowse
			m.input.Blur()
			break
		}
		m.mode = modeBrowse
		m.input.Blur()
		m.status = "created " + msg.path
		return m, tea.Batch(openEditor(msg.path), scanCmd())

	case symlinkDoneMsg:
		m.mode = modeBrowse
		m.input.Blur()
		m.linkPath = ""
		if msg.err != nil {
			m.status = "symlink: " + msg.err.Error()
			break
		}
		m.status = msg.op + " " + msg.path
		m.scanning = true
		cmds = append(cmds, scanCmd())

	case editorFinishedMsg:
		if msg.err != nil {
			m.status = "editor: " + msg.err.Error()
			break
		}
		m.status = "edited " + filepath.Base(msg.path)
		preview.Invalidate(msg.path)
		if ignPath, err := ignore.Path(); err == nil && msg.path == ignPath {
			m.scanning = true
			return m, scanCmd()
		}
		if cfgPath, err := config.Path(); err == nil && msg.path == cfgPath {
			m.scanning = true
			return m, scanCmd()
		}
		m.previewPath = ""
		m.scanning = true
		cmds = append(cmds, scanCmd())

	case tea.MouseClickMsg:
		if m.cfg.Mouse && m.mode == modeBrowse {
			var cmd tea.Cmd
			m, cmd = m.handleMouseClick(msg)
			cmds = append(cmds, cmd)
		}

	case tea.MouseWheelMsg:
		if m.cfg.Mouse {
			var cmd tea.Cmd
			m, cmd = m.handleMouseWheel(msg)
			cmds = append(cmds, cmd)
		}

	case tea.PasteMsg:
		if looksLikeTermReply(msg.Content) || looksLikeTermReply(strings.TrimSpace(msg.Content)) {
			break
		}
		if m.mode == modeFilter {
			m.input.SetValue(m.input.Value() + msg.Content)
			m.filter = m.input.Value()
			m.rebuildFilter()
			if isFilterJunk(m.filter) {
				return m.abortJunkFilter()
			}
		}

	case junkTickMsg:
		if msg.id != m.junkID {
			break
		}
		var cmd tea.Cmd
		m, cmd = m.flushJunkTimeout(msg.buf)
		cmds = append(cmds, cmd)

	case tea.KeyPressMsg:
		switch m.mode {
		case modeIgnores:
			return m.updateIgnoresKey(msg)
		case modeFilter:
			return m.updateFilterKey(msg)
		case modePaste:
			return m.updatePasteKey(msg)
		case modeNew:
			return m.updateNewKey(msg)
		case modeLinkPath:
			return m.updateLinkPathKey(msg)
		case modeLinkTarget:
			return m.updateLinkTargetKey(msg)
		case modeRetarget:
			return m.updateRetargetKey(msg)
		case modeRestore:
			return m.updateRestoreKey(msg)
		case modeHelp:
			return m.updateHelpKey(msg)
		case modeConfirm:
			return m.updateConfirmKey(msg)
		case modeStow:
			return m.updateStowKey(msg)
		default:
			var cmd tea.Cmd
			m, cmd = m.ingestBrowseKey(msg)
			cmds = append(cmds, cmd)
		}
	}

	if m.mode == modeBrowse && m.focus == focusPreview {
		var cmd tea.Cmd
		m.viewport, cmd = m.viewport.Update(msg)
		cmds = append(cmds, cmd)
	}

	return m, tea.Batch(cmds...)
}

func (m Model) handleMouseClick(msg tea.MouseClickMsg) (Model, tea.Cmd) {
	if msg.Button != tea.MouseLeft {
		return m, nil
	}
	mx, my := msg.X, msg.Y
	leftW, _ := m.paneWidths()
	ox, oy := m.contentOrigin()
	listH := m.listInnerHeight()

	// Click in left panel content?
	if mx < leftW && my >= oy && my < oy+listH {
		m.focus = focusList
		row := m.offset + (my - oy)
		if row >= 0 && row < len(m.filtered) {
			m.cursor = row
			m.ensureVisible()
			return m, m.requestPreview()
		}
		return m, nil
	}
	// Right panel → focus preview
	if mx >= leftW {
		m.focus = focusPreview
		_ = ox
	}
	return m, nil
}

func (m Model) handleMouseWheel(msg tea.MouseWheelMsg) (Model, tea.Cmd) {
	mx := msg.Mouse().X
	leftW, _ := m.paneWidths()
	delta := 0
	switch msg.Button {
	case tea.MouseWheelUp:
		delta = -3
	case tea.MouseWheelDown:
		delta = 3
	default:
		return m, nil
	}

	if m.mode != modeBrowse {
		return m, nil
	}

	if mx < leftW || m.focus == focusList {
		m.focus = focusList
		return m.moveCursor(delta)
	}
	m.focus = focusPreview
	if delta < 0 {
		m.viewport.ScrollUp(-delta)
	} else {
		m.viewport.ScrollDown(delta)
	}
	return m, nil
}

func (m Model) updateBrowseKey(msg tea.KeyPressMsg) (Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.watcher != nil {
			_ = m.watcher.Close()
			m.watcher = nil
		}
		return m, tea.Quit

	case "?":
		m.mode = modeHelp
		return m, nil

	case ",":
		path, err := config.EnsureFile()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.status = "editing dotr config…"
		return m, openEditor(path)

	case "/", "ctrl+f":
		return m, m.enterFilter()

	case "n":
		return m, m.enterNew()

	case "l":
		return m, m.enterLinkPath()

	case "t":
		return m, m.enterRetarget()

	case "D":
		e, ok := m.selected()
		if !ok {
			return m, nil
		}
		if !symlink.Is(e.AbsPath) {
			m.status = "not a symlink"
			return m, nil
		}
		info, err := symlink.Read(e.AbsPath)
		target := "?"
		if err == nil {
			target = info.Target
		}
		m.askConfirm(confirmDeleteSymlink, e.AbsPath,
			fmt.Sprintf("⚠ delete symlink (keeps target %s)?", target))
		return m, nil

	case "tab":
		if m.focus == focusList {
			m.focus = focusPreview
		} else {
			m.focus = focusList
		}

	case "e":
		if e, ok := m.selected(); ok {
			m.status = "opening $EDITOR…"
			return m, openEditor(e.AbsPath)
		}

	case "y":
		e, ok := m.selected()
		if !ok {
			return m, nil
		}
		m.yankPath = e.AbsPath
		m.status = "yanked path " + e.AbsPath
		return m, tea.SetClipboard(e.AbsPath)

	case "Y":
		e, ok := m.selected()
		if !ok {
			return m, nil
		}
		if m.cfg.ConfirmSecrets && isSecret(e.AbsPath) {
			m.askConfirm(confirmYankContents, e.AbsPath,
				"⚠ secret file — enter to yank contents, esc cancel")
			return m, nil
		}
		return m.doYankContents(e.AbsPath)

	case "p":
		return m, m.enterPaste()

	case "b":
		e, ok := m.selected()
		if !ok {
			return m, nil
		}
		if m.cfg.ConfirmSecrets && isSecret(e.AbsPath) {
			m.askConfirm(confirmBackup, e.AbsPath,
				"⚠ secret file — enter to backup, esc cancel")
			return m, nil
		}
		m.status = "backing up…"
		return m, backupCmd(e.AbsPath, m.cfg.BackupKeep)

	case "R":
		return m, m.enterRestore()

	case "i":
		e, ok := m.selected()
		if !ok {
			return m, nil
		}
		pattern := ignore.SuggestForEntry(e.App, e.RelPath)
		m.status = "ignoring " + pattern + "…"
		return m, addIgnoreCmd(m.ignores, pattern)

	case "I":
		e, ok := m.selected()
		if !ok {
			return m, nil
		}
		pattern := ignore.SuggestApp(e.App, e.RelPath)
		m.status = "ignoring " + pattern + "…"
		return m, addIgnoreCmd(m.ignores, pattern)

	case "x":
		m.mode = modeIgnores
		m.focus = focusList
		m.ignoreCursor = 0
		m.ignoreOffset = 0
		m.status = "ignore list — d delete  e edit file  esc back"
		return m, nil

	case "s":
		return m, m.enterStow()

	case "r":
		m.scanning = true
		m.status = "rescanning…"
		return m, scanCmd()

	case "esc":
		if m.filter != "" {
			m.filter = ""
			m.rebuildFilter()
			m.status = "search cleared"
			return m, m.requestPreview()
		}

	case "g":
		if m.focus == focusPreview {
			m.viewport.GotoTop()
		} else {
			return m.moveCursor(-m.cursor)
		}

	case "G":
		if m.focus == focusPreview {
			m.viewport.GotoBottom()
		} else if len(m.filtered) > 0 {
			return m.moveCursor(len(m.filtered) - 1 - m.cursor)
		}

	case "j", "down":
		if m.focus == focusPreview {
			m.viewport.ScrollDown(1)
		} else {
			return m.moveCursor(1)
		}

	case "k", "up":
		if m.focus == focusPreview {
			m.viewport.ScrollUp(1)
		} else {
			return m.moveCursor(-1)
		}

	case "ctrl+d":
		if m.focus == focusPreview {
			m.viewport.HalfPageDown()
		} else {
			return m.moveCursor(m.listInnerHeight() / 2)
		}

	case "ctrl+u":
		if m.focus == focusPreview {
			m.viewport.HalfPageUp()
		} else {
			return m.moveCursor(-(m.listInnerHeight() / 2))
		}

	case "pgdown":
		if m.focus == focusPreview {
			m.viewport.PageDown()
		} else {
			return m.moveCursor(m.listInnerHeight())
		}

	case "pgup":
		if m.focus == focusPreview {
			m.viewport.PageUp()
		} else {
			return m.moveCursor(-m.listInnerHeight())
		}
	}
	return m, nil
}

func (m Model) doYankContents(path string) (Model, tea.Cmd) {
	data, err := os.ReadFile(path)
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.yankPath = path
	m.status = fmt.Sprintf("yanked contents (%dB)", len(data))
	return m, tea.SetClipboard(string(data))
}

func (m Model) updateConfirmKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "n", "q":
		wasUnstow := m.confirm == confirmUnstow
		m.confirm = confirmNone
		m.confirmPkg = ""
		if wasUnstow {
			m.mode = modeStow
			m.status = "unstow cancelled"
			return m, nil
		}
		m.mode = modeBrowse
		m.status = "cancelled"
		return m, nil
	case "enter", "y":
		kind := m.confirm
		path := m.confirmPath
		pkg := m.confirmPkg
		m.mode = modeBrowse
		m.confirm = confirmNone
		m.confirmPkg = ""
		switch kind {
		case confirmYankContents:
			return m.doYankContents(path)
		case confirmBackup:
			m.status = "backing up…"
			return m, backupCmd(path, m.cfg.BackupKeep)
		case confirmUnstow:
			m.mode = modeStow
			m.status = fmt.Sprintf("unstow %s…", pkg)
			return m, stowOpCmd(m.stowOpts, stow.ActionUnstow, []string{pkg})
		case confirmDeleteSymlink:
			m.status = "removing symlink…"
			return m, symlinkRemoveCmd(path)
		}
	}
	return m, nil
}

func (m Model) updateFilterKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.mode = modeBrowse
		m.input.Blur()
		m.filter = ""
		m.input.SetValue("")
		m.rebuildFilter()
		m.status = "search cleared"
		return m, m.requestPreview()

	case "ctrl+c":
		if m.watcher != nil {
			_ = m.watcher.Close()
			m.watcher = nil
		}
		return m, tea.Quit

	case "enter":
		m.mode = modeBrowse
		m.input.Blur()
		m.filter = m.input.Value()
		m.rebuildFilter()
		if m.filter == "" {
			m.status = "search cleared"
		} else {
			m.status = fmt.Sprintf("search %q · %d matches", m.filter, len(m.filtered))
		}
		return m, m.requestPreview()

	case "ctrl+w":
		m.input.SetValue("")
		m.filter = ""
		m.rebuildFilter()
		m.status = fmt.Sprintf("%d/%d matches", len(m.filtered), len(m.entries))
		return m, m.requestPreview()

	case "down", "ctrl+n", "ctrl+j":
		return m.moveCursor(1)

	case "up", "ctrl+p", "ctrl+k":
		return m.moveCursor(-1)

	case "tab":
		// Keep typing; don't steal tab into the textinput as a character if possible.
		if m.focus == focusList {
			m.focus = focusPreview
		} else {
			m.focus = focusList
		}
		return m, nil
	}

	// j/k navigate when the input is empty; otherwise type into the query.
	if msg.String() == "j" && m.input.Value() == "" {
		return m.moveCursor(1)
	}
	if msg.String() == "k" && m.input.Value() == "" {
		return m.moveCursor(-1)
	}

	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	m.filter = m.input.Value()
	if isFilterJunk(m.filter) {
		return m.abortJunkFilter()
	}
	m.rebuildFilter()
	m.status = fmt.Sprintf("%d/%d matches", len(m.filtered), len(m.entries))
	return m, tea.Batch(cmd, m.requestPreview())
}

func (m Model) updatePasteKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBrowse
		m.input.Blur()
		m.status = "paste cancelled"
		return m, nil

	case "enter":
		name := strings.TrimSpace(m.input.Value())
		if name == "" {
			m.status = "empty name"
			return m, nil
		}
		if strings.Contains(name, string(os.PathSeparator)) {
			m.status = "name must be a single filename"
			return m, nil
		}
		dest := filepath.Join(filepath.Dir(m.yankPath), name)
		m.input.Blur()
		m.status = "pasting…"
		return m, pasteCmd(m.yankPath, dest)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateNewKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBrowse
		m.input.Blur()
		m.status = "new cancelled"
		return m, nil

	case "enter":
		path, err := resolveNewPath(m.input.Value())
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		m.input.Blur()
		m.status = "creating " + path
		return m, newFileCmd(path)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateLinkPathKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBrowse
		m.input.Blur()
		m.linkPath = ""
		m.status = "link cancelled"
		return m, nil

	case "enter":
		path, err := resolveNewPath(m.input.Value())
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		return m, m.enterLinkTarget(path)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateLinkTargetKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBrowse
		m.input.Blur()
		m.linkPath = ""
		m.status = "link cancelled"
		return m, nil

	case "enter":
		target, err := resolveSymlinkTarget(m.input.Value())
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		link := m.linkPath
		m.input.Blur()
		m.status = "linking…"
		return m, symlinkCreateCmd(link, target)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func (m Model) updateRetargetKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "ctrl+c":
		m.mode = modeBrowse
		m.input.Blur()
		m.linkPath = ""
		m.status = "retarget cancelled"
		return m, nil

	case "enter":
		target, err := resolveSymlinkTarget(m.input.Value())
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		link := m.linkPath
		m.input.Blur()
		m.status = "retargeting…"
		return m, symlinkRetargetCmd(link, target)
	}
	var cmd tea.Cmd
	m.input, cmd = m.input.Update(msg)
	return m, cmd
}

func resolveSymlinkTarget(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("empty target")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	// Keep relative targets as-is (stow-style); only expand ~/ and leave
	// other relative paths untouched for os.Symlink.
	if strings.HasPrefix(raw, "~/") || raw == "~" {
		return symlink.ExpandTilde(raw, home), nil
	}
	return raw, nil
}

func (m Model) updateRestoreKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "q", "R":
		m.mode = modeBrowse
		m.status = "back to configs"
		return m, m.requestPreview()

	case "j", "down":
		return m.moveSnapCursor(1), nil

	case "k", "up":
		return m.moveSnapCursor(-1), nil

	case "g":
		m.snapCur = 0
		m.ensureSnapVisible()
		return m, nil

	case "G":
		if len(m.snapshots) > 0 {
			m.snapCur = len(m.snapshots) - 1
			m.ensureSnapVisible()
		}
		return m, nil

	case "enter":
		if len(m.snapshots) == 0 {
			m.status = "no backups"
			return m, nil
		}
		snap := m.snapshots[m.snapCur]
		m.status = "restoring…"
		return m, restoreCmd(snap)
	}
	return m, nil
}

func (m Model) updateHelpKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c":
		return m, tea.Quit
	default:
		m.mode = modeBrowse
		m.status = ""
		return m, nil
	}
}

func (m Model) updateIgnoresKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		return m, tea.Quit

	case "esc", "x":
		m.mode = modeBrowse
		m.status = "back to configs"
		return m, m.requestPreview()

	case "j", "down":
		return m.moveIgnoreCursor(1), nil

	case "k", "up":
		return m.moveIgnoreCursor(-1), nil

	case "g":
		m.ignoreCursor = 0
		m.ensureIgnoreVisible()
		return m, nil

	case "G":
		if m.ignores != nil && len(m.ignores.Patterns) > 0 {
			m.ignoreCursor = len(m.ignores.Patterns) - 1
			m.ensureIgnoreVisible()
		}
		return m, nil

	case "d", "delete", "backspace", "enter":
		if m.ignores == nil || len(m.ignores.Patterns) == 0 {
			m.status = "no ignores"
			return m, nil
		}
		return m, removeIgnoreCmd(m.ignores, m.ignoreCursor)

	case "e":
		path, err := ignore.Path()
		if err != nil {
			m.status = err.Error()
			return m, nil
		}
		if m.ignores != nil {
			_ = m.ignores.Save()
		}
		m.status = "editing ignore file…"
		return m, openEditor(path)
	}
	return m, nil
}

func (m Model) updateStowKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "ctrl+c", "q":
		if m.watcher != nil {
			_ = m.watcher.Close()
			m.watcher = nil
		}
		return m, tea.Quit

	case "esc", "s":
		m.mode = modeBrowse
		m.status = "back to configs"
		return m, m.requestPreview()

	case "j", "down":
		return m.moveStowCursor(1), nil

	case "k", "up":
		return m.moveStowCursor(-1), nil

	case "g":
		m.stowCur = 0
		m.ensureStowVisible()
		return m, nil

	case "G":
		if len(m.stowPkgs) > 0 {
			m.stowCur = len(m.stowPkgs) - 1
			m.ensureStowVisible()
		}
		return m, nil

	case "r":
		m.status = "refreshing stow…"
		return m, refreshStowCmd(m.cfg, m.entries)

	case "enter", "l":
		return m.runSelectedStow(stow.ActionStow)

	case "u":
		if len(m.stowPkgs) == 0 {
			m.status = "no packages"
			return m, nil
		}
		pkg := m.stowPkgs[m.stowCur].Name
		m.askConfirmUnstow(pkg, fmt.Sprintf("⚠ unstow %q — enter confirm, esc cancel", pkg))
		return m, nil

	case "R":
		return m.runSelectedStow(stow.ActionRestow)
	}
	return m, nil
}

func (m Model) runSelectedStow(action stow.Action) (Model, tea.Cmd) {
	if len(m.stowPkgs) == 0 {
		m.status = "no packages"
		return m, nil
	}
	if m.stowOpts.Dir == "" {
		m.status = "stow dir unknown"
		return m, nil
	}
	pkg := m.stowPkgs[m.stowCur]
	m.status = fmt.Sprintf("%s %s…", action, pkg.Name)
	return m, stowOpCmd(m.stowOpts, action, []string{pkg.Name})
}

func (m *Model) applySizes() {
	_, right := m.paneWidths()
	_, h := m.contentSize()
	innerW := max(1, right-4)
	innerH := max(1, h-1)
	m.viewport.SetWidth(innerW)
	m.viewport.SetHeight(innerH)
	m.input.SetWidth(max(10, m.width-4))
}
