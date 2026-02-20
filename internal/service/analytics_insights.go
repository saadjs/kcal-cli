package service

import (
	"database/sql"
	"fmt"
	"math"
	"sort"
	"time"
)

type InsightsGranularity string

const (
	InsightsGranularityAuto  InsightsGranularity = "auto"
	InsightsGranularityDay   InsightsGranularity = "day"
	InsightsGranularityWeek  InsightsGranularity = "week"
	InsightsGranularityMonth InsightsGranularity = "month"
)

type InsightsBucket struct {
	Label            string  `json:"label"`
	FromDate         string  `json:"from_date"`
	ToDate           string  `json:"to_date"`
	IntakeCalories   int     `json:"intake_calories"`
	ExerciseCalories int     `json:"exercise_calories"`
	NetCalories      int     `json:"net_calories"`
	ProteinG         float64 `json:"protein_g"`
	CarbsG           float64 `json:"carbs_g"`
	FatG             float64 `json:"fat_g"`
	AdherenceWithin  *bool   `json:"adherence_within,omitempty"`
}

type DeltaStat struct {
	Current  float64  `json:"current"`
	Previous float64  `json:"previous"`
	AbsDelta float64  `json:"abs_delta"`
	PctDelta *float64 `json:"pct_delta,omitempty"`
}

type ConsistencyStat struct {
	Mean     float64 `json:"mean"`
	StdDev   float64 `json:"std_dev"`
	CoeffVar float64 `json:"coeff_var"`
}

type InsightsCurrentSummary struct {
	TotalDays          int     `json:"total_days"`
	IntakeActiveDays   int     `json:"intake_active_days"`
	ExerciseActiveDays int     `json:"exercise_active_days"`
	IntakeActiveRate   float64 `json:"intake_active_rate"`
	ExerciseActiveRate float64 `json:"exercise_active_rate"`

	TotalIntakeCalories   int `json:"total_intake_calories"`
	TotalExerciseCalories int `json:"total_exercise_calories"`
	TotalNetCalories      int `json:"total_net_calories"`

	TotalProteinG float64 `json:"total_protein_g"`
	TotalCarbsG   float64 `json:"total_carbs_g"`
	TotalFatG     float64 `json:"total_fat_g"`

	AvgIntakeCaloriesPerDay   float64 `json:"avg_intake_calories_per_day"`
	AvgExerciseCaloriesPerDay float64 `json:"avg_exercise_calories_per_day"`
	AvgNetCaloriesPerDay      float64 `json:"avg_net_calories_per_day"`
	AvgProteinPerDay          float64 `json:"avg_protein_g_per_day"`
	AvgCarbsPerDay            float64 `json:"avg_carbs_g_per_day"`
	AvgFatPerDay              float64 `json:"avg_fat_g_per_day"`

	Adherence AdherenceSummary `json:"adherence"`
}

type InsightsDeltaSummary struct {
	AvgIntakeCaloriesPerDay   DeltaStat `json:"avg_intake_calories_per_day"`
	AvgExerciseCaloriesPerDay DeltaStat `json:"avg_exercise_calories_per_day"`
	AvgNetCaloriesPerDay      DeltaStat `json:"avg_net_calories_per_day"`
	AvgProteinPerDay          DeltaStat `json:"avg_protein_g_per_day"`
	AvgCarbsPerDay            DeltaStat `json:"avg_carbs_g_per_day"`
	AvgFatPerDay              DeltaStat `json:"avg_fat_g_per_day"`
	AdherencePercent          DeltaStat `json:"adherence_percent"`
}

type InsightsConsistencySummary struct {
	IntakeCalories ConsistencyStat `json:"intake_calories"`
	NetCalories    ConsistencyStat `json:"net_calories"`
	ProteinG       ConsistencyStat `json:"protein_g"`
	CarbsG         ConsistencyStat `json:"carbs_g"`
	FatG           ConsistencyStat `json:"fat_g"`
}

