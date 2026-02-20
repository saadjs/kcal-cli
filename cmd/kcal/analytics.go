package kcal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"time"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var analyticsCmd = &cobra.Command{
	Use:   "analytics",
	Short: "View weekly, monthly, and range analytics",
}

var (
	analyticsJSON      bool
	analyticsTolerance float64
)

var weekArg string

var analyticsWeekCmd = &cobra.Command{
	Use:   "week",
	Short: "Weekly trend analytics",
	RunE: func(cmd *cobra.Command, args []string) error {
		start, end, err := resolveWeekRange(weekArg)
		if err != nil {
			return err
		}
		return runAnalytics(cmd, start, end)
	},
}

var monthArg string

var analyticsMonthCmd = &cobra.Command{
	Use:   "month",
	Short: "Monthly trend analytics",
	RunE: func(cmd *cobra.Command, args []string) error {
		start, end, err := resolveMonthRange(monthArg)
		if err != nil {
			return err
		}
		return runAnalytics(cmd, start, end)
	},
}

var (
	rangeFrom string
	rangeTo   string
)

var analyticsRangeCmd = &cobra.Command{
	Use:   "range",
	Short: "Range trend analytics",
	RunE: func(cmd *cobra.Command, args []string) error {
		if rangeFrom == "" || rangeTo == "" {
			return fmt.Errorf("--from and --to are required")
		}
		start, err := time.ParseInLocation("2006-01-02", rangeFrom, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --from date (expected YYYY-MM-DD)")
		}
		end, err := time.ParseInLocation("2006-01-02", rangeTo, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --to date (expected YYYY-MM-DD)")
		}
		return runAnalytics(cmd, start, end)
	},
}

func runAnalytics(cmd *cobra.Command, from, to time.Time) error {
	return withDB(func(sqldb *sql.DB) error {
		report, err := service.AnalyticsRange(sqldb, from, to, analyticsTolerance)
		if err != nil {
			return err
		}
		if analyticsJSON {
			b, err := json.MarshalIndent(report, "", "  ")
			if err != nil {
				return fmt.Errorf("marshal analytics json: %w", err)
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(b))
			return nil
		}
		printAnalyticsTable(cmd, report)
		return nil
	})
}

func printAnalyticsTable(cmd *cobra.Command, r *service.AnalyticsReport) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Range: %s to %s\n", r.FromDate, r.ToDate)
	fmt.Fprintf(out, "Totals: kcal=%d P=%.1f C=%.1f F=%.1f\n", r.TotalCalories, r.TotalProtein, r.TotalCarbs, r.TotalFat)
	fmt.Fprintf(out, "Averages/day: kcal=%.1f P=%.1f C=%.1f F=%.1f\n", r.AverageCaloriesPerDay, r.AverageProteinPerDay, r.AverageCarbsPerDay, r.AverageFatPerDay)
	if r.HighestDay != nil && r.LowestDay != nil {
		fmt.Fprintf(out, "Highest day: %s (%d kcal)\n", r.HighestDay.Date, r.HighestDay.Calories)
		fmt.Fprintf(out, "Lowest day: %s (%d kcal)\n", r.LowestDay.Date, r.LowestDay.Calories)
	}
	fmt.Fprintf(out, "Adherence: %d/%d days within goals (%.1f%%), %d days without goal\n", r.Adherence.WithinGoalDays, r.Adherence.EvaluatedDays, r.Adherence.PercentWithin, r.Adherence.SkippedGoalDays)

	fmt.Fprintln(out, "\nBy Category")
	fmt.Fprintln(out, "CATEGORY\tKCAL\tP\tC\tF")
	for _, c := range r.ByCategory {
		fmt.Fprintf(out, "%s\t%d\t%.1f\t%.1f\t%.1f\n", c.Category, c.Calories, c.Protein, c.Carbs, c.Fat)
	}

	fmt.Fprintln(out, "\nSources")
	for source, count := range r.Metadata.SourceCounts {
		fmt.Fprintf(out, "%s: %d\n", source, count)
	}
	if len(r.Metadata.BarcodeTierCounts) > 0 {
		fmt.Fprintln(out, "Barcode tiers:")
		for tier, count := range r.Metadata.BarcodeTierCounts {
			fmt.Fprintf(out, "  %s: %d\n", tier, count)
		}
	}
	if r.Metadata.Confidence.Count > 0 {
		fmt.Fprintf(out, "Confidence: n=%d avg=%.2f min=%.2f max=%.2f\n", r.Metadata.Confidence.Count, r.Metadata.Confidence.Avg, r.Metadata.Confidence.Min, r.Metadata.Confidence.Max)
	}

	fmt.Fprintln(out, "\nBody")
	fmt.Fprintf(out, "Measurements: %d\n", r.Body.MeasurementsCount)
	if r.Body.MeasurementsCount > 0 {
		fmt.Fprintf(out, "Weight: start=%.2fkg end=%.2fkg change=%.2fkg\n", r.Body.StartWeightKg, r.Body.EndWeightKg, r.Body.WeightChangeKg)
		if r.Body.AvgWeeklyChangeKg != 0 {
			fmt.Fprintf(out, "Avg weekly weight change: %.2fkg\n", r.Body.AvgWeeklyChangeKg)
		}
		if r.Body.StartBodyFatPct != nil && r.Body.EndBodyFatPct != nil {
			fmt.Fprintf(out, "Body fat: start=%.2f%% end=%.2f%% change=%.2f%%\n", *r.Body.StartBodyFatPct, *r.Body.EndBodyFatPct, *r.Body.BodyFatChangePct)
		}
		if r.Body.StartLeanMassKg != nil && r.Body.EndLeanMassKg != nil {
			fmt.Fprintf(out, "Lean mass: start=%.2fkg end=%.2fkg change=%.2fkg\n", *r.Body.StartLeanMassKg, *r.Body.EndLeanMassKg, *r.Body.LeanMassChangeKg)
		}
		if r.Body.GoalProgress != nil {
			fmt.Fprintf(out, "Body goal progress: target %.2fkg, latest %.2fkg (delta %.2fkg)\n", r.Body.GoalProgress.TargetWeightKg, r.Body.GoalProgress.LatestWeightKg, r.Body.GoalProgress.WeightDeltaKg)
		}
	}
}

