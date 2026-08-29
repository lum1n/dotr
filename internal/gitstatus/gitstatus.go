// Package gitstatus reports porcelain git state for config paths.
package gitstatus

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Kind is a simplified git working-tree state for one path.
type Kind int

const (
	None Kind = iota // not in a git repo / unknown
	Clean
	Modified
	Added
	Deleted
	Untracked
	Ignored
)

func (k Kind) String() string {
	switch k {
	case Clean:
		return " "
	case Modified:
		return "M"
	case Added:
		return "A"
	case Deleted:
		return "D"
	case Untracked:
		return "?"
	case Ignored:
		return "!"
	default:
		return ""
	}
}

// Mark is a short list badge (empty if None).
func (k Kind) Mark() string {
	switch k {
	case Modified, Added, Deleted, Untracked, Ignored:
		return k.String()
	case Clean:
		return "·"
	default:
		return ""
	}
}

// Map returns status keyed by absolute path (as given).
func Map(paths []string) map[string]Kind {
	out := make(map[string]Kind, len(paths))
	if len(paths) == 0 {
		return out
	}
	if _, err := exec.LookPath("git"); err != nil {
		return out
	}

	type item struct {
		orig string
		rel  string
	}

	cache := newRepoCache()
	byRoot := map[string][]item{}
	for _, p := range paths {
		real := cache.realPath(p)
		root := cache.repoRoot(real)
		if root == "" {
			out[p] = None
			continue
		}
		rel, err := filepath.Rel(root, real)
		if err != nil {
			out[p] = None
			continue
		}
		byRoot[root] = append(byRoot[root], item{orig: p, rel: rel})
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for root, items := range byRoot {
		wg.Add(1)
		go func(root string, items []item) {
			defer wg.Done()
			st := porcelain(root)
			mu.Lock()
			defer mu.Unlock()
			for _, it := range items {
				rel := filepath.ToSlash(it.rel)
				if k, ok := st[rel]; ok {
					out[it.orig] = k
				} else {
					out[it.orig] = Clean
				}
			}
		}(root, items)
	}
	wg.Wait()
	return out
}

// repoCache finds symlink-resolved paths and git tops without spawning git
// once per file. A single Map call of 1100 paths in one repo should do a
// handful of stats plus one `git status`.
type repoCache struct {
	real    map[string]string
	dirReal map[string]string
	root    map[string]string // dir → repo root or ""
}

func newRepoCache() *repoCache {
	return &repoCache{
		real:    map[string]string{},
		dirReal: map[string]string{},
		root:    map[string]string{},
	}
}

func (c *repoCache) realPath(p string) string {
	if v, ok := c.real[p]; ok {
		return v
	}
	dir := filepath.Dir(p)
	if rd, ok := c.dirReal[dir]; ok {
		out := filepath.Join(rd, filepath.Base(p))
		c.real[p] = out
		return out
	}
	real, err := filepath.EvalSymlinks(p)
	if err != nil {
		c.real[p] = p
		c.dirReal[dir] = dir
		return p
	}
	c.real[p] = real
	c.dirReal[dir] = filepath.Dir(real)
	return real
}

func (c *repoCache) repoRoot(path string) string {
	dir := path
	if fi, err := os.Stat(path); err == nil && !fi.IsDir() {
		dir = filepath.Dir(path)
	}

	var chain []string
	cur := dir
	for {
		if v, cached := c.root[cur]; cached {
			for _, d := range chain {
				c.root[d] = v
			}
			return v
		}
		chain = append(chain, cur)
		if isGitRepo(cur) {
			for _, d := range chain {
				c.root[d] = cur
			}
			return cur
		}
		parent := filepath.Dir(cur)
		if parent == cur {
			for _, d := range chain {
				c.root[d] = ""
			}
			return ""
		}
		cur = parent
	}
}

func isGitRepo(dir string) bool {
	_, err := os.Lstat(filepath.Join(dir, ".git"))
	return err == nil
}

func porcelain(root string) map[string]Kind {
	cmd := exec.Command("git", "-C", root, "status", "--porcelain", "-uall", "--ignored=matching")
	out, err := cmd.Output()
	if err != nil {
		return nil
	}
	res := map[string]Kind{}
	for _, line := range bytes.Split(out, []byte{'\n'}) {
		if len(line) < 4 {
			continue
		}
		xy := string(line[:2])
		path := string(line[3:])
		// rename: "R  old -> new"
		if i := strings.Index(path, " -> "); i >= 0 {
			path = path[i+4:]
		}
		path = filepath.ToSlash(path)
		res[path] = decode(xy)
	}
	return res
}

func decode(xy string) Kind {
	// Prefer index/worktree dirty bits. See git-status porcelain.
	x, y := xy[0], xy[1]
	switch {
	case x == '?' || y == '?':
		return Untracked
	case x == '!' || y == '!':
		return Ignored
	case x == 'A' || y == 'A':
		return Added
	case x == 'D' || y == 'D':
		return Deleted
	case x == 'M' || y == 'M' || x == 'U' || y == 'U' || x == 'R' || y == 'R' || x == 'C' || y == 'C':
		return Modified
	default:
		if strings.TrimSpace(xy) == "" {
			return Clean
		}
		return Modified
	}
}
