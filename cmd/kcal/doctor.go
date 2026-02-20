package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var doctorFix bool

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Run data integrity checks",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			report, err := service.RunDoctor(sqldb, doctorFix)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Orphan entries: %d\n", report.OrphanEntries)
			fmt.Fprintf(cmd.OutOrStdout(), "Invalid metadata rows: %d\n", report.InvalidMetadata)
			fmt.Fprintf(cmd.OutOrStdout(), "Duplicate entry rows: %d\n", report.DuplicateEntryRows)
			if doctorFix {
				fmt.Fprintf(cmd.OutOrStdout(), "Fixed metadata rows: %d\n", report.FixedMetadataRows)
				// Re-check after fixes so exit status reflects final state.
				report, err = service.RunDoctor(sqldb, false)
				if err != nil {
					return err
				}
			}
			if report.OrphanEntries > 0 || report.InvalidMetadata > 0 || report.DuplicateEntryRows > 0 {
				return fmt.Errorf("doctor found integrity issues")
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Attempt safe auto-fixes")
}
