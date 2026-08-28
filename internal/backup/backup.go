// Package backup stores timestamped copies of config files under XDG_DATA_HOME.
package backup

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	defaultMaxPerFile = 20
	timeLayout        = "2006-01-02T15-04-05"
)

// Snapshot is one backup of a source file.
type Snapshot struct {
	Source  string
	Path    string
	Name    string
	ModTime time.Time
	Size    int64
}

func dataDir() (string, error) {
	base := os.Getenv("XDG_DATA_HOME")
	if base == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		base = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(base, "dotr", "backups"), nil
}

// dirFor returns the backup directory for a source absolute path.
func dirFor(source string) (string, error) {
	root, err := dataDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(root, encodePath(source)), nil
}

func encodePath(abs string) string {
	abs = filepath.Clean(abs)
	abs = strings.TrimPrefix(abs, string(os.PathSeparator))
	return strings.ReplaceAll(abs, string(os.PathSeparator), "__")
}

// Create copies source into a new timestamped snapshot and prunes old ones.
// keep <= 0 uses the default retention (20).
func Create(source string, keep int) (Snapshot, error) {
	if keep <= 0 {
		keep = defaultMaxPerFile
	}
	fi, err := os.Stat(source)
	if err != nil {
		return Snapshot{}, err
	}
	if !fi.Mode().IsRegular() {
		fi, err = os.Stat(source)
		if err != nil || !fi.Mode().IsRegular() {
			return Snapshot{}, fmt.Errorf("not a regular file: %s", source)
		}
	}

	dir, err := dirFor(source)
	if err != nil {
		return Snapshot{}, err
	}
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return Snapshot{}, err
	}

	stamp := time.Now().Format(timeLayout)
	base := filepath.Base(source)
	name := stamp + "__" + base
	dest := filepath.Join(dir, name)

	if err := copyFile(source, dest); err != nil {
		return Snapshot{}, err
	}
	_ = os.Chmod(dest, 0o600)

	_ = prune(dir, keep)

	st, err := os.Stat(dest)
	if err != nil {
		return Snapshot{Source: source, Path: dest, Name: name}, nil
	}
	return Snapshot{
		Source:  source,
		Path:    dest,
		Name:    name,
		ModTime: st.ModTime(),
		Size:    st.Size(),
	}, nil
}

// List returns snapshots for source, newest first.
func List(source string) ([]Snapshot, error) {
	dir, err := dirFor(source)
	if err != nil {
		return nil, err
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []Snapshot
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		out = append(out, Snapshot{
			Source:  source,
			Path:    filepath.Join(dir, e.Name()),
			Name:    e.Name(),
			ModTime: info.ModTime(),
			Size:    info.Size(),
		})
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].ModTime.After(out[j].ModTime)
	})
	return out, nil
}

// Restore copies snapshot back onto source (overwrites).
func Restore(snap Snapshot) error {
	if snap.Path == "" || snap.Source == "" {
		return fmt.Errorf("invalid snapshot")
	}
	dir := filepath.Dir(snap.Source)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	tmp := snap.Source + ".dotr-restore-tmp"
	if err := copyFile(snap.Path, tmp); err != nil {
		return err
	}
	return os.Rename(tmp, snap.Source)
}

func prune(dir string, keep int) error {
	ents, err := os.ReadDir(dir)
	if err != nil {
		return err
	}
	type item struct {
		name string
		t    time.Time
	}
	var files []item
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, item{name: e.Name(), t: info.ModTime()})
	}
	if len(files) <= keep {
		return nil
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].t.After(files[j].t)
	})
	for _, f := range files[keep:] {
		_ = os.Remove(filepath.Join(dir, f.name))
	}
	return nil
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Close()
}

// DisplayName formats a snapshot for the TUI list.
func (s Snapshot) DisplayName() string {
	stamp := s.ModTime.Format("2006-01-02 15:04:05")
	if s.ModTime.IsZero() {
		// parse from filename prefix if possible
		if i := strings.Index(s.Name, "__"); i > 0 {
			if t, err := time.Parse(timeLayout, s.Name[:i]); err == nil {
				stamp = t.Format("2006-01-02 15:04:05")
			}
		}
	}
	return fmt.Sprintf("%s  (%dB)", stamp, s.Size)
}
