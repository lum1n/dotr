package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/lum1n/dotr/internal/config"
	"github.com/lum1n/dotr/internal/ignore"
)

var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Edit dotr config.yaml in $EDITOR",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := config.EnsureFile()
		if err != nil {
			return err
		}
		fmt.Println(path)
		return runEditor(path)
	},
}

var ignoreList bool
var ignoreAdd string

var ignoreCmd = &cobra.Command{
	Use:   "ignore",
	Short: "Manage ignore patterns",
	Long:  "With no flags, opens the ignore file in $EDITOR. Use --list or --add.",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		list, err := ignore.Load()
		if err != nil {
			return err
		}
		if ignoreAdd != "" {
			if !list.Add(ignoreAdd) {
				return fmt.Errorf("already ignored: %s", ignoreAdd)
			}
			if err := list.Save(); err != nil {
				return err
			}
			fmt.Println("added", ignoreAdd)
			return nil
		}
		if ignoreList {
			for _, p := range list.Patterns {
				fmt.Println(p)
			}
			return nil
		}
		path, err := ignore.Path()
		if err != nil {
			return err
		}
		_ = list.Save() // ensure file exists
		fmt.Println(path)
		return runEditor(path)
	},
}

func init() {
	ignoreCmd.Flags().BoolVar(&ignoreList, "list", false, "print ignore patterns")
	ignoreCmd.Flags().StringVar(&ignoreAdd, "add", "", "add an ignore pattern")
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(ignoreCmd)
}