type TrendStat struct {
	SlopePerBucket float64 `json:"slope_per_bucket"`
	Direction      string  `json:"direction"`
}

type InsightsTrendSummary struct {
	IntakeCalories TrendStat `json:"intake_calories"`
	Exercise       TrendStat `json:"exercise_calories"`
	NetCalories    TrendStat `json:"net_calories"`
}

type InsightsExtremes struct {
	HighestNetDay      *DaySummary `json:"highest_net_day,omitempty"`
	LowestNetDay       *DaySummary `json:"lowest_net_day,omitempty"`
	HighestExerciseDay *DaySummary `json:"highest_exercise_day,omitempty"`
}

type MacroShare struct {
	ProteinPct float64 `json:"protein_pct"`
	CarbsPct   float64 `json:"carbs_pct"`
	FatPct     float64 `json:"fat_pct"`
}

type MacroGoalShareDelta struct {
	ProteinPctPointDelta float64 `json:"protein_pct_point_delta"`
	CarbsPctPointDelta   float64 `json:"carbs_pct_point_delta"`
	FatPctPointDelta     float64 `json:"fat_pct_point_delta"`
}

type InsightsMacroBalance struct {
	CurrentShare MacroShare           `json:"current_share"`
	GoalShare    *MacroShare          `json:"goal_share,omitempty"`
	GoalDelta    *MacroGoalShareDelta `json:"goal_delta,omitempty"`
}

type Streak struct {
	Current int `json:"current"`
	Longest int `json:"longest"`
}

type InsightsStreaks struct {
	Logging    Streak `json:"logging"`
	Exercise   Streak `json:"exercise"`
	WithinGoal Streak `json:"within_goal"`
}

type RollingWindowPoint struct {
	Date                      string  `json:"date"`
	AvgIntakeCaloriesPerDay   float64 `json:"avg_intake_calories_per_day"`
	AvgExerciseCaloriesPerDay float64 `json:"avg_exercise_calories_per_day"`
	AvgNetCaloriesPerDay      float64 `json:"avg_net_calories_per_day"`
	AvgProteinGPerDay         float64 `json:"avg_protein_g_per_day"`
	AvgCarbsGPerDay           float64 `json:"avg_carbs_g_per_day"`
	AvgFatGPerDay             float64 `json:"avg_fat_g_per_day"`
}

type RollingWindowSummary struct {
	WindowDays int                  `json:"window_days"`
	Latest     *RollingWindowPoint  `json:"latest,omitempty"`
	Points     []RollingWindowPoint `json:"points"`
}

type InsightsRollingWindows struct {
	Window7  RollingWindowSummary `json:"window_7"`
	Window30 RollingWindowSummary `json:"window_30"`
}

type CategoryTrend struct {
	Category            string    `json:"category"`
	CurrentCalories     int       `json:"current_calories"`
	PreviousCalories    int       `json:"previous_calories"`
	DeltaCalories       int       `json:"delta_calories"`
	PctDeltaCalories    *float64  `json:"pct_delta_calories,omitempty"`
	CurrentSharePct     float64   `json:"current_share_pct"`
	PreviousSharePct    float64   `json:"previous_share_pct"`
	ShareDeltaPctPoints float64   `json:"share_delta_pct_points"`
	Trend               TrendStat `json:"trend"`
}

type InsightsReport struct {
	FromDate         string              `json:"from_date"`
	ToDate           string              `json:"to_date"`
	PreviousFromDate string              `json:"previous_from_date"`
	PreviousToDate   string              `json:"previous_to_date"`
	Granularity      InsightsGranularity `json:"granularity"`

	Current        InsightsCurrentSummary     `json:"current"`
	Deltas         InsightsDeltaSummary       `json:"deltas"`
	Consistency    InsightsConsistencySummary `json:"consistency"`
	Trends         InsightsTrendSummary       `json:"trends"`
	Extremes       InsightsExtremes           `json:"extremes"`
	MacroBalance   InsightsMacroBalance       `json:"macro_balance"`
	Streaks        InsightsStreaks            `json:"streaks"`
	RollingWindows InsightsRollingWindows     `json:"rolling_windows"`
	CategoryTrends []CategoryTrend            `json:"category_trends"`
	Series         []InsightsBucket           `json:"series"`
}

