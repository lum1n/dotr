// Package stow is a thin wrapper around GNU Stow for dotr.
package stow

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// Options are the stow directory and target tree.
type Options struct {
	Dir    string
	Target string
	// Only, when non-empty, limits package listing (used when --dir pointed
	// at a package directory and we rewrote Dir to its parent).
	Only []string
}

// Action is a stow operation.
type Action string

const (
	ActionStow   Action = "stow"
	ActionUnstow Action = "unstow"
	ActionRestow Action = "restow"
)

// LinkState is the status of one expected target path.
type LinkState int

const (
	LinkLinked LinkState = iota
	LinkMissing
	LinkConflict
)

func (s LinkState) String() string {
	switch s {
	case LinkLinked:
		return "linked"
	case LinkMissing:
		return "missing"
	case LinkConflict:
		return "conflict"
	default:
		return "?"
	}
}

// Link is one path that stow would manage for a package.
type Link struct {
	Rel    string // relative to target
	Target string // absolute path in target tree
	Source string // absolute path in package
	State  LinkState
}

// PkgStatus is the aggregate status of a package.
type PkgStatus int

const (
	StatusEmpty PkgStatus = iota
	StatusLinked
	StatusPartial
	StatusUnlinked
	StatusConflict
)

func (s PkgStatus) String() string {
	switch s {
	case StatusEmpty:
		return "empty"
	case StatusLinked:
		return "linked"
	case StatusPartial:
		return "partial"
	case StatusUnlinked:
		return "unlinked"
	case StatusConflict:
		return "conflict"
	default:
		return "?"
	}
}

// Mark is a short status glyph for TUI/CLI.
func (s PkgStatus) Mark() string {
	switch s {
	case StatusLinked:
		return "✓"
	case StatusPartial:
		return "±"
	case StatusUnlinked:
		return "·"
	case StatusConflict:
		return "!"
	default:
		return " "
	}
}

// Package is a stow package with analyzed link status.
type Package struct {
	Name      string
	Path      string
	Status    PkgStatus
	Links     []Link
	Linked    int
	Missing   int
	Conflicts int
}

var skipPackageNames = map[string]struct{}{
	".git": {}, ".github": {},
}

// Resolve builds Options from optional config overrides and ~/.stowrc.
// When the configured dir looks like a package (common: --dir=…/dotfiles),
// but links match parent-dir + basename(package), Dir is rewritten to the parent.
func Resolve(stowDir, stowTarget string) (Options, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Options{}, err
	}

	opts := Options{}
	if rc, err := ReadRC(filepath.Join(home, ".stowrc")); err == nil {
		opts = rc
	}
	if stowDir != "" {
		opts.Dir = stowDir
	}
	if stowTarget != "" {
		opts.Target = stowTarget
	}
	if opts.Target == "" {
		opts.Target = home
	}
	if opts.Dir == "" {
		for _, cand := range []string{
			filepath.Join(home, "repos", "dotfiles"),
			filepath.Join(home, "dotfiles"),
			filepath.Join(home, ".dotfiles"),
		} {
			if st, err := os.Stat(cand); err == nil && st.IsDir() {
				opts.Dir = cand
				break
			}
		}
	}
	opts.Dir = expandTilde(opts.Dir, home)
	opts.Target = expandTilde(opts.Target, home)
	if opts.Dir == "" {
		return opts, fmt.Errorf("no stow dir — set stow_dir in config or --dir in ~/.stowrc")
	}
	opts.Dir = filepath.Clean(opts.Dir)
	opts.Target = filepath.Clean(opts.Target)
	return normalize(opts), nil
}

// ReadRC parses a .stowrc for --dir and --target.
func ReadRC(path string) (Options, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Options{}, err
	}
	var opts Options
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, val, ok := strings.Cut(line, "=")
		if !ok {
			fields := strings.Fields(line)
			if len(fields) >= 2 {
				key, val = fields[0], fields[1]
			} else {
				continue
			}
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "--dir", "-d":
			opts.Dir = val
		case "--target", "-t":
			opts.Target = val
		}
	}
	return opts, sc.Err()
}

func expandTilde(p, home string) string {
	if p == "~" {
		return home
	}
	if strings.HasPrefix(p, "~/") {
		return filepath.Join(home, p[2:])
	}
	return p
}

