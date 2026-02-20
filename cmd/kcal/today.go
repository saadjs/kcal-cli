package kcal

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var todayDate string

var todayCmd = &cobra.Command{
	Use:   "today",
	Short: "Show today's intake, exercise, and goal progress",
	RunE: func(cmd *cobra.Command, args []string) error {
		target := time.Now()
		if todayDate != "" {
			parsed, err := time.ParseInLocation("2006-01-02", todayDate, time.Local)
			if err != nil {
				return fmt.Errorf("invalid --date %q (expected YYYY-MM-DD)", todayDate)
			}
			target = parsed
		}
		return withDB(func(sqldb *sql.DB) error {
			status, err := service.TodaySummary(sqldb, target)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Date: %s\n", status.Date)
			fmt.Fprintf(cmd.OutOrStdout(), "Intake: %d kcal\n", status.IntakeCalories)
			fmt.Fprintf(cmd.OutOrStdout(), "Exercise: %d kcal\n", status.ExerciseCalories)
			fmt.Fprintf(cmd.OutOrStdout(), "Net: %d kcal\n", status.NetCalories)
			fmt.Fprintf(cmd.OutOrStdout(), "Macros: P %.1fg | C %.1fg | F %.1fg\n", status.ProteinG, status.CarbsG, status.FatG)
			if status.HasGoal {
				fmt.Fprintf(cmd.OutOrStdout(), "Goal: %d kcal | P %.1fg | C %.1fg | F %.1fg\n", status.GoalCalories, status.GoalProteinG, status.GoalCarbsG, status.GoalFatG)
				fmt.Fprintf(cmd.OutOrStdout(), "Remaining: %d kcal | P %.1fg | C %.1fg | F %.1fg\n", status.RemainingCalories, status.RemainingProteinG, status.RemainingCarbsG, status.RemainingFatG)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), "Goal: not set")
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(todayCmd)
	todayCmd.Flags().StringVar(&todayDate, "date", "", "Date YYYY-MM-DD (default today)")
}
