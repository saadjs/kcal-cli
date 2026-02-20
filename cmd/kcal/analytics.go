package kcal

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
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

var analyticsInsightsCmd = &cobra.Command{
	Use:   "insights",
	Short: "Premium-style insights with trends and charts",
}

var (
	insightsJSON        bool
	insightsTolerance   float64
	insightsGranularity string
	insightsNoCharts    bool
	insightsOutPath     string
	insightsOutFormat   string
)

var insightsWeekArg string

var analyticsInsightsWeekCmd = &cobra.Command{
	Use:   "week",
	Short: "Weekly premium insights",
	RunE: func(cmd *cobra.Command, args []string) error {
		start, end, err := resolveWeekRange(insightsWeekArg)
		if err != nil {
			return err
		}
		return runAnalyticsInsights(cmd, start, end)
	},
}

var insightsMonthArg string

var analyticsInsightsMonthCmd = &cobra.Command{
	Use:   "month",
	Short: "Monthly premium insights",
	RunE: func(cmd *cobra.Command, args []string) error {
		start, end, err := resolveMonthRange(insightsMonthArg)
		if err != nil {
			return err
		}
		return runAnalyticsInsights(cmd, start, end)
	},
}

var (
	insightsRangeFrom string
	insightsRangeTo   string
)

