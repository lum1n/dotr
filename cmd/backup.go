package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/vegarringdal/dotr/internal/backup"
	"github.com/vegarringdal/dotr/internal/find"
	"github.com/vegarringdal/dotr/internal/load"
)

var backupCmd = &cobra.Command{
	Use:   "backup <query>",
	Short: "Create a backup snapshot of a matching config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := load.All()
		if err != nil {
			return err
		}
		e, ok := find.Best(res.Entries, args[0])
		if !ok {
			return fmt.Errorf("no match for %q", args[0])
		}
		snap, err := backup.Create(e.AbsPath, res.Config.BackupKeep)
		if err != nil {
			return err
		}
		fmt.Printf("backed up %s → %s\n", find.Display(e), snap.Path)
		return nil
	},
}

var restoreCmd = &cobra.Command{
	Use:   "restore <query>",
	Short: "Restore the latest backup of a matching config",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		res, err := load.All()
		if err != nil {
			return err
		}
		e, ok := find.Best(res.Entries, args[0])
		if !ok {
			return fmt.Errorf("no match for %q", args[0])
		}
		snaps, err := backup.List(e.AbsPath)
		if err != nil {
			return err
		}
		if len(snaps) == 0 {
			return fmt.Errorf("no backups for %s", find.Display(e))
		}
		if err := backup.Restore(snaps[0]); err != nil {
			return err
		}
		fmt.Printf("restored %s from %s\n", find.Display(e), snaps[0].Name)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	rootCmd.AddCommand(restoreCmd)
}
