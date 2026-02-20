package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var goalCmd = &cobra.Command{
	Use:   "goal",
	Short: "Manage daily calorie and macro goals",
}

var (
	goalCalories int
	goalProtein  float64
	goalCarbs    float64
	goalFat      float64
	goalDate     string
)

var goalSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set daily goals with an effective date",
	RunE: func(cmd *cobra.Command, args []string) error {
		in := service.SetGoalInput{
			Calories:      goalCalories,
			ProteinG:      goalProtein,
			CarbsG:        goalCarbs,
			FatG:          goalFat,
			EffectiveDate: goalDate,
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.SetGoal(sqldb, in); err != nil {
				return err
			}
			if in.EffectiveDate == "" {
				in.EffectiveDate = "today"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set goal effective %s\n", in.EffectiveDate)
			return nil
		})
	},
}

var currentGoalDate string

var goalCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current goal",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			goal, err := service.CurrentGoal(sqldb, currentGoalDate)
			if err != nil {
				return err
			}
			if goal == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "No goal configured")
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Effective: %s\nCalories: %d\nProtein: %.1fg\nCarbs: %.1fg\nFat: %.1fg\n", goal.EffectiveDate, goal.Calories, goal.ProteinG, goal.CarbsG, goal.FatG)
			return nil
		})
	},
}

var goalHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show goal history",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			goals, err := service.GoalHistory(sqldb)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "DATE\tKCAL\tP\tC\tF")
			for _, g := range goals {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%d\t%.1f\t%.1f\t%.1f\n", g.EffectiveDate, g.Calories, g.ProteinG, g.CarbsG, g.FatG)
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(goalCmd)
	goalCmd.AddCommand(goalSetCmd, goalCurrentCmd, goalHistoryCmd)

	goalSetCmd.Flags().IntVar(&goalCalories, "calories", 0, "Daily calorie target")
	goalSetCmd.Flags().Float64Var(&goalProtein, "protein", 0, "Daily protein target grams")
	goalSetCmd.Flags().Float64Var(&goalCarbs, "carbs", 0, "Daily carbs target grams")
	goalSetCmd.Flags().Float64Var(&goalFat, "fat", 0, "Daily fat target grams")
	goalSetCmd.Flags().StringVar(&goalDate, "effective-date", "", "Effective date YYYY-MM-DD (default today)")
	_ = goalSetCmd.MarkFlagRequired("calories")
	_ = goalSetCmd.MarkFlagRequired("protein")
	_ = goalSetCmd.MarkFlagRequired("carbs")
	_ = goalSetCmd.MarkFlagRequired("fat")

	goalCurrentCmd.Flags().StringVar(&currentGoalDate, "date", "", "Resolve goal at date YYYY-MM-DD (default today)")
}
