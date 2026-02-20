package kcal

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var backupCmd = &cobra.Command{
	Use:   "backup",
	Short: "Manage database backups",
}

var (
	backupOut    string
	backupDir    string
	restoreFile  string
	restoreForce bool
)

var backupCreateCmd = &cobra.Command{
	Use:   "create",
	Short: "Create database backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := resolveDBPath()
		if err != nil {
			return err
		}
		out := backupOut
		if out == "" {
			dir := backupDir
			if dir == "" {
				dir = filepath.Join(filepath.Dir(db), "backups")
			}
			out = filepath.Join(dir, fmt.Sprintf("kcal-%s.db", time.Now().Format("20060102-150405")))
		}
		info, err := service.CreateBackup(db, out)
		if err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Created backup: %s\n", info.Path)
		fmt.Fprintf(cmd.OutOrStdout(), "Checksum: %s\n", info.Checksum)
		return nil
	},
}

var backupListCmd = &cobra.Command{
	Use:   "list",
	Short: "List backups",
	RunE: func(cmd *cobra.Command, args []string) error {
		db, err := resolveDBPath()
		if err != nil {
			return err
		}
		dir := backupDir
		if dir == "" {
			dir = filepath.Join(filepath.Dir(db), "backups")
		}
		items, err := service.ListBackups(dir)
		if err != nil {
			return err
		}
		fmt.Fprintln(cmd.OutOrStdout(), "FILE\tSIZE\tCREATED\tCHECKSUM")
		for _, it := range items {
			fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%s\t%s\n", it.Path, it.SizeBytes, it.CreatedAt.Format(time.RFC3339), it.Checksum)
		}
		return nil
	},
}

var backupRestoreCmd = &cobra.Command{
	Use:   "restore",
	Short: "Restore database from backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		if restoreFile == "" {
			return fmt.Errorf("--file is required")
		}
		db, err := resolveDBPath()
		if err != nil {
			return err
		}
		if err := service.RestoreBackup(restoreFile, db, restoreForce); err != nil {
			return err
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Restored backup from %s\n", restoreFile)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(backupCmd)
	backupCmd.AddCommand(backupCreateCmd, backupListCmd, backupRestoreCmd)

	backupCreateCmd.Flags().StringVar(&backupOut, "out", "", "Backup output file path")
	backupCreateCmd.Flags().StringVar(&backupDir, "dir", "", "Backup directory (used when --out is empty)")
	backupListCmd.Flags().StringVar(&backupDir, "dir", "", "Backup directory (default: alongside DB under backups/) ")
	backupRestoreCmd.Flags().StringVar(&restoreFile, "file", "", "Backup .db file path")
	backupRestoreCmd.Flags().BoolVar(&restoreForce, "force", false, "Overwrite existing DB if present")
}