func AnalyticsInsightsRange(db *sql.DB, from, to time.Time, tolerance float64, granularity InsightsGranularity) (*InsightsReport, error) {
	if from.After(to) {
		return nil, fmt.Errorf("from date must be <= to date")
	}
	from = beginningOfDay(from)
	to = beginningOfDay(to)

	daysCount := inclusiveDayCount(from, to)
	prevTo := from.AddDate(0, 0, -1)
	prevFrom := prevTo.AddDate(0, 0, -(daysCount - 1))

	resolvedGranularity, err := resolveInsightsGranularity(granularity, daysCount)
	if err != nil {
		return nil, err
	}

	currentDays, currentAdherence, err := loadFullDaySeries(db, from, to, tolerance)
	if err != nil {
		return nil, err
	}
	prevDays, previousAdherence, err := loadFullDaySeries(db, prevFrom, prevTo, tolerance)
	if err != nil {
		return nil, err
	}

	current := summarizeDaySeries(currentDays, currentAdherence)
	previous := summarizeDaySeries(prevDays, previousAdherence)

	report := &InsightsReport{
		FromDate:         from.Format("2006-01-02"),
		ToDate:           to.Format("2006-01-02"),
		PreviousFromDate: prevFrom.Format("2006-01-02"),
		PreviousToDate:   prevTo.Format("2006-01-02"),
		Granularity:      resolvedGranularity,
		Current:          current,
		Deltas: InsightsDeltaSummary{
			AvgIntakeCaloriesPerDay:   computeDelta(current.AvgIntakeCaloriesPerDay, previous.AvgIntakeCaloriesPerDay),
			AvgExerciseCaloriesPerDay: computeDelta(current.AvgExerciseCaloriesPerDay, previous.AvgExerciseCaloriesPerDay),
			AvgNetCaloriesPerDay:      computeDelta(current.AvgNetCaloriesPerDay, previous.AvgNetCaloriesPerDay),
			AvgProteinPerDay:          computeDelta(current.AvgProteinPerDay, previous.AvgProteinPerDay),
			AvgCarbsPerDay:            computeDelta(current.AvgCarbsPerDay, previous.AvgCarbsPerDay),
			AvgFatPerDay:              computeDelta(current.AvgFatPerDay, previous.AvgFatPerDay),
			AdherencePercent:          computeDelta(current.Adherence.PercentWithin, previous.Adherence.PercentWithin),
		},
		Consistency: computeConsistency(currentDays),
		Extremes:    computeExtremes(currentDays),
		Series:      bucketDaySeries(currentDays, resolvedGranularity),
		Streaks:     computeStreaks(currentDays),
		RollingWindows: InsightsRollingWindows{
			Window7:  computeRollingWindow(currentDays, 7),
			Window30: computeRollingWindow(currentDays, 30),
		},
	}
	report.Trends = computeTrends(report.Series)
	report.MacroBalance, err = computeMacroBalance(db, from, to, current)
	if err != nil {
		return nil, err
	}
	report.CategoryTrends, err = computeCategoryTrends(db, from, to, prevFrom, prevTo, resolvedGranularity)
	if err != nil {
		return nil, err
	}
	return report, nil
}

