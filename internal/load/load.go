// Package load loads config, ignores, and scanned entries for CLI and TUI.
package load

import (
	"fmt"

	"github.com/lum1n/dotr/internal/config"
	"github.com/lum1n/dotr/internal/ignore"
	"github.com/lum1n/dotr/internal/preview"
	"github.com/lum1n/dotr/internal/scan"
)

// Result is a loaded workspace snapshot.
type Result struct {
	Config    config.Config
	Ignores   *ignore.List
	Entries   []scan.Entry
	Truncated bool
}

// All loads dotr config, ignore list, and scanned entries.
func All() (Result, error) {
	cfg, err := config.Load()
	if err != nil {
		return Result{}, fmt.Errorf("config: %w", err)
	}
	preview.StyleName = cfg.ChromaStyle

	ign, err := ignore.Load()
	if err != nil {
		return Result{}, fmt.Errorf("ignores: %w", err)
	}
	for _, p := range cfg.ExtraIgnores {
		ign.Add(p)
	}
	entries, truncated, err := scan.ScanWithIgnore(ign)
	if err != nil {
		return Result{}, fmt.Errorf("scan: %w", err)
	}
	return Result{
		Config:    cfg,
		Ignores:   ign,
		Entries:   entries,
		Truncated: truncated,
	}, nil
}
