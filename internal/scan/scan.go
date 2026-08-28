// Package scan discovers config files under $HOME and $XDG_CONFIG_HOME.
package scan

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/lum1n/dotr/internal/ignore"
)

const (
	maxDepthConfig = 5
	maxFiles       = 1500
)

// Entry is one discovered config path.
type Entry struct {
	App     string // grouping key: "home", "config", or ~/.config/<app>
	RelPath string
	AbsPath string
	Size    int64
	IsDir   bool
	Symlink bool
}

var skipDirNames = map[string]struct{}{
	".git": {}, ".android": {}, ".cache": {},
	"cache": {}, "Cache": {}, "CachedData": {}, "Code Cache": {},
	"GPUCache": {}, "Crashpad": {}, "Crash Reports": {},
	"node_modules": {}, "blob_storage": {}, "Service Worker": {},
	"Session Storage": {}, "Local Storage": {}, "IndexedDB": {},
	"databases": {}, "logs": {}, "log": {}, "tmp": {}, "Temp": {},
	"BraveSoftware": {}, "chromium": {}, "google-chrome": {}, "chrome": {},
	"firefox": {}, "Slack": {}, "discord": {}, "Zoom": {},
	"Code": {}, "Cursor": {}, "cursor": {}, "Claude": {}, "1Password": {},
	"com.docker.sandboxes": {}, "configstore": {}, "Electron": {},
}

func init() {
	for _, n := range []string{
		"Caprine", "spotify", "Signal", "TelegramDesktop", "Microsoft",
		"Google", "JetBrains", "github-copilot", "Epiphany",
		"google-chrome-beta", "vivaldi", "opera",
		"autostart", "dconf", "pulse", "evolution", "libreoffice",
		"GIMP", "inkscape", "obs-studio", "unity3d", "goa-1.0",
		"ibus", "ibus-table", "fcitx", "fcitx5", "systemd",
		".wrangler", "wrangler", "nextjs-nodejs",
		"dotr", // dotr's own config dir
	} {
		skipDirNames[n] = struct{}{}
	}
}

var homeDotAllow = map[string]struct{}{
	".bashrc": {}, ".bash_profile": {}, ".profile": {},
	".zshrc": {}, ".zshenv": {}, ".zprofile": {},
	".gitconfig": {}, ".gitignore_global": {},
	".tmux.conf": {}, ".vimrc": {}, ".gvimrc": {},
	".editorconfig": {}, ".npmrc": {}, ".stowrc": {},
	".inputrc": {}, ".dir_colors": {},
	".Xresources": {}, ".xprofile": {}, ".xinitrc": {},
	".imwheelrc": {},
}

var configExt = map[string]struct{}{
	".toml": {}, ".yaml": {}, ".yml": {}, ".json": {}, ".jsonc": {},
	".conf": {}, ".cfg": {}, ".ini": {}, ".env": {},
	".lua": {}, ".vim": {}, ".nix": {}, ".fish": {}, ".sh": {},
	".zsh": {}, ".bash": {}, ".py": {}, ".ts": {}, ".js": {},
	".css": {}, ".scss": {}, ".md": {}, ".txt": {}, ".xml": {},
	".kdl": {}, ".ron": {}, ".hcl": {}, ".tf": {}, ".service": {},
	".desktop": {}, ".plist": {},
}

// Roots returns the home and config directories to scan.
func Roots() (home, config string, err error) {
	home, err = os.UserHomeDir()
	if err != nil {
		return "", "", err
	}
	config = os.Getenv("XDG_CONFIG_HOME")
	if config == "" {
		config = filepath.Join(home, ".config")
	}
	return home, config, nil
}

// MaxFiles is the soft cap on discovered entries.
func MaxFiles() int { return maxFiles }

// Scan walks home (depth 1) and XDG config (bounded recurse, follows stow symlinks).
func Scan() ([]Entry, error) {
	ents, _, err := ScanWithIgnore(nil)
	return ents, err
}

// ScanWithIgnore is Scan with a user ignore list applied during the walk.
// truncated is true when results were capped at MaxFiles.
func ScanWithIgnore(ign *ignore.List) (entries []Entry, truncated bool, err error) {
	home, config, err := Roots()
	if err != nil {
		return nil, false, err
	}

	homeEntries := scanHomeDots(home, ign)
	cfgEntries, err := scanConfigTree(config, ign)
	if err != nil && !os.IsNotExist(err) {
		return homeEntries, false, err
	}

	out := append(homeEntries, cfgEntries...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].App != out[j].App {
			return out[i].App < out[j].App
		}
		return out[i].RelPath < out[j].RelPath
	})
	if len(out) > maxFiles {
		return prioritizeTrim(out, maxFiles), true, nil
	}
	return out, false, nil
}

