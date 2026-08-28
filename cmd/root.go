package cmd

import (
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/spf13/cobra"

	"github.com/vegarringdal/dotr/internal/tui"
)

var rootCmd = &cobra.Command{
	Use:   "dotr",
	Short: "Browse and edit local config files",
	Long: `dotr is a lightweight TUI for browsing configs under $HOME and $XDG_CONFIG_HOME.

Run with no arguments to open the interactive TUI.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runTUI()
	},
}

func runTUI() error {
	p := tea.NewProgram(tui.New())
	_, err := p.Run()
	return err
}

// Execute runs the CLI.
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.SilenceUsage = true
	rootCmd.CompletionOptions.DisableDefaultCmd = false
}
