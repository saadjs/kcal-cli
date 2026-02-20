package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saadjs/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var bodyGoalCmd = &cobra.Command{
	Use:   "body-goal",
	Short: "Manage body composition goals",
}

var (
	goalWeightValue float64
	goalWeightUnit  string
	goalBodyFat     float64
	goalTargetDate  string
	goalEffective   string
)

var bodyGoalSetCmd = &cobra.Command{
	Use:   "set",
	Short: "Set body goal with effective date",
	RunE: func(cmd *cobra.Command, args []string) error {
		in := service.SetBodyGoalInput{
			TargetWeight:  goalWeightValue,
			Unit:          goalWeightUnit,
			TargetBodyFat: optionalBodyFat(goalBodyFat),
			TargetDate:    goalTargetDate,
			EffectiveDate: goalEffective,
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.SetBodyGoal(sqldb, in); err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "Set body goal")
			return nil
		})
	},
}

var bodyGoalCurrentDate string

var bodyGoalCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show current body goal",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			goal, err := service.CurrentBodyGoal(sqldb, bodyGoalCurrentDate)
			if err != nil {
				return err
			}
			if goal == nil {
				fmt.Fprintln(cmd.OutOrStdout(), "No body goal configured")
				return nil
			}
			weight, err := service.WeightFromKg(goal.TargetWeightKg, goalWeightUnit)
			if err != nil {
				return err
			}
			bf := ""
			if goal.TargetBodyFatPct != nil {
				bf = fmt.Sprintf("%.2f%%", *goal.TargetBodyFatPct)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Effective: %s\nTarget Weight: %.2f %s\nTarget Body Fat: %s\nTarget Date: %s\n", goal.EffectiveDate, weight, goalWeightUnit, bf, goal.TargetDate)
			return nil
		})
	},
}

var bodyGoalHistoryCmd = &cobra.Command{
	Use:   "history",
	Short: "Show body goal history",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.BodyGoalHistory(sqldb)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "EFFECTIVE\tTARGET_WEIGHT\tUNIT\tTARGET_BODY_FAT\tTARGET_DATE")
			for _, it := range items {
				w, err := service.WeightFromKg(it.TargetWeightKg, goalWeightUnit)
				if err != nil {
					return err
				}
				bf := ""
				if it.TargetBodyFatPct != nil {
					bf = fmt.Sprintf("%.2f", *it.TargetBodyFatPct)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%.2f\t%s\t%s\t%s\n", it.EffectiveDate, w, goalWeightUnit, bf, it.TargetDate)
			}
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(bodyGoalCmd)
	bodyGoalCmd.AddCommand(bodyGoalSetCmd, bodyGoalCurrentCmd, bodyGoalHistoryCmd)

	bodyGoalSetCmd.Flags().Float64Var(&goalWeightValue, "target-weight", 0, "Target weight value")
	bodyGoalSetCmd.Flags().StringVar(&goalWeightUnit, "unit", "kg", "Weight unit: kg or lb")
	bodyGoalSetCmd.Flags().Float64Var(&goalBodyFat, "target-body-fat", -1, "Target body-fat percentage (optional)")
	bodyGoalSetCmd.Flags().StringVar(&goalTargetDate, "target-date", "", "Target date YYYY-MM-DD")
	bodyGoalSetCmd.Flags().StringVar(&goalEffective, "effective-date", "", "Effective date YYYY-MM-DD (default today)")
	_ = bodyGoalSetCmd.MarkFlagRequired("target-weight")

	bodyGoalCurrentCmd.Flags().StringVar(&bodyGoalCurrentDate, "date", "", "Resolve goal at date YYYY-MM-DD")
	bodyGoalCurrentCmd.Flags().StringVar(&goalWeightUnit, "unit", "kg", "Weight unit: kg or lb")
	bodyGoalHistoryCmd.Flags().StringVar(&goalWeightUnit, "unit", "kg", "Weight unit: kg or lb")
}
