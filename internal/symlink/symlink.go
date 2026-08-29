// Package symlink provides basic create / retarget / remove helpers for dotr.
package symlink

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Info describes a symlink.
type Info struct {
	Path   string
	Target string // as stored (may be relative)
	Abs    string // absolute resolved target when possible
	Dangle bool
}

// Read returns symlink info. err if path is not a symlink.
func Read(path string) (Info, error) {
	fi, err := os.Lstat(path)
	if err != nil {
		return Info{}, err
	}
	if fi.Mode()&os.ModeSymlink == 0 {
		return Info{}, fmt.Errorf("not a symlink: %s", path)
	}
	target, err := os.Readlink(path)
	if err != nil {
		return Info{}, err
	}
	info := Info{Path: path, Target: target}
	abs := target
	if !filepath.IsAbs(target) {
		abs = filepath.Join(filepath.Dir(path), target)
	}
	abs = filepath.Clean(abs)
	info.Abs = abs
	if _, err := os.Stat(abs); err != nil {
		info.Dangle = true
	}
	return info, nil
}

// Is reports whether path is a symlink.
func Is(path string) bool {
	fi, err := os.Lstat(path)
	return err == nil && fi.Mode()&os.ModeSymlink != 0
}

// Create makes a new symlink. Fails if link already exists.
// target is stored as given (after optional tilde expansion by caller).
func Create(link, target string) error {
	link = filepath.Clean(link)
	if target == "" {
		return fmt.Errorf("empty symlink target")
	}
	if _, err := os.Lstat(link); err == nil {
		return fmt.Errorf("already exists: %s", link)
	} else if !os.IsNotExist(err) {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		return err
	}
	if err := os.Symlink(target, link); err != nil {
		return fmt.Errorf("symlink: %w", err)
	}
	return nil
}

// Retarget replaces an existing symlink's target (same path).
func Retarget(link, newTarget string) error {
	if newTarget == "" {
		return fmt.Errorf("empty symlink target")
	}
	if !Is(link) {
		return fmt.Errorf("not a symlink: %s", link)
	}
	tmp := link + ".dotr-new"
	_ = os.Remove(tmp)
	if err := os.Symlink(newTarget, tmp); err != nil {
		return err
	}
	if err := os.Rename(tmp, link); err != nil {
		_ = os.Remove(tmp)
		// Fallback: remove + create
		if err2 := os.Remove(link); err2 != nil {
			return err
		}
		if err2 := os.Symlink(newTarget, link); err2 != nil {
			return err2
		}
	}
	return nil
}

// Remove deletes a symlink only (never a regular file/dir).
func Remove(link string) error {
	if !Is(link) {
		return fmt.Errorf("not a symlink: %s", link)
	}
	return os.Remove(link)
}

// ExpandTilde expands a leading ~/ in path.
func ExpandTilde(path, home string) string {
	path = strings.TrimSpace(path)
	if path == "~" {
		return home
	}
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
