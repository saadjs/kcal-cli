package kcal

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

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

var (
	suggestWeight       float64
	suggestWeightUnit   string
	suggestMaintenance  int
	suggestPace         string
	suggestProteinPerKg float64
	suggestApply        bool
	suggestDate         string
)

var goalSuggestCmd = &cobra.Command{
	Use:   "suggest",
	Short: "Suggest a maintain/cut/bulk goal",
	RunE: func(cmd *cobra.Command, args []string) error {
		if suggestWeight <= 0 {
			return fmt.Errorf("--weight must be > 0")
		}
		if suggestMaintenance <= 0 {
			return fmt.Errorf("--maintenance-calories must be > 0")
		}
		if suggestProteinPerKg <= 0 {
			return fmt.Errorf("--protein-per-kg must be > 0")
		}
		weightKg, err := service.ToKg(suggestWeight, suggestWeightUnit)
		if err != nil {
			return err
		}
		targetCalories := suggestMaintenance
		switch strings.ToLower(strings.TrimSpace(suggestPace)) {
		case "maintain":
		case "cut":
			targetCalories -= 500
		case "bulk":
			targetCalories += 300
		default:
			return fmt.Errorf("--pace must be maintain, cut, or bulk")
		}
		protein := weightKg * suggestProteinPerKg
		fat := weightKg * 0.8
		remainingCalories := float64(targetCalories) - ((protein * 4) + (fat * 9))
		if remainingCalories < 0 {
			return fmt.Errorf("calorie target too low for selected protein/fat heuristics")
		}
		carbs := remainingCalories / 4

		fmt.Fprintf(cmd.OutOrStdout(), "Suggested goal (%s):\n", strings.ToLower(strings.TrimSpace(suggestPace)))
		fmt.Fprintf(cmd.OutOrStdout(), "Calories: %d\nProtein: %.1fg\nCarbs: %.1fg\nFat: %.1fg\n", targetCalories, protein, carbs, fat)

		if !suggestApply {
			return nil
		}
		effectiveDate := strings.TrimSpace(suggestDate)
		if effectiveDate == "" {
			effectiveDate = time.Now().Format("2006-01-02")
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.SetGoal(sqldb, service.SetGoalInput{
				Calories:      targetCalories,
				ProteinG:      protein,
				CarbsG:        carbs,
				FatG:          fat,
				EffectiveDate: effectiveDate,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Applied suggested goal effective %s\n", effectiveDate)
			return nil
		})
	},
}

func init() {
	rootCmd.AddCommand(goalCmd)
	goalCmd.AddCommand(goalSetCmd, goalCurrentCmd, goalHistoryCmd, goalSuggestCmd)

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

	goalSuggestCmd.Flags().Float64Var(&suggestWeight, "weight", 0, "Body weight for macro heuristic")
	goalSuggestCmd.Flags().StringVar(&suggestWeightUnit, "unit", "kg", "Weight unit: kg or lb")
	goalSuggestCmd.Flags().IntVar(&suggestMaintenance, "maintenance-calories", 0, "Estimated maintenance calories")
	goalSuggestCmd.Flags().StringVar(&suggestPace, "pace", "maintain", "Target pace: maintain, cut, bulk")
	goalSuggestCmd.Flags().Float64Var(&suggestProteinPerKg, "protein-per-kg", 2.0, "Protein heuristic grams per kg")
	goalSuggestCmd.Flags().BoolVar(&suggestApply, "apply", false, "Apply suggestion as a goal")
	goalSuggestCmd.Flags().StringVar(&suggestDate, "effective-date", "", "Effective date YYYY-MM-DD (default today)")
}
