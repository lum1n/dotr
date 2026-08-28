// Package watch observes config directories and emits debounced change events.
package watch

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Event is a debounced filesystem change.
type Event struct {
	Path string
	Op   string // write|create|remove|rename|other
}

// Watcher watches top-level config roots and an optional focus directory.
type Watcher struct {
	w      *fsnotify.Watcher
	events chan Event
	done   chan struct{}

	mu     sync.Mutex
	focus  string // currently focused parent dir
	roots  []string
}

// New starts watching roots (typically $HOME and $XDG_CONFIG_HOME).
func New(roots ...string) (*Watcher, error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	ww := &Watcher{
		w:      w,
		events: make(chan Event, 16),
		done:   make(chan struct{}),
		roots:  append([]string{}, roots...),
	}
	for _, r := range roots {
		if r == "" {
			continue
		}
		if fi, err := os.Stat(r); err == nil && fi.IsDir() {
			_ = w.Add(r)
		}
	}
	go ww.loop()
	return ww, nil
}

// Events returns the debounced event channel.
func (w *Watcher) Events() <-chan Event { return w.events }

// SetFocus watches an additional directory (e.g. parent of selected file).
// Replaces any previous focus watch.
func (w *Watcher) SetFocus(dir string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if dir == w.focus {
		return
	}
	if w.focus != "" && !w.isRoot(w.focus) {
		_ = w.w.Remove(w.focus)
	}
	w.focus = ""
	if dir == "" {
		return
	}
	fi, err := os.Stat(dir)
	if err != nil || !fi.IsDir() {
		return
	}
	if err := w.w.Add(dir); err != nil {
		return
	}
	w.focus = dir
}

func (w *Watcher) isRoot(dir string) bool {
	for _, r := range w.roots {
		if r == dir {
			return true
		}
	}
	return false
}

// Close stops the watcher.
func (w *Watcher) Close() error {
	select {
	case <-w.done:
	default:
		close(w.done)
	}
	return w.w.Close()
}

func (w *Watcher) loop() {
	const debounce = 250 * time.Millisecond
	var (
		timer *time.Timer
		pending Event
		has bool
	)
	flush := func() {
		if !has {
			return
		}
		select {
		case w.events <- pending:
		default:
			// drop if UI is busy
		}
		has = false
	}

	for {
		select {
		case <-w.done:
			if timer != nil {
				timer.Stop()
			}
			return
		case err, ok := <-w.w.Errors:
			if !ok {
				return
			}
			_ = err
		case ev, ok := <-w.w.Events:
			if !ok {
				return
			}
			// Ignore chmod noise from editors.
			if ev.Has(fsnotify.Chmod) && !ev.Has(fsnotify.Write) && !ev.Has(fsnotify.Create) && !ev.Has(fsnotify.Remove) && !ev.Has(fsnotify.Rename) {
				continue
			}
			op := "other"
			switch {
			case ev.Has(fsnotify.Write):
				op = "write"
			case ev.Has(fsnotify.Create):
				op = "create"
			case ev.Has(fsnotify.Remove):
				op = "remove"
			case ev.Has(fsnotify.Rename):
				op = "rename"
			}
			pending = Event{Path: ev.Name, Op: op}
			has = true
			if timer == nil {
				timer = time.AfterFunc(debounce, func() {
					w.mu.Lock()
					flush()
					w.mu.Unlock()
				})
			} else {
				timer.Reset(debounce)
			}
			// Dynamically watch new top-level dirs under roots.
			if ev.Has(fsnotify.Create) {
				if fi, err := os.Stat(ev.Name); err == nil && fi.IsDir() {
					parent := filepath.Dir(ev.Name)
					w.mu.Lock()
					if w.isRoot(parent) {
						_ = w.w.Add(ev.Name)
					}
					w.mu.Unlock()
				}
			}
		}
	}
}