// normalize rewrites --dir=…/dotfiles to --dir=… package=dotfiles when that matches installed links.
func normalize(opts Options) Options {
	names, err := packageNames(opts.Dir)
	if err != nil || len(names) == 0 {
		return opts
	}
	linked := 0
	for _, name := range names {
		pkg, err := Analyze(opts, name)
		if err != nil {
			continue
		}
		linked += pkg.Linked
	}
	if linked > 0 {
		return opts
	}
	parent := filepath.Dir(opts.Dir)
	name := filepath.Base(opts.Dir)
	if parent == "" || parent == opts.Dir || name == "." || name == "/" {
		return opts
	}
	alt := Options{Dir: parent, Target: opts.Target, Only: []string{name}}
	pkg, err := Analyze(alt, name)
	if err != nil || pkg.Linked == 0 {
		return opts
	}
	return alt
}

// Packages lists package names under the stow dir.
func Packages(opts Options) ([]string, error) {
	names, err := packageNames(opts.Dir)
	if err != nil {
		return nil, err
	}
	if len(opts.Only) == 0 {
		return names, nil
	}
	allow := make(map[string]struct{}, len(opts.Only))
	for _, n := range opts.Only {
		allow[n] = struct{}{}
	}
	var out []string
	for _, n := range names {
		if _, ok := allow[n]; ok {
			out = append(out, n)
		}
	}
	// Prefer declared Only even if the directory listing missed it.
	if len(out) == 0 {
		return append([]string{}, opts.Only...), nil
	}
	return out, nil
}

func packageNames(dir string) ([]string, error) {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var out []string
	for _, e := range ents {
		name := e.Name()
		if !e.IsDir() {
			info, err := os.Stat(filepath.Join(dir, name))
			if err != nil || !info.IsDir() {
				continue
			}
		}
		if _, skip := skipPackageNames[name]; skip {
			continue
		}
		// Allow .config as a package; skip other hidden dirs.
		if strings.HasPrefix(name, ".") && name != ".config" {
			continue
		}
		out = append(out, name)
	}
	return out, nil
}

// AnalyzeAll returns status for every package.
func AnalyzeAll(opts Options) ([]Package, error) {
	names, err := Packages(opts)
	if err != nil {
		return nil, err
	}
	out := make([]Package, 0, len(names))
	for _, name := range names {
		pkg, err := Analyze(opts, name)
		if err != nil {
			return nil, err
		}
		out = append(out, pkg)
	}
	return out, nil
}

// Analyze walks a package and reports link state in the target tree.
func Analyze(opts Options, name string) (Package, error) {
	pkgPath := filepath.Join(opts.Dir, name)
	st, err := os.Stat(pkgPath)
	if err != nil {
		return Package{}, err
	}
	if !st.IsDir() {
		return Package{}, fmt.Errorf("%s: not a package directory", name)
	}
	pkg := Package{Name: name, Path: pkgPath}
	links, err := analyzeTree(pkgPath, opts.Target, "")
	if err != nil {
		return pkg, err
	}
	pkg.Links = links
	for _, l := range links {
		switch l.State {
		case LinkLinked:
			pkg.Linked++
		case LinkMissing:
			pkg.Missing++
		case LinkConflict:
			pkg.Conflicts++
		}
	}
	switch {
	case len(links) == 0:
		pkg.Status = StatusEmpty
	case pkg.Conflicts > 0:
		pkg.Status = StatusConflict
	case pkg.Missing == 0 && pkg.Linked > 0:
		pkg.Status = StatusLinked
	case pkg.Linked == 0:
		pkg.Status = StatusUnlinked
	default:
		pkg.Status = StatusPartial
	}
	return pkg, nil
}

func analyzeTree(pkgRoot, targetRoot, rel string) ([]Link, error) {
	src := pkgRoot
	if rel != "" {
		src = filepath.Join(pkgRoot, rel)
	}
	ents, err := os.ReadDir(src)
	if err != nil {
		return nil, err
	}
	var out []Link
	for _, e := range ents {
		name := e.Name()
		childRel := name
		if rel != "" {
			childRel = filepath.Join(rel, name)
		}
		if ignorePath(childRel) {
			continue
		}
		links, err := analyzeNode(pkgRoot, targetRoot, childRel)
		if err != nil {
			return nil, err
		}
		out = append(out, links...)
	}
	return out, nil
}

