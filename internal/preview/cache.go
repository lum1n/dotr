package preview

import (
	"os"
	"sync"
)

const cacheCap = 32

type cacheKey struct {
	path  string
	width int
	size  int64
	mod   int64
	style string
}

type cacheEntry struct {
	key cacheKey
	res Result
}

type lru struct {
	mu    sync.Mutex
	items []cacheEntry
}

func keyOf(path string, width int, fi os.FileInfo) cacheKey {
	return cacheKey{
		path:  path,
		width: width,
		size:  fi.Size(),
		mod:   fi.ModTime().UnixNano(),
		style: StyleName,
	}
}

func (c *lru) get(k cacheKey) (Result, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	for i, it := range c.items {
		if it.key != k {
			continue
		}
		if i > 0 {
			c.items = append(append([]cacheEntry{it}, c.items[:i]...), c.items[i+1:]...)
		}
		return it.res, true
	}
	return Result{}, false
}

func (c *lru) put(k cacheKey, res Result) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, it := range c.items {
		if it.key.path == k.path && it.key.width == k.width {
			continue
		}
		c.items[n] = it
		n++
	}
	c.items = append([]cacheEntry{{key: k, res: res}}, c.items[:n]...)
	if len(c.items) > cacheCap {
		c.items = c.items[:cacheCap]
	}
}

func (c *lru) invalidate(path string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	n := 0
	for _, it := range c.items {
		if it.key.path != path {
			c.items[n] = it
			n++
		}
	}
	c.items = c.items[:n]
}

func (c *lru) clear() {
	c.mu.Lock()
	c.items = c.items[:0]
	c.mu.Unlock()
}

var previewCache lru

// Lookup returns a cached preview when path, width, size, and mtime match.
func Lookup(path string, width int) (Result, bool) {
	if width < 20 {
		width = 80
	}
	fi, err := os.Stat(path)
	if err != nil {
		return Result{}, false
	}
	return previewCache.get(keyOf(path, width, fi))
}

// Invalidate drops cached previews for path (all widths).
func Invalidate(path string) {
	previewCache.invalidate(path)
}

// Clear drops the whole preview cache.
func Clear() {
	previewCache.clear()
}