func resolveInsightsGranularity(in InsightsGranularity, daysCount int) (InsightsGranularity, error) {
	switch in {
	case "", InsightsGranularityAuto:
		if daysCount <= 31 {
			return InsightsGranularityDay, nil
		}
		if daysCount <= 120 {
			return InsightsGranularityWeek, nil
		}
		return InsightsGranularityMonth, nil
	case InsightsGranularityDay, InsightsGranularityWeek, InsightsGranularityMonth:
		return in, nil
	default:
		return "", fmt.Errorf("invalid granularity %q (use auto, day, week, month)", in)
	}
}

func loadFullDaySeries(db *sql.DB, from, to time.Time, tolerance float64) ([]DaySummary, AdherenceSummary, error) {
	intakeByDay, err := loadIntakeDaySummaries(db, from, to)
	if err != nil {
		return nil, AdherenceSummary{}, err
	}
	exerciseByDay, err := loadExerciseCaloriesByDay(db, from, to)
	if err != nil {
		return nil, AdherenceSummary{}, err
	}

	days := make([]DaySummary, 0, inclusiveDayCount(from, to))
	for d := from; !d.After(to); d = d.AddDate(0, 0, 1) {
		key := d.Format("2006-01-02")
		day := DaySummary{Date: key}
		if intake, ok := intakeByDay[key]; ok {
			day = intake
		}
		day.IntakeCalories = day.Calories
		day.ExerciseCalories = exerciseByDay[key]
		day.NetCalories = day.IntakeCalories - day.ExerciseCalories
		days = append(days, day)
	}

	adherence, err := calculateAdherence(db, days, tolerance)
	if err != nil {
		return nil, AdherenceSummary{}, err
	}
	return days, adherence, nil
}

func inclusiveDayCount(from, to time.Time) int {
	count := 0
	for d := beginningOfDay(from); !d.After(to); d = d.AddDate(0, 0, 1) {
		count++
	}
	return count
}

func summarizeDaySeries(days []DaySummary, adherence AdherenceSummary) InsightsCurrentSummary {
	out := InsightsCurrentSummary{
		TotalDays: len(days),
		Adherence: adherence,
	}
	if len(days) == 0 {
		return out
	}

	for i := range days {
		out.TotalIntakeCalories += days[i].IntakeCalories
		out.TotalExerciseCalories += days[i].ExerciseCalories
		out.TotalNetCalories += days[i].NetCalories
		out.TotalProteinG += days[i].Protein
		out.TotalCarbsG += days[i].Carbs
		out.TotalFatG += days[i].Fat
		if days[i].IntakeCalories > 0 {
			out.IntakeActiveDays++
		}
		if days[i].ExerciseCalories > 0 {
			out.ExerciseActiveDays++
		}
	}

	div := float64(len(days))
	out.IntakeActiveRate = float64(out.IntakeActiveDays) / div
	out.ExerciseActiveRate = float64(out.ExerciseActiveDays) / div
	out.AvgIntakeCaloriesPerDay = float64(out.TotalIntakeCalories) / div
	out.AvgExerciseCaloriesPerDay = float64(out.TotalExerciseCalories) / div
	out.AvgNetCaloriesPerDay = float64(out.TotalNetCalories) / div
	out.AvgProteinPerDay = out.TotalProteinG / div
	out.AvgCarbsPerDay = out.TotalCarbsG / div
	out.AvgFatPerDay = out.TotalFatG / div
	return out
}

func computeDelta(current, previous float64) DeltaStat {
	out := DeltaStat{
		Current:  current,
		Previous: previous,
		AbsDelta: current - previous,
	}
	if previous == 0 {
		return out
	}
	v := ((current - previous) / previous) * 100
	out.PctDelta = &v
	return out
}

