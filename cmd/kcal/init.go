package kcal

import (
	"fmt"

	"github.com/saad/kcal-cli/internal/app"
	"github.com/saad/kcal-cli/internal/db"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize local kcal database",
	RunE: func(cmd *cobra.Command, args []string) error {
		path, err := resolveDBPath()
		if err != nil {
			return err
		}
		if err := app.EnsureDBDir(path); err != nil {
			return err
		}

		sqldb, err := db.Open(path)
		if err != nil {
			return err
		}
		defer sqldb.Close()

		if err := db.ApplyMigrations(sqldb); err != nil {
			return err
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Initialized kcal database at %s\n", path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(initCmd)
}

func resolveDBPath() (string, error) {
	if dbPath != "" {
		return dbPath, nil
	}
	return app.DefaultDBPath()
}