func prioritizeTrim(in []Entry, limit int) []Entry {
	var home, rest []Entry
	for _, e := range in {
		if e.App == "home" {
			home = append(home, e)
		} else {
			rest = append(rest, e)
		}
	}
	remain := limit - len(home)
	if remain < 0 {
		return home[:limit]
	}
	if len(rest) > remain {
		rest = rest[:remain]
	}
	out := append(home, rest...)
	sort.Slice(out, func(i, j int) bool {
		if out[i].App != out[j].App {
			return out[i].App < out[j].App
		}
		return out[i].RelPath < out[j].RelPath
	})
	return out
}

func scanHomeDots(home string, ign *ignore.List) []Entry {
	ents, err := os.ReadDir(home)
	if err != nil {
		return nil
	}
	var out []Entry
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, ".") || e.IsDir() {
			continue
		}
		if _, ok := homeDotAllow[name]; !ok && !looksLikeHomeConfig(name) {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		abs := filepath.Join(home, name)
		rel := ignore.RelPath("home", name)
		if ign != nil && ign.Match(rel, abs) {
			continue
		}
		out = append(out, Entry{
			App:     "home",
			RelPath: name,
			AbsPath: abs,
			Size:    info.Size(),
			Symlink: isSymlink(abs),
		})
	}
	return out
}

func looksLikeHomeConfig(name string) bool {
	lower := strings.ToLower(name)
	return strings.HasSuffix(lower, "rc") ||
		strings.HasSuffix(lower, "profile") ||
		strings.HasSuffix(lower, ".conf") ||
		strings.HasSuffix(lower, ".toml") ||
		strings.HasSuffix(lower, ".yaml") ||
		strings.HasSuffix(lower, ".yml") ||
		strings.HasSuffix(lower, ".json")
}

func scanConfigTree(config string, ign *ignore.List) ([]Entry, error) {
	info, err := os.Stat(config)
	if err != nil {
		return nil, err
	}
	if !info.IsDir() {
		return nil, nil
	}

	var out []Entry
	seen := map[string]struct{}{}
	var walk func(abs, rel string, depth int)
	walk = func(abs, rel string, depth int) {
		if len(out) >= maxFiles || depth > maxDepthConfig {
			return
		}
		if rel != "" && ign != nil && ign.MatchDir(rel) {
			return
		}
		real, err := filepath.EvalSymlinks(abs)
		if err == nil {
			if _, ok := seen[real]; ok {
				return
			}
			seen[real] = struct{}{}
		}

		ents, err := os.ReadDir(abs)
		if err != nil {
			return
		}
		for _, e := range ents {
			if len(out) >= maxFiles {
				return
			}
			name := e.Name()
			childAbs := filepath.Join(abs, name)
			childRel := name
			if rel != "" {
				childRel = filepath.Join(rel, name)
			}

			// Directory (or symlink-to-dir): ReadDir/Stat follow links.
			fi, err := os.Stat(childAbs)
			if err != nil {
				continue
			}
			if fi.IsDir() {
				if shouldSkipDir(name) {
					continue
				}
				if ign != nil && ign.MatchDir(childRel) {
					continue
				}
				walk(childAbs, childRel, depth+1)
				continue
			}
			if !fi.Mode().IsRegular() {
				continue
			}
			if !looksLikeConfigFile(name, childAbs) {
				continue
			}

			parts := strings.Split(childRel, string(os.PathSeparator))
			var app, relInApp string
			if len(parts) == 1 {
				app = "config"
				relInApp = parts[0]
			} else {
				app = parts[0]
				relInApp = filepath.Join(parts[1:]...)
			}
			matchKey := ignore.RelPath(app, relInApp)
			if ign != nil && ign.Match(matchKey, childAbs) {
				continue
			}
			out = append(out, Entry{
				App:     app,
				RelPath: relInApp,
				AbsPath: childAbs,
				Size:    fi.Size(),
				Symlink: isSymlink(childAbs) || isSymlink(filepath.Join(config, parts[0])),
			})
		}
	}

	walk(config, "", 0)
	return out, nil
}

func shouldSkipDir(name string) bool {
	if _, ok := skipDirNames[name]; ok {
		return true
	}
	lower := strings.ToLower(name)
	return strings.Contains(lower, "cache") || strings.Contains(lower, "crash")
}

func looksLikeConfigFile(name, path string) bool {
	lower := strings.ToLower(name)
	ext := strings.ToLower(filepath.Ext(name))
	if _, ok := configExt[ext]; ok {
		return true
	}
	if strings.HasSuffix(lower, "rc") || strings.HasSuffix(lower, "profile") {
		return true
	}
	switch lower {
	case "config", "configuration", "settings", "preferences",
		"dockerfile", "makefile", "justfile", "gemfile":
		return true
	}
	if ext == "" {
		fi, err := os.Stat(path)
		if err == nil && fi.Size() > 0 && fi.Size() < 64<<10 {
			return true
		}
	}
	return false
}

func isSymlink(path string) bool {
	fi, err := os.Lstat(path)
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeSymlink != 0
}