// ignorePath mirrors GNU Stow's built-in global ignore list (subset).
func ignorePath(rel string) bool {
	base := filepath.Base(rel)
	switch base {
	case ".git", ".gitignore", ".gitmodules",
		"RCS", "CVS", ".svn", "_darcs", ".hg",
		".stow-local-ignore", "node_modules":
		return true
	}
	if strings.HasSuffix(base, "~") {
		return true
	}
	// Path patterns that only apply at package root.
	if !strings.Contains(rel, string(os.PathSeparator)) {
		if strings.HasPrefix(base, "README") || strings.HasPrefix(base, "LICENSE") || base == "COPYING" {
			return true
		}
	}
	return false
}

func analyzeNode(pkgRoot, targetRoot, rel string) ([]Link, error) {
	src := filepath.Join(pkgRoot, rel)
	dst := filepath.Join(targetRoot, rel)

	srcInfo, err := os.Lstat(src)
	if err != nil {
		return nil, err
	}

	dstInfo, err := os.Lstat(dst)
	if err != nil {
		if os.IsNotExist(err) {
			return []Link{{Rel: rel, Target: dst, Source: src, State: LinkMissing}}, nil
		}
		return nil, err
	}

	if dstInfo.Mode()&os.ModeSymlink != 0 {
		if pointsTo(dst, src) {
			return []Link{{Rel: rel, Target: dst, Source: src, State: LinkLinked}}, nil
		}
		return []Link{{Rel: rel, Target: dst, Source: src, State: LinkConflict}}, nil
	}

	// Tree folding: both sides are directories → recurse.
	if srcInfo.IsDir() && dstInfo.IsDir() {
		return analyzeTree(pkgRoot, targetRoot, rel)
	}

	// Regular file/dir occupying the target slot.
	return []Link{{Rel: rel, Target: dst, Source: src, State: LinkConflict}}, nil
}

func pointsTo(link, want string) bool {
	resolved, err := filepath.EvalSymlinks(link)
	if err != nil {
		// dangling or unreadable — compare readlink when possible
		target, err2 := os.Readlink(link)
		if err2 != nil {
			return false
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(link), target)
		}
		resolved = filepath.Clean(target)
	}
	wantResolved, err := filepath.EvalSymlinks(want)
	if err != nil {
		wantResolved = filepath.Clean(want)
	}
	return resolved == wantResolved
}

// Owner reports which package owns absPath (via symlink resolution into the stow dir).
func Owner(opts Options, absPath string) (pkg string, ok bool) {
	if opts.Dir == "" || absPath == "" {
		return "", false
	}
	real := absPath
	if r, err := filepath.EvalSymlinks(absPath); err == nil {
		real = r
	}
	dir := filepath.Clean(opts.Dir)
	rel, err := filepath.Rel(dir, real)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return "", false
	}
	pkg, _, _ = strings.Cut(rel, string(os.PathSeparator))
	if pkg == "" {
		return "", false
	}
	return pkg, true
}

// OwnershipMap maps absolute config paths to owning package names.
func OwnershipMap(opts Options, paths []string) map[string]string {
	out := make(map[string]string)
	for _, p := range paths {
		if pkg, ok := Owner(opts, p); ok {
			out[p] = pkg
		}
	}
	return out
}

// Run invokes the stow binary. dryRun adds -n.
func Run(opts Options, action Action, packages []string, dryRun bool) (string, error) {
	if len(packages) == 0 {
		return "", fmt.Errorf("no packages specified")
	}
	if _, err := exec.LookPath("stow"); err != nil {
		return "", fmt.Errorf("stow not found on PATH")
	}
	args := []string{"-d", opts.Dir, "-t", opts.Target, "-v"}
	if dryRun {
		args = append(args, "-n")
	}
	switch action {
	case ActionStow:
		// default
	case ActionUnstow:
		args = append(args, "-D")
	case ActionRestow:
		args = append(args, "-R")
	default:
		return "", fmt.Errorf("unknown action %q", action)
	}
	args = append(args, packages...)
	cmd := exec.Command("stow", args...)
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			return "", fmt.Errorf("stow %s: %w", action, err)
		}
		return text, fmt.Errorf("stow %s: %w\n%s", action, err, text)
	}
	return text, nil
}

// Available reports whether the stow binary is on PATH.
func Available() bool {
	_, err := exec.LookPath("stow")
	return err == nil
}