func resolveWeekRange(week string) (time.Time, time.Time, error) {
	if week == "" {
		now := time.Now().In(time.Local)
		start := beginningOfWeek(now)
		return start, start.AddDate(0, 0, 6), nil
	}
	if !regexp.MustCompile(`^\d{4}-W\d{2}$`).MatchString(week) {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --week value %q (expected YYYY-Www)", week)
	}
	var year, weekNum int
	if _, err := fmt.Sscanf(week, "%4d-W%2d", &year, &weekNum); err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --week value %q (expected YYYY-Www)", week)
	}
	if weekNum < 1 {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --week value %q (week must be between 01 and %02d for %d)", week, weeksInISOYear(year), year)
	}
	maxWeek := weeksInISOYear(year)
	if weekNum > maxWeek {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --week value %q (week must be between 01 and %02d for %d)", week, maxWeek, year)
	}
	start := isoWeekStart(year, weekNum)
	return start, start.AddDate(0, 0, 6), nil
}

func resolveMonthRange(month string) (time.Time, time.Time, error) {
	if month == "" {
		now := time.Now().In(time.Local)
		start := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.Local)
		end := start.AddDate(0, 1, -1)
		return start, end, nil
	}
	parsed, err := time.ParseInLocation("2006-01", month, time.Local)
	if err != nil {
		return time.Time{}, time.Time{}, fmt.Errorf("invalid --month value %q (expected YYYY-MM)", month)
	}
	start := time.Date(parsed.Year(), parsed.Month(), 1, 0, 0, 0, 0, time.Local)
	end := start.AddDate(0, 1, -1)
	return start, end, nil
}

func beginningOfWeek(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
}

func isoWeekStart(year, week int) time.Time {
	jan4 := time.Date(year, 1, 4, 0, 0, 0, 0, time.Local)
	weekday := int(jan4.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	week1Monday := jan4.AddDate(0, 0, -(weekday - 1))
	return week1Monday.AddDate(0, 0, (week-1)*7)
}

func weeksInISOYear(year int) int {
	_, wk := time.Date(year, 12, 28, 0, 0, 0, 0, time.Local).ISOWeek()
	return wk
}

func init() {
	rootCmd.AddCommand(analyticsCmd)
	analyticsCmd.AddCommand(analyticsWeekCmd, analyticsMonthCmd, analyticsRangeCmd)

	for _, c := range []*cobra.Command{analyticsWeekCmd, analyticsMonthCmd, analyticsRangeCmd} {
		c.Flags().BoolVar(&analyticsJSON, "json", false, "Output as JSON")
		c.Flags().Float64Var(&analyticsTolerance, "tolerance", 0.10, "Macro adherence tolerance (0.10 = 10%)")
	}
	analyticsWeekCmd.Flags().StringVar(&weekArg, "week", "", "ISO week in format YYYY-Www")
	analyticsMonthCmd.Flags().StringVar(&monthArg, "month", "", "Month in format YYYY-MM")
	analyticsRangeCmd.Flags().StringVar(&rangeFrom, "from", "", "Start date YYYY-MM-DD")
	analyticsRangeCmd.Flags().StringVar(&rangeTo, "to", "", "End date YYYY-MM-DD")
}
