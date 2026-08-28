package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/lum1n/dotr/internal/config"
	"github.com/lum1n/dotr/internal/ignore"
	"github.com/lum1n/dotr/internal/scan"
	"github.com/lum1n/dotr/internal/stow"
	"github.com/lum1n/dotr/internal/version"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check dotr environment health",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		ok := true
		check := func(name string, err error, detail string) {
			if err != nil {
				ok = false
				fmt.Printf("✗ %s: %v\n", name, err)
				return
			}
			if detail == "" {
				fmt.Printf("✓ %s\n", name)
				return
			}
			fmt.Printf("✓ %s: %s\n", name, detail)
		}

		fmt.Printf("dotr %s\n\n", version.String())

		home, configHome, err := scan.Roots()
		check("home / config roots", err, fmt.Sprintf("%s · %s", home, configHome))

		cfgDir, err := config.Dir()
		check("config dir", err, cfgDir)
		if err == nil {
			if err := os.MkdirAll(cfgDir, 0o700); err != nil {
				check("config dir writable", err, "")
			} else {
				check("config dir writable", nil, "")
			}
		}

		cfg, err := config.Load()
		check("config.yaml", err, "")

		ignPath, err := ignore.Path()
		if err != nil {
			check("ignore file", err, "")
		} else {
			check("ignore file", nil, ignPath)
		}

		editor := strings.TrimSpace(os.Getenv("EDITOR"))
		if editor == "" {
			editor = strings.TrimSpace(os.Getenv("VISUAL"))
		}
		if editor == "" {
			ok = false
			fmt.Println("✗ $EDITOR / $VISUAL: not set")
		} else {
			fmt.Printf("✓ editor: %s\n", editor)
		}

		if stow.Available() {
			path, _ := exec.LookPath("stow")
			fmt.Printf("✓ stow: %s\n", path)
			opts, err := stow.Resolve(cfg.StowDir, cfg.StowTarget)
			if err != nil {
				fmt.Printf("· stow resolve: %v (optional until you use stow)\n", err)
			} else {
				pkgs, err := stow.Packages(opts)
				if err != nil {
					check("stow packages", err, "")
				} else {
					check("stow", nil, fmt.Sprintf("%s → %s (%s)", opts.Dir, opts.Target, strings.Join(pkgs, ", ")))
				}
			}
		} else {
			fmt.Println("· stow: not on PATH (optional)")
		}

		backupBase := os.Getenv("XDG_DATA_HOME")
		if backupBase == "" {
			backupBase = filepath.Join(home, ".local", "share")
		}
		fmt.Printf("✓ backups: %s\n", filepath.Join(backupBase, "dotr", "backups"))

		if !ok {
			return fmt.Errorf("doctor found issues")
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}