var analyticsInsightsRangeCmd = &cobra.Command{
	Use:   "range",
	Short: "Range premium insights",
	RunE: func(cmd *cobra.Command, args []string) error {
		if insightsRangeFrom == "" || insightsRangeTo == "" {
			return fmt.Errorf("--from and --to are required")
		}
		start, err := time.ParseInLocation("2006-01-02", insightsRangeFrom, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --from date (expected YYYY-MM-DD)")
		}
		end, err := time.ParseInLocation("2006-01-02", insightsRangeTo, time.Local)
		if err != nil {
			return fmt.Errorf("invalid --to date (expected YYYY-MM-DD)")
		}
		return runAnalyticsInsights(cmd, start, end)
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

func runAnalyticsInsights(cmd *cobra.Command, from, to time.Time) error {
	granularity, err := parseInsightsGranularity(insightsGranularity)
	if err != nil {
		return err
	}
	return withDB(func(sqldb *sql.DB) error {
		report, err := service.AnalyticsInsightsRange(sqldb, from, to, insightsTolerance, granularity)
		if err != nil {
			return err
		}

		jsonBytes, err := json.MarshalIndent(report, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal analytics insights json: %w", err)
		}

		if insightsOutPath != "" {
			data, err := renderAnalyticsInsightsExport(report, jsonBytes, insightsNoCharts, insightsOutFormat)
			if err != nil {
				return err
			}
			if err := os.WriteFile(insightsOutPath, data, 0o644); err != nil {
				return fmt.Errorf("write insights report to %q: %w", insightsOutPath, err)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Saved insights report to %s\n", insightsOutPath)
		}

		if insightsJSON {
			fmt.Fprintln(cmd.OutOrStdout(), string(jsonBytes))
			return nil
		}
		printAnalyticsInsightsTable(cmd.OutOrStdout(), report, insightsNoCharts)
		return nil
	})
}

func printAnalyticsTable(cmd *cobra.Command, r *service.AnalyticsReport) {
	out := cmd.OutOrStdout()
	fmt.Fprintf(out, "Range: %s to %s\n", r.FromDate, r.ToDate)
	fmt.Fprintf(out, "Totals: intake=%d exercise=%d net=%d P=%.1f C=%.1f F=%.1f\n", r.TotalIntakeCalories, r.TotalExerciseCalories, r.TotalNetCalories, r.TotalProtein, r.TotalCarbs, r.TotalFat)
	fmt.Fprintf(out, "Averages/day: intake=%.1f exercise=%.1f net=%.1f P=%.1f C=%.1f F=%.1f\n", r.AverageIntakeCaloriesPerDay, r.AverageExerciseCaloriesPerDay, r.AverageNetCaloriesPerDay, r.AverageProteinPerDay, r.AverageCarbsPerDay, r.AverageFatPerDay)
	if r.HighestDay != nil && r.LowestDay != nil {
		fmt.Fprintf(out, "Highest day: %s (net %d kcal)\n", r.HighestDay.Date, r.HighestDay.NetCalories)
		fmt.Fprintf(out, "Lowest day: %s (net %d kcal)\n", r.LowestDay.Date, r.LowestDay.NetCalories)
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

func printAnalyticsInsightsTable(out anyWriter, r *service.InsightsReport, noCharts bool) {
	fmt.Fprintf(out, "Range: %s to %s\n", r.FromDate, r.ToDate)
	fmt.Fprintf(out, "Previous: %s to %s\n", r.PreviousFromDate, r.PreviousToDate)
	fmt.Fprintf(out, "Granularity: %s\n", r.Granularity)

	fmt.Fprintln(out, "\nKey Metrics")
	fmt.Fprintf(out, "Avg intake/day: %.1f kcal (%s)\n", r.Current.AvgIntakeCaloriesPerDay, formatDelta(r.Deltas.AvgIntakeCaloriesPerDay, "kcal"))
	fmt.Fprintf(out, "Avg exercise/day: %.1f kcal (%s)\n", r.Current.AvgExerciseCaloriesPerDay, formatDelta(r.Deltas.AvgExerciseCaloriesPerDay, "kcal"))
	fmt.Fprintf(out, "Avg net/day: %.1f kcal (%s)\n", r.Current.AvgNetCaloriesPerDay, formatDelta(r.Deltas.AvgNetCaloriesPerDay, "kcal"))

	fmt.Fprintln(out, "\nMacros")
	fmt.Fprintf(out, "Protein/day: %.1fg (%s)\n", r.Current.AvgProteinPerDay, formatDelta(r.Deltas.AvgProteinPerDay, "g"))
	fmt.Fprintf(out, "Carbs/day: %.1fg (%s)\n", r.Current.AvgCarbsPerDay, formatDelta(r.Deltas.AvgCarbsPerDay, "g"))
	fmt.Fprintf(out, "Fat/day: %.1fg (%s)\n", r.Current.AvgFatPerDay, formatDelta(r.Deltas.AvgFatPerDay, "g"))
	fmt.Fprintf(out, "Macro energy split: P %.1f%%, C %.1f%%, F %.1f%%\n", r.MacroBalance.CurrentShare.ProteinPct, r.MacroBalance.CurrentShare.CarbsPct, r.MacroBalance.CurrentShare.FatPct)
	if r.MacroBalance.GoalShare != nil && r.MacroBalance.GoalDelta != nil {
		fmt.Fprintf(out, "Goal split: P %.1f%%, C %.1f%%, F %.1f%%\n", r.MacroBalance.GoalShare.ProteinPct, r.MacroBalance.GoalShare.CarbsPct, r.MacroBalance.GoalShare.FatPct)
		fmt.Fprintf(out, "Split delta (pp): P %+0.1f, C %+0.1f, F %+0.1f\n", r.MacroBalance.GoalDelta.ProteinPctPointDelta, r.MacroBalance.GoalDelta.CarbsPctPointDelta, r.MacroBalance.GoalDelta.FatPctPointDelta)
	}

	fmt.Fprintln(out, "\nAdherence + Activity")
	fmt.Fprintf(out, "Goal adherence: %d/%d (%.1f%%, %s)\n", r.Current.Adherence.WithinGoalDays, r.Current.Adherence.EvaluatedDays, r.Current.Adherence.PercentWithin, formatDelta(r.Deltas.AdherencePercent, "pp"))
	fmt.Fprintf(out, "Intake active days: %d/%d (%.1f%%)\n", r.Current.IntakeActiveDays, r.Current.TotalDays, r.Current.IntakeActiveRate*100)
	fmt.Fprintf(out, "Exercise active days: %d/%d (%.1f%%)\n", r.Current.ExerciseActiveDays, r.Current.TotalDays, r.Current.ExerciseActiveRate*100)

	fmt.Fprintln(out, "\nConsistency")
	fmt.Fprintf(out, "Intake kcal stddev=%.1f cv=%.2f\n", r.Consistency.IntakeCalories.StdDev, r.Consistency.IntakeCalories.CoeffVar)
	fmt.Fprintf(out, "Net kcal stddev=%.1f cv=%.2f\n", r.Consistency.NetCalories.StdDev, r.Consistency.NetCalories.CoeffVar)
	fmt.Fprintf(out, "Protein g stddev=%.1f cv=%.2f\n", r.Consistency.ProteinG.StdDev, r.Consistency.ProteinG.CoeffVar)
	fmt.Fprintf(out, "Carbs g stddev=%.1f cv=%.2f\n", r.Consistency.CarbsG.StdDev, r.Consistency.CarbsG.CoeffVar)
	fmt.Fprintf(out, "Fat g stddev=%.1f cv=%.2f\n", r.Consistency.FatG.StdDev, r.Consistency.FatG.CoeffVar)

	fmt.Fprintln(out, "\nTrends")
	fmt.Fprintf(out, "Intake trend: %s (%.2f kcal/bucket)\n", r.Trends.IntakeCalories.Direction, r.Trends.IntakeCalories.SlopePerBucket)
	fmt.Fprintf(out, "Exercise trend: %s (%.2f kcal/bucket)\n", r.Trends.Exercise.Direction, r.Trends.Exercise.SlopePerBucket)
	fmt.Fprintf(out, "Net trend: %s (%.2f kcal/bucket)\n", r.Trends.NetCalories.Direction, r.Trends.NetCalories.SlopePerBucket)

	fmt.Fprintln(out, "\nExtremes")
	if r.Extremes.HighestNetDay != nil {
		fmt.Fprintf(out, "Highest net day: %s (%d kcal)\n", r.Extremes.HighestNetDay.Date, r.Extremes.HighestNetDay.NetCalories)
	}
	if r.Extremes.LowestNetDay != nil {
		fmt.Fprintf(out, "Lowest net day: %s (%d kcal)\n", r.Extremes.LowestNetDay.Date, r.Extremes.LowestNetDay.NetCalories)
	}
	if r.Extremes.HighestExerciseDay != nil {
		fmt.Fprintf(out, "Highest exercise day: %s (%d kcal)\n", r.Extremes.HighestExerciseDay.Date, r.Extremes.HighestExerciseDay.ExerciseCalories)
	}

	fmt.Fprintln(out, "\nStreaks")
	fmt.Fprintf(out, "Logging streak: current=%d longest=%d\n", r.Streaks.Logging.Current, r.Streaks.Logging.Longest)
	fmt.Fprintf(out, "Exercise streak: current=%d longest=%d\n", r.Streaks.Exercise.Current, r.Streaks.Exercise.Longest)
	fmt.Fprintf(out, "Within-goal streak: current=%d longest=%d\n", r.Streaks.WithinGoal.Current, r.Streaks.WithinGoal.Longest)

	fmt.Fprintln(out, "\nRolling Windows")
	if r.RollingWindows.Window7.Latest != nil {
		p := r.RollingWindows.Window7.Latest
		fmt.Fprintf(out, "7-day latest (%s): intake=%.1f exercise=%.1f net=%.1f P=%.1f C=%.1f F=%.1f\n", p.Date, p.AvgIntakeCaloriesPerDay, p.AvgExerciseCaloriesPerDay, p.AvgNetCaloriesPerDay, p.AvgProteinGPerDay, p.AvgCarbsGPerDay, p.AvgFatGPerDay)
	} else {
		fmt.Fprintln(out, "7-day latest: n/a")
	}
	if r.RollingWindows.Window30.Latest != nil {
		p := r.RollingWindows.Window30.Latest
		fmt.Fprintf(out, "30-day latest (%s): intake=%.1f exercise=%.1f net=%.1f P=%.1f C=%.1f F=%.1f\n", p.Date, p.AvgIntakeCaloriesPerDay, p.AvgExerciseCaloriesPerDay, p.AvgNetCaloriesPerDay, p.AvgProteinGPerDay, p.AvgCarbsGPerDay, p.AvgFatGPerDay)
	} else {
		fmt.Fprintln(out, "30-day latest: n/a")
	}

	fmt.Fprintln(out, "\nCategory Trends")
	if len(r.CategoryTrends) == 0 {
		fmt.Fprintln(out, "No category data")
	} else {
		for _, ct := range r.CategoryTrends {
			fmt.Fprintf(out, "%s: current=%d prev=%d delta=%+d (%s) share=%.1f%%->%.1f%% trend=%s\n", ct.Category, ct.CurrentCalories, ct.PreviousCalories, ct.DeltaCalories, formatPctDelta(ct.PctDeltaCalories), ct.PreviousSharePct, ct.CurrentSharePct, ct.Trend.Direction)
		}
	}

	if noCharts {
		return
	}

	fmt.Fprintln(out, "\nCharts")
	printSeriesBars(out, "Intake", r.Series, func(b service.InsightsBucket) int { return b.IntakeCalories })
	printSeriesBars(out, "Exercise", r.Series, func(b service.InsightsBucket) int { return b.ExerciseCalories })
	printSeriesBars(out, "Net", r.Series, func(b service.InsightsBucket) int { return b.NetCalories })

	fmt.Fprintln(out, "Macro Sparklines")
	fmt.Fprintf(out, "P %s\n", sparklineFromSeries(r.Series, func(b service.InsightsBucket) float64 { return b.ProteinG }))
	fmt.Fprintf(out, "C %s\n", sparklineFromSeries(r.Series, func(b service.InsightsBucket) float64 { return b.CarbsG }))
	fmt.Fprintf(out, "F %s\n", sparklineFromSeries(r.Series, func(b service.InsightsBucket) float64 { return b.FatG }))
}

func printSeriesBars(out anyWriter, name string, series []service.InsightsBucket, valueFn func(service.InsightsBucket) int) {
	fmt.Fprintf(out, "%s:\n", name)
	maxAbs := 0
	for i := range series {
		v := valueFn(series[i])
		if abs(v) > maxAbs {
			maxAbs = abs(v)
		}
	}
	if maxAbs == 0 {
		fmt.Fprintln(out, "  (all zero)")
		return
	}
	for i := range series {
		v := valueFn(series[i])
		bar := horizontalBar(v, maxAbs, 24)
		fmt.Fprintf(out, "  %-10s %s %d\n", series[i].Label, bar, v)
	}
}

type anyWriter interface {
	Write(p []byte) (n int, err error)
}

func horizontalBar(value, maxAbs, width int) string {
	if width <= 0 || maxAbs <= 0 {
		return ""
	}
	bars := int(math.Round((float64(abs(value)) / float64(maxAbs)) * float64(width)))
	if bars == 0 && value != 0 {
		bars = 1
	}
	prefix := ""
	if value < 0 {
		prefix = "-"
	}
	return prefix + strings.Repeat("#", bars)
}

func sparklineFromSeries(series []service.InsightsBucket, valueFn func(service.InsightsBucket) float64) string {
	if len(series) == 0 {
		return ""
	}
	chars := []rune("._-~=*#@")
	values := make([]float64, 0, len(series))
	minV := 0.0
	maxV := 0.0
	initialized := false
	for i := range series {
		v := valueFn(series[i])
		values = append(values, v)
		if !initialized {
			minV = v
			maxV = v
			initialized = true
			continue
		}
		if v < minV {
			minV = v
		}
		if v > maxV {
			maxV = v
		}
	}
	if maxV == minV {
		return strings.Repeat(string(chars[0]), len(values))
	}
	var b strings.Builder
	for _, v := range values {
		ratio := (v - minV) / (maxV - minV)
		idx := int(math.Round(ratio * float64(len(chars)-1)))
		if idx < 0 {
			idx = 0
		}
		if idx >= len(chars) {
			idx = len(chars) - 1
		}
		b.WriteRune(chars[idx])
	}
	return b.String()
}

func abs(v int) int {
	if v < 0 {
		return -v
	}
	return v
}

func formatDelta(d service.DeltaStat, unit string) string {
	pct := "n/a"
	if d.PctDelta != nil {
		pct = fmt.Sprintf("%+.1f%%", *d.PctDelta)
	}
	return fmt.Sprintf("%+.1f %s, %s", d.AbsDelta, unit, pct)
}

func formatPctDelta(v *float64) string {
	if v == nil {
		return "n/a"
	}
	return fmt.Sprintf("%+.1f%%", *v)
}

func renderAnalyticsInsightsExport(report *service.InsightsReport, jsonBytes []byte, noCharts bool, format string) ([]byte, error) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "", "text":
		var b bytes.Buffer
		printAnalyticsInsightsTable(&b, report, noCharts)
		return b.Bytes(), nil
	case "markdown", "md":
		return []byte(renderAnalyticsInsightsMarkdown(report)), nil
	case "json":
		return jsonBytes, nil
	default:
		return nil, fmt.Errorf("invalid --out-format value %q (use text|markdown|json)", format)
	}
}

func renderAnalyticsInsightsMarkdown(r *service.InsightsReport) string {
	var b strings.Builder
	fmt.Fprintf(&b, "# Analytics Insights Report\n\n")
	fmt.Fprintf(&b, "- Range: `%s` to `%s`\n", r.FromDate, r.ToDate)
	fmt.Fprintf(&b, "- Previous: `%s` to `%s`\n", r.PreviousFromDate, r.PreviousToDate)
	fmt.Fprintf(&b, "- Granularity: `%s`\n\n", r.Granularity)

	fmt.Fprintf(&b, "## Key Metrics\n")
	fmt.Fprintf(&b, "- Avg intake/day: %.1f kcal (%s)\n", r.Current.AvgIntakeCaloriesPerDay, formatDelta(r.Deltas.AvgIntakeCaloriesPerDay, "kcal"))
	fmt.Fprintf(&b, "- Avg exercise/day: %.1f kcal (%s)\n", r.Current.AvgExerciseCaloriesPerDay, formatDelta(r.Deltas.AvgExerciseCaloriesPerDay, "kcal"))
	fmt.Fprintf(&b, "- Avg net/day: %.1f kcal (%s)\n\n", r.Current.AvgNetCaloriesPerDay, formatDelta(r.Deltas.AvgNetCaloriesPerDay, "kcal"))

	fmt.Fprintf(&b, "## Streaks\n")
	fmt.Fprintf(&b, "- Logging: current=%d, longest=%d\n", r.Streaks.Logging.Current, r.Streaks.Logging.Longest)
	fmt.Fprintf(&b, "- Exercise: current=%d, longest=%d\n", r.Streaks.Exercise.Current, r.Streaks.Exercise.Longest)
	fmt.Fprintf(&b, "- Within-goal: current=%d, longest=%d\n\n", r.Streaks.WithinGoal.Current, r.Streaks.WithinGoal.Longest)

	fmt.Fprintf(&b, "## Rolling Windows\n")
	if r.RollingWindows.Window7.Latest != nil {
		p := r.RollingWindows.Window7.Latest
		fmt.Fprintf(&b, "- 7-day latest `%s`: intake=%.1f, exercise=%.1f, net=%.1f, P=%.1f, C=%.1f, F=%.1f\n", p.Date, p.AvgIntakeCaloriesPerDay, p.AvgExerciseCaloriesPerDay, p.AvgNetCaloriesPerDay, p.AvgProteinGPerDay, p.AvgCarbsGPerDay, p.AvgFatGPerDay)
	} else {
		fmt.Fprintf(&b, "- 7-day latest: n/a\n")
	}
	if r.RollingWindows.Window30.Latest != nil {
		p := r.RollingWindows.Window30.Latest
		fmt.Fprintf(&b, "- 30-day latest `%s`: intake=%.1f, exercise=%.1f, net=%.1f, P=%.1f, C=%.1f, F=%.1f\n\n", p.Date, p.AvgIntakeCaloriesPerDay, p.AvgExerciseCaloriesPerDay, p.AvgNetCaloriesPerDay, p.AvgProteinGPerDay, p.AvgCarbsGPerDay, p.AvgFatGPerDay)
	} else {
		fmt.Fprintf(&b, "- 30-day latest: n/a\n\n")
	}

	fmt.Fprintf(&b, "## Category Trends\n")
	if len(r.CategoryTrends) == 0 {
		fmt.Fprintf(&b, "- No category data\n")
		return b.String()
	}
	for _, ct := range r.CategoryTrends {
		fmt.Fprintf(&b, "- **%s**: current=%d, prev=%d, delta=%+d (%s), share=%.1f%%->%.1f%%, trend=%s\n", ct.Category, ct.CurrentCalories, ct.PreviousCalories, ct.DeltaCalories, formatPctDelta(ct.PctDeltaCalories), ct.PreviousSharePct, ct.CurrentSharePct, ct.Trend.Direction)
	}
	return b.String()
}

func parseInsightsGranularity(value string) (service.InsightsGranularity, error) {
	s := strings.ToLower(strings.TrimSpace(value))
	switch s {
	case "", "auto":
		return service.InsightsGranularityAuto, nil
	case "day":
		return service.InsightsGranularityDay, nil
	case "week":
		return service.InsightsGranularityWeek, nil
	case "month":
		return service.InsightsGranularityMonth, nil
	default:
		return "", fmt.Errorf("invalid --granularity value %q (use auto|day|week|month)", value)
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
	analyticsCmd.AddCommand(analyticsWeekCmd, analyticsMonthCmd, analyticsRangeCmd, analyticsInsightsCmd)
	analyticsInsightsCmd.AddCommand(analyticsInsightsWeekCmd, analyticsInsightsMonthCmd, analyticsInsightsRangeCmd)

	for _, c := range []*cobra.Command{analyticsWeekCmd, analyticsMonthCmd, analyticsRangeCmd} {
		c.Flags().BoolVar(&analyticsJSON, "json", false, "Output as JSON")
		c.Flags().Float64Var(&analyticsTolerance, "tolerance", 0.10, "Macro adherence tolerance (0.10 = 10%)")
	}
	analyticsWeekCmd.Flags().StringVar(&weekArg, "week", "", "ISO week in format YYYY-Www")
	analyticsMonthCmd.Flags().StringVar(&monthArg, "month", "", "Month in format YYYY-MM")
	analyticsRangeCmd.Flags().StringVar(&rangeFrom, "from", "", "Start date YYYY-MM-DD")
	analyticsRangeCmd.Flags().StringVar(&rangeTo, "to", "", "End date YYYY-MM-DD")

	for _, c := range []*cobra.Command{analyticsInsightsWeekCmd, analyticsInsightsMonthCmd, analyticsInsightsRangeCmd} {
		c.Flags().BoolVar(&insightsJSON, "json", false, "Output as JSON")
		c.Flags().Float64Var(&insightsTolerance, "tolerance", 0.10, "Macro adherence tolerance (0.10 = 10%)")
		c.Flags().StringVar(&insightsGranularity, "granularity", "auto", "Bucket granularity: auto|day|week|month")
		c.Flags().BoolVar(&insightsNoCharts, "no-charts", false, "Disable ASCII charts in text output")
		c.Flags().StringVar(&insightsOutPath, "out", "", "Write a report to a file path")
		c.Flags().StringVar(&insightsOutFormat, "out-format", "text", "Report file format: text|markdown|json")
	}
	analyticsInsightsWeekCmd.Flags().StringVar(&insightsWeekArg, "week", "", "ISO week in format YYYY-Www")
	analyticsInsightsMonthCmd.Flags().StringVar(&insightsMonthArg, "month", "", "Month in format YYYY-MM")
	analyticsInsightsRangeCmd.Flags().StringVar(&insightsRangeFrom, "from", "", "Start date YYYY-MM-DD")
	analyticsInsightsRangeCmd.Flags().StringVar(&insightsRangeTo, "to", "", "End date YYYY-MM-DD")
}
