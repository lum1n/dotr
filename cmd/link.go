package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/lum1n/dotr/internal/symlink"
)

var linkCmd = &cobra.Command{
	Use:   "link <path> <target>",
	Short: "Create a symlink at path pointing to target",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		link := symlink.ExpandTilde(args[0], home)
		target := symlink.ExpandTilde(args[1], home)
		if err := symlink.Create(link, target); err != nil {
			return err
		}
		fmt.Printf("%s → %s\n", link, target)
		return nil
	},
}

var unlinkCmd = &cobra.Command{
	Use:   "unlink <path>",
	Short: "Remove a symlink (refuses regular files)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path := symlink.ExpandTilde(args[0], home)
		info, err := symlink.Read(path)
		if err != nil {
			return err
		}
		if err := symlink.Remove(path); err != nil {
			return err
		}
		fmt.Printf("removed symlink %s (was → %s)\n", path, info.Target)
		return nil
	},
}

var retargetCmd = &cobra.Command{
	Use:   "retarget <path> <new-target>",
	Short: "Change where an existing symlink points",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		home, err := os.UserHomeDir()
		if err != nil {
			return err
		}
		path := symlink.ExpandTilde(args[0], home)
		target := symlink.ExpandTilde(args[1], home)
		if err := symlink.Retarget(path, target); err != nil {
			return err
		}
		fmt.Printf("%s → %s\n", path, target)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(linkCmd, unlinkCmd, retargetCmd)
}