func computeConsistency(days []DaySummary) InsightsConsistencySummary {
	intake := make([]float64, 0, len(days))
	net := make([]float64, 0, len(days))
	protein := make([]float64, 0, len(days))
	carbs := make([]float64, 0, len(days))
	fat := make([]float64, 0, len(days))
	for i := range days {
		intake = append(intake, float64(days[i].IntakeCalories))
		net = append(net, float64(days[i].NetCalories))
		protein = append(protein, days[i].Protein)
		carbs = append(carbs, days[i].Carbs)
		fat = append(fat, days[i].Fat)
	}
	return InsightsConsistencySummary{
		IntakeCalories: calcConsistencyStat(intake),
		NetCalories:    calcConsistencyStat(net),
		ProteinG:       calcConsistencyStat(protein),
		CarbsG:         calcConsistencyStat(carbs),
		FatG:           calcConsistencyStat(fat),
	}
}

func calcConsistencyStat(values []float64) ConsistencyStat {
	if len(values) == 0 {
		return ConsistencyStat{}
	}
	mean := avg(values)
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	stddev := math.Sqrt(sum / float64(len(values)))
	out := ConsistencyStat{
		Mean:   mean,
		StdDev: stddev,
	}
	if mean != 0 {
		out.CoeffVar = stddev / math.Abs(mean)
	}
	return out
}

func avg(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func computeExtremes(days []DaySummary) InsightsExtremes {
	if len(days) == 0 {
		return InsightsExtremes{}
	}
	copied := make([]DaySummary, len(days))
	copy(copied, days)
	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].NetCalories < copied[j].NetCalories
	})
	low := copied[0]
	high := copied[len(copied)-1]

	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].ExerciseCalories < copied[j].ExerciseCalories
	})
	highEx := copied[len(copied)-1]

	return InsightsExtremes{
		HighestNetDay:      &high,
		LowestNetDay:       &low,
		HighestExerciseDay: &highEx,
	}
}

func computeStreaks(days []DaySummary) InsightsStreaks {
	logCurrent, logLongest := computeBooleanStreak(days, func(d DaySummary) bool {
		return d.IntakeCalories > 0
	})
	exCurrent, exLongest := computeBooleanStreak(days, func(d DaySummary) bool {
		return d.ExerciseCalories > 0
	})
	goalCurrent, goalLongest := computeBooleanStreak(days, func(d DaySummary) bool {
		if d.EffectiveGoalCalories == 0 && d.EffectiveGoalProtein == 0 && d.EffectiveGoalCarbs == 0 && d.EffectiveGoalFat == 0 {
			return false
		}
		return float64(d.NetCalories) <= float64(d.EffectiveGoalCalories) &&
			AdherenceWithin(d.Protein, d.EffectiveGoalProtein, 0.10) &&
			AdherenceWithin(d.Carbs, d.EffectiveGoalCarbs, 0.10) &&
			AdherenceWithin(d.Fat, d.EffectiveGoalFat, 0.10)
	})
	return InsightsStreaks{
		Logging: Streak{
			Current: logCurrent,
			Longest: logLongest,
		},
		Exercise: Streak{
			Current: exCurrent,
			Longest: exLongest,
		},
		WithinGoal: Streak{
			Current: goalCurrent,
			Longest: goalLongest,
		},
	}
}

func computeBooleanStreak(days []DaySummary, predicate func(DaySummary) bool) (current, longest int) {
	run := 0
	for i := range days {
		if predicate(days[i]) {
			run++
			if run > longest {
				longest = run
			}
			continue
		}
		run = 0
	}
	for i := len(days) - 1; i >= 0; i-- {
		if predicate(days[i]) {
			current++
			continue
		}
		break
	}
	return current, longest
}

