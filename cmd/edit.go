package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/charmbracelet/x/editor"
	"github.com/spf13/cobra"

	"github.com/lum1n/dotr/internal/find"
	"github.com/lum1n/dotr/internal/load"
)

var editAll bool

var editCmd = &cobra.Command{
	Use:   "edit <query>",
	Short: "Open a matching config in $EDITOR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := load.All()
		if err != nil {
			return err
		}
		matches := find.Match(res.Entries, args[0])
		if len(matches) == 0 {
			return fmt.Errorf("no match for %q", args[0])
		}
		if len(matches) > 1 && !editAll {
			best, ok := find.Best(res.Entries, args[0])
			if !ok {
				return fmt.Errorf("no match for %q", args[0])
			}
			fmt.Fprintf(os.Stderr, "dotr: %d matches; opening best %s (use --all to print)\n", len(matches), find.Display(best))
			return runEditor(best.AbsPath)
		}
		if editAll && len(matches) > 1 {
			for _, e := range matches {
				fmt.Println(e.AbsPath)
			}
			return fmt.Errorf("%d matches; refine query or omit --all to open best", len(matches))
		}
		return runEditor(matches[0].AbsPath)
	},
}

func runEditor(path string) error {
	c, err := editor.Cmd("dotr", path)
	if err != nil {
		return err
	}
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	if err := c.Run(); err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			os.Exit(ee.ExitCode())
		}
		return err
	}
	return nil
}

func init() {
	editCmd.Flags().BoolVar(&editAll, "all", false, "list all matches instead of opening best")
	rootCmd.AddCommand(editCmd)
}
