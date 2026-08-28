package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/vegarringdal/dotr/internal/config"
	"github.com/vegarringdal/dotr/internal/stow"
)

var stowDryRun bool
var stowJSON bool

var stowCmd = &cobra.Command{
	Use:   "stow",
	Short: "Manage GNU Stow packages",
	Long: `Thin wrapper around GNU Stow using ~/.stowrc (or stow_dir / stow_target in config).

  dotr stow              list packages and link status
  dotr stow list
  dotr stow link [pkg…]  stow (add links)
  dotr stow unlink [pkg…] unstow
  dotr stow restow [pkg…]`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStowList()
	},
}

var stowListCmd = &cobra.Command{
	Use:   "list",
	Short: "List stow packages and status",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStowList()
	},
}

var stowLinkCmd = &cobra.Command{
	Use:   "link [package...]",
	Short: "Stow packages (create symlinks)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStowAction(stow.ActionStow, args)
	},
}

var stowUnlinkCmd = &cobra.Command{
	Use:   "unlink [package...]",
	Short: "Unstow packages (remove symlinks)",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStowAction(stow.ActionUnstow, args)
	},
}

var stowRestowCmd = &cobra.Command{
	Use:   "restow [package...]",
	Short: "Restow packages",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runStowAction(stow.ActionRestow, args)
	},
}

func resolveStowOpts() (stow.Options, error) {
	cfg, err := config.Load()
	if err != nil {
		return stow.Options{}, err
	}
	return stow.Resolve(cfg.StowDir, cfg.StowTarget)
}

func runStowList() error {
	opts, err := resolveStowOpts()
	if err != nil {
		return err
	}
	pkgs, err := stow.AnalyzeAll(opts)
	if err != nil {
		return err
	}
	if stowJSON {
		type row struct {
			Name      string `json:"name"`
			Status    string `json:"status"`
			Linked    int    `json:"linked"`
			Missing   int    `json:"missing"`
			Conflicts int    `json:"conflicts"`
			Path      string `json:"path"`
		}
		out := make([]row, 0, len(pkgs))
		for _, p := range pkgs {
			out = append(out, row{
				Name: p.Name, Status: p.Status.String(),
				Linked: p.Linked, Missing: p.Missing, Conflicts: p.Conflicts,
				Path: p.Path,
			})
		}
		enc := json.NewEncoder(os.Stdout)
		enc.SetIndent("", "  ")
		return enc.Encode(out)
	}
	fmt.Fprintf(os.Stderr, "stow dir %s → target %s\n", opts.Dir, opts.Target)
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	for _, p := range pkgs {
		fmt.Fprintf(w, "%s\t%s\t%d linked\t%d missing\t%d conflict\n",
			p.Status.Mark(), p.Name, p.Linked, p.Missing, p.Conflicts)
	}
	return w.Flush()
}

func runStowAction(action stow.Action, packages []string) error {
	opts, err := resolveStowOpts()
	if err != nil {
		return err
	}
	out, err := stow.Run(opts, action, packages, stowDryRun)
	if out != "" {
		fmt.Println(out)
	}
	if err != nil {
		return err
	}
	if stowDryRun {
		fmt.Fprintf(os.Stderr, "dry-run: %s %s\n", action, strings.Join(packages, " "))
	}
	return nil
}

func init() {
	stowCmd.PersistentFlags().BoolVarP(&stowDryRun, "dry-run", "n", false, "pass -n to stow")
	stowListCmd.Flags().BoolVar(&stowJSON, "json", false, "JSON output")
	stowCmd.AddCommand(stowListCmd, stowLinkCmd, stowUnlinkCmd, stowRestowCmd)
	rootCmd.AddCommand(stowCmd)
}