func computeRollingWindow(days []DaySummary, windowDays int) RollingWindowSummary {
	out := RollingWindowSummary{
		WindowDays: windowDays,
		Points:     make([]RollingWindowPoint, 0),
	}
	if windowDays <= 0 || len(days) < windowDays {
		return out
	}
	for i := windowDays - 1; i < len(days); i++ {
		start := i - (windowDays - 1)
		slice := days[start : i+1]
		point := RollingWindowPoint{
			Date:                      days[i].Date,
			AvgIntakeCaloriesPerDay:   avgDayInt(slice, func(d DaySummary) int { return d.IntakeCalories }),
			AvgExerciseCaloriesPerDay: avgDayInt(slice, func(d DaySummary) int { return d.ExerciseCalories }),
			AvgNetCaloriesPerDay:      avgDayInt(slice, func(d DaySummary) int { return d.NetCalories }),
			AvgProteinGPerDay:         avgDayFloat(slice, func(d DaySummary) float64 { return d.Protein }),
			AvgCarbsGPerDay:           avgDayFloat(slice, func(d DaySummary) float64 { return d.Carbs }),
			AvgFatGPerDay:             avgDayFloat(slice, func(d DaySummary) float64 { return d.Fat }),
		}
		out.Points = append(out.Points, point)
	}
	last := out.Points[len(out.Points)-1]
	out.Latest = &last
	return out
}

func avgDayInt(days []DaySummary, selector func(DaySummary) int) float64 {
	if len(days) == 0 {
		return 0
	}
	sum := 0
	for i := range days {
		sum += selector(days[i])
	}
	return float64(sum) / float64(len(days))
}

func avgDayFloat(days []DaySummary, selector func(DaySummary) float64) float64 {
	if len(days) == 0 {
		return 0
	}
	sum := 0.0
	for i := range days {
		sum += selector(days[i])
	}
	return sum / float64(len(days))
}

type dayBucket struct {
	from time.Time
	to   time.Time
	key  string
}

func bucketDaySeries(days []DaySummary, granularity InsightsGranularity) []InsightsBucket {
	if len(days) == 0 {
		return nil
	}
	acc := map[string]*InsightsBucket{}
	order := make([]string, 0, len(days))

	for i := range days {
		date, _ := time.ParseInLocation("2006-01-02", days[i].Date, time.Local)
		b := makeBucket(date, granularity)
		item, ok := acc[b.key]
		if !ok {
			item = &InsightsBucket{
				Label:    b.key,
				FromDate: b.from.Format("2006-01-02"),
				ToDate:   b.to.Format("2006-01-02"),
			}
			order = append(order, b.key)
			acc[b.key] = item
		}
		item.IntakeCalories += days[i].IntakeCalories
		item.ExerciseCalories += days[i].ExerciseCalories
		item.NetCalories += days[i].NetCalories
		item.ProteinG += days[i].Protein
		item.CarbsG += days[i].Carbs
		item.FatG += days[i].Fat
	}

	out := make([]InsightsBucket, 0, len(order))
	for _, k := range order {
		out = append(out, *acc[k])
	}
	return out
}

func makeBucket(date time.Time, granularity InsightsGranularity) dayBucket {
	switch granularity {
	case InsightsGranularityMonth:
		from := time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, time.Local)
		to := from.AddDate(0, 1, -1)
		return dayBucket{
			from: from,
			to:   to,
			key:  from.Format("2006-01"),
		}
	case InsightsGranularityWeek:
		year, week := date.ISOWeek()
		start := beginningOfWeekLocal(date)
		return dayBucket{
			from: start,
			to:   start.AddDate(0, 0, 6),
			key:  fmt.Sprintf("%04d-W%02d", year, week),
		}
	default:
		return dayBucket{
			from: date,
			to:   date,
			key:  date.Format("2006-01-02"),
		}
	}
}

func beginningOfWeekLocal(t time.Time) time.Time {
	weekday := int(t.Weekday())
	if weekday == 0 {
		weekday = 7
	}
	start := t.AddDate(0, 0, -(weekday - 1))
	return time.Date(start.Year(), start.Month(), start.Day(), 0, 0, 0, 0, time.Local)
}

