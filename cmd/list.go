package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/spf13/cobra"

	"github.com/lum1n/dotr/internal/find"
	"github.com/lum1n/dotr/internal/gitstatus"
	"github.com/lum1n/dotr/internal/load"
)

var listJSON bool
var listGit bool

var listCmd = &cobra.Command{
	Use:   "list [query]",
	Short: "List discovered config files",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := load.All()
		if err != nil {
			return err
		}
		if res.Truncated {
			fmt.Fprintf(os.Stderr, "warning: scan truncated at %d entries\n", len(res.Entries))
		}
		entries := res.Entries
		if len(args) == 1 {
			entries = find.Match(entries, args[0])
		}

		var git map[string]gitstatus.Kind
		if listGit {
			paths := make([]string, len(entries))
			for i, e := range entries {
				paths[i] = e.AbsPath
			}
			git = gitstatus.Map(paths)
		}

		if listJSON {
			type row struct {
				App     string `json:"app"`
				RelPath string `json:"relpath"`
				Path    string `json:"path"`
				Size    int64  `json:"size"`
				Symlink bool   `json:"symlink"`
				Git     string `json:"git,omitempty"`
			}
			out := make([]row, 0, len(entries))
			for _, e := range entries {
				r := row{App: e.App, RelPath: e.RelPath, Path: e.AbsPath, Size: e.Size, Symlink: e.Symlink}
				if git != nil {
					r.Git = git[e.AbsPath].String()
				}
				out = append(out, r)
			}
			enc := json.NewEncoder(os.Stdout)
			enc.SetIndent("", "  ")
			return enc.Encode(out)
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		for _, e := range entries {
			mark := " "
			if e.Symlink {
				mark = "→"
			}
			g := ""
			if git != nil {
				if m := git[e.AbsPath].Mark(); m != "" {
					g = m + " "
				}
			}
			fmt.Fprintf(w, "%s%s\t%s\t%d\n", g, mark, find.Display(e), e.Size)
		}
		return w.Flush()
	},
}

func init() {
	listCmd.Flags().BoolVar(&listJSON, "json", false, "JSON output")
	listCmd.Flags().BoolVar(&listGit, "git", false, "include git status marks")
	rootCmd.AddCommand(listCmd)
}