func computeTrends(series []InsightsBucket) InsightsTrendSummary {
	intake := make([]float64, 0, len(series))
	exercise := make([]float64, 0, len(series))
	net := make([]float64, 0, len(series))
	for i := range series {
		intake = append(intake, float64(series[i].IntakeCalories))
		exercise = append(exercise, float64(series[i].ExerciseCalories))
		net = append(net, float64(series[i].NetCalories))
	}
	return InsightsTrendSummary{
		IntakeCalories: trendFromValues(intake),
		Exercise:       trendFromValues(exercise),
		NetCalories:    trendFromValues(net),
	}
}

func trendFromValues(values []float64) TrendStat {
	slope := linearRegressionSlope(values)
	direction := "flat"
	if slope >= 0.5 {
		direction = "up"
	} else if slope <= -0.5 {
		direction = "down"
	}
	return TrendStat{
		SlopePerBucket: slope,
		Direction:      direction,
	}
}

func linearRegressionSlope(values []float64) float64 {
	n := len(values)
	if n < 2 {
		return 0
	}
	var sumX, sumY, sumXY, sumX2 float64
	for i := range values {
		x := float64(i)
		y := values[i]
		sumX += x
		sumY += y
		sumXY += x * y
		sumX2 += x * x
	}
	denom := (float64(n) * sumX2) - (sumX * sumX)
	if denom == 0 {
		return 0
	}
	return ((float64(n) * sumXY) - (sumX * sumY)) / denom
}

func computeMacroBalance(db *sql.DB, from, to time.Time, current InsightsCurrentSummary) (InsightsMacroBalance, error) {
	proteinKcal := current.TotalProteinG * 4
	carbsKcal := current.TotalCarbsG * 4
	fatKcal := current.TotalFatG * 9
	total := proteinKcal + carbsKcal + fatKcal

	out := InsightsMacroBalance{}
	if total > 0 {
		out.CurrentShare = MacroShare{
			ProteinPct: (proteinKcal / total) * 100,
			CarbsPct:   (carbsKcal / total) * 100,
			FatPct:     (fatKcal / total) * 100,
		}
	}

	midDate := from.Add(to.Sub(from) / 2).Format("2006-01-02")
	goal, err := CurrentGoal(db, midDate)
	if err != nil {
		return out, err
	}
	if goal == nil {
		return out, nil
	}

	goalProteinKcal := goal.ProteinG * 4
	goalCarbsKcal := goal.CarbsG * 4
	goalFatKcal := goal.FatG * 9
	goalTotal := goalProteinKcal + goalCarbsKcal + goalFatKcal
	if goalTotal <= 0 {
		return out, nil
	}

	goalShare := &MacroShare{
		ProteinPct: (goalProteinKcal / goalTotal) * 100,
		CarbsPct:   (goalCarbsKcal / goalTotal) * 100,
		FatPct:     (goalFatKcal / goalTotal) * 100,
	}
	delta := &MacroGoalShareDelta{
		ProteinPctPointDelta: out.CurrentShare.ProteinPct - goalShare.ProteinPct,
		CarbsPctPointDelta:   out.CurrentShare.CarbsPct - goalShare.CarbsPct,
		FatPctPointDelta:     out.CurrentShare.FatPct - goalShare.FatPct,
	}
	out.GoalShare = goalShare
	out.GoalDelta = delta
	return out, nil
}

func computeCategoryTrends(db *sql.DB, from, to, prevFrom, prevTo time.Time, granularity InsightsGranularity) ([]CategoryTrend, error) {
	currentTotals, currentTotalCalories, err := loadCategoryTotals(db, from, to)
	if err != nil {
		return nil, err
	}
	previousTotals, previousTotalCalories, err := loadCategoryTotals(db, prevFrom, prevTo)
	if err != nil {
		return nil, err
	}

	currentBuckets, bucketLabels, err := loadCategoryBucketCalories(db, from, to, granularity)
	if err != nil {
		return nil, err
	}

	categories := make(map[string]struct{})
	for k := range currentTotals {
		categories[k] = struct{}{}
	}
	for k := range previousTotals {
		categories[k] = struct{}{}
	}

	out := make([]CategoryTrend, 0, len(categories))
	for category := range categories {
		cur := currentTotals[category]
		prev := previousTotals[category]
		var pctDelta *float64
		if prev != 0 {
			v := (float64(cur-prev) / float64(prev)) * 100
			pctDelta = &v
		}

		series := make([]float64, 0, len(bucketLabels))
		byBucket := currentBuckets[category]
		for _, label := range bucketLabels {
			series = append(series, float64(byBucket[label]))
		}

		trend := CategoryTrend{
			Category:            category,
			CurrentCalories:     cur,
			PreviousCalories:    prev,
			DeltaCalories:       cur - prev,
			PctDeltaCalories:    pctDelta,
			CurrentSharePct:     pctShare(cur, currentTotalCalories),
			PreviousSharePct:    pctShare(prev, previousTotalCalories),
			ShareDeltaPctPoints: pctShare(cur, currentTotalCalories) - pctShare(prev, previousTotalCalories),
			Trend:               trendFromValues(series),
		}
		out = append(out, trend)
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].CurrentCalories == out[j].CurrentCalories {
			return out[i].Category < out[j].Category
		}
		return out[i].CurrentCalories > out[j].CurrentCalories
	})
	return out, nil
}

func pctShare(value, total int) float64 {
	if total <= 0 {
		return 0
	}
	return (float64(value) / float64(total)) * 100
}

func loadCategoryTotals(db *sql.DB, from, to time.Time) (map[string]int, int, error) {
	rows, err := db.Query(`
SELECT c.name, SUM(e.calories)
FROM entries e
JOIN categories c ON c.id = e.category_id
WHERE e.consumed_at >= ? AND e.consumed_at < ?
GROUP BY c.name
`, from.Format(time.RFC3339), to.Add(24*time.Hour).Format(time.RFC3339))
	if err != nil {
		return nil, 0, fmt.Errorf("query category totals: %w", err)
	}
	defer rows.Close()

	out := map[string]int{}
	total := 0
	for rows.Next() {
		var category string
		var calories int
		if err := rows.Scan(&category, &calories); err != nil {
			return nil, 0, fmt.Errorf("scan category totals: %w", err)
		}
		out[category] = calories
		total += calories
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("iterate category totals: %w", err)
	}
	return out, total, nil
}

func loadCategoryBucketCalories(db *sql.DB, from, to time.Time, granularity InsightsGranularity) (map[string]map[string]int, []string, error) {
	rows, err := db.Query(`
SELECT substr(e.consumed_at, 1, 10) as day, c.name, SUM(e.calories)
FROM entries e
JOIN categories c ON c.id = e.category_id
WHERE e.consumed_at >= ? AND e.consumed_at < ?
GROUP BY day, c.name
ORDER BY day ASC
`, from.Format(time.RFC3339), to.Add(24*time.Hour).Format(time.RFC3339))
	if err != nil {
		return nil, nil, fmt.Errorf("query category bucket calories: %w", err)
	}
	defer rows.Close()

	out := map[string]map[string]int{}
	labelSeen := map[string]struct{}{}
	labels := make([]string, 0)
	for rows.Next() {
		var day string
		var category string
		var calories int
		if err := rows.Scan(&day, &category, &calories); err != nil {
			return nil, nil, fmt.Errorf("scan category bucket calories: %w", err)
		}
		date, err := time.ParseInLocation("2006-01-02", day, time.Local)
		if err != nil {
			return nil, nil, fmt.Errorf("parse category bucket day: %w", err)
		}
		label := makeBucket(date, granularity).key
		if _, ok := out[category]; !ok {
			out[category] = map[string]int{}
		}
		out[category][label] += calories
		if _, ok := labelSeen[label]; !ok {
			labelSeen[label] = struct{}{}
			labels = append(labels, label)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate category bucket calories: %w", err)
	}
	sort.Strings(labels)
	return out, labels, nil
}
