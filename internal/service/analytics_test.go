package service_test

import (
	"testing"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
)

func TestAnalyticsRangeTotalsAndAdherence(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if err := service.SetGoal(db, service.SetGoalInput{
		Calories:      2000,
		ProteinG:      150,
		CarbsG:        200,
		FatG:          70,
		EffectiveDate: "2026-02-01",
	}); err != nil {
		t.Fatalf("set goal: %v", err)
	}

	seed := []service.CreateEntryInput{
		{
			Name:       "Breakfast",
			Calories:   500,
			ProteinG:   40,
			CarbsG:     50,
			FatG:       15,
			Category:   "breakfast",
			Consumed:   time.Date(2026, 2, 10, 8, 0, 0, 0, time.Local),
			SourceType: "manual",
		},
		{
			Name:       "Dinner",
			Calories:   700,
			ProteinG:   60,
			CarbsG:     70,
			FatG:       20,
			Category:   "dinner",
			Consumed:   time.Date(2026, 2, 10, 19, 0, 0, 0, time.Local),
			SourceType: "manual",
		},
		{
			Name:       "Lunch",
			Calories:   900,
			ProteinG:   80,
			CarbsG:     90,
			FatG:       25,
			Category:   "lunch",
			Consumed:   time.Date(2026, 2, 11, 13, 0, 0, 0, time.Local),
			SourceType: "manual",
		},
	}
	for _, e := range seed {
		if _, err := service.CreateEntry(db, e); err != nil {
			t.Fatalf("create seed entry %s: %v", e.Name, err)
		}
	}
	if _, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "running",
		CaloriesBurned: 300,
		PerformedAt:    time.Date(2026, 2, 10, 20, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("add exercise log: %v", err)
	}
	if err := service.SetBodyGoal(db, service.SetBodyGoalInput{
		TargetWeight:  78,
		Unit:          "kg",
		TargetBodyFat: floatPtr(19),
		EffectiveDate: "2026-02-01",
	}); err != nil {
		t.Fatalf("set body goal: %v", err)
	}
	if _, err := service.AddBodyMeasurement(db, service.BodyMeasurementInput{
		Weight:     80,
		Unit:       "kg",
		BodyFatPct: floatPtr(21),
		MeasuredAt: time.Date(2026, 2, 10, 7, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("add body measurement 1: %v", err)
	}
	if _, err := service.AddBodyMeasurement(db, service.BodyMeasurementInput{
		Weight:     79.5,
		Unit:       "kg",
		BodyFatPct: floatPtr(20.5),
		MeasuredAt: time.Date(2026, 2, 11, 7, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("add body measurement 2: %v", err)
	}

	report, err := service.AnalyticsRange(
		db,
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.Local),
		time.Date(2026, 2, 11, 0, 0, 0, 0, time.Local),
		0.10,
	)
	if err != nil {
		t.Fatalf("analytics range: %v", err)
	}

	if report.TotalCalories != 2100 {
		t.Fatalf("expected total calories 2100, got %d", report.TotalCalories)
	}
	if report.TotalIntakeCalories != 2100 {
		t.Fatalf("expected total intake calories 2100, got %d", report.TotalIntakeCalories)
	}
	if report.TotalExerciseCalories != 300 {
		t.Fatalf("expected total exercise calories 300, got %d", report.TotalExerciseCalories)
	}
	if report.TotalNetCalories != 1800 {
		t.Fatalf("expected total net calories 1800, got %d", report.TotalNetCalories)
	}
	if report.DaysWithEntries != 2 {
		t.Fatalf("expected 2 days with entries, got %d", report.DaysWithEntries)
	}
	if report.Adherence.EvaluatedDays != 2 {
		t.Fatalf("expected 2 adherence evaluated days, got %d", report.Adherence.EvaluatedDays)
	}
	if len(report.ByCategory) != 3 {
		t.Fatalf("expected 3 categories in breakdown, got %d", len(report.ByCategory))
	}
	if report.Body.MeasurementsCount != 2 {
		t.Fatalf("expected 2 body points, got %d", report.Body.MeasurementsCount)
	}
	if report.Body.GoalProgress == nil {
		t.Fatalf("expected goal progress in body summary")
	}
	if report.Metadata.SourceCounts["manual"] != 3 {
		t.Fatalf("expected metadata source count for manual=3, got %+v", report.Metadata.SourceCounts)
	}
	if report.Days[0].ExerciseCalories != 300 {
		t.Fatalf("expected first day exercise calories 300, got %+v", report.Days[0])
	}
	if report.Days[0].EffectiveGoalCalories != 2300 {
		t.Fatalf("expected first day effective goal calories 2300, got %+v", report.Days[0])
	}
	if report.HighestDay == nil || report.HighestDay.EffectiveGoalCalories != 2000 {
		t.Fatalf("expected highest day effective goal calories 2000, got %+v", report.HighestDay)
	}
	if report.LowestDay == nil || report.LowestDay.EffectiveGoalCalories != 2300 {
		t.Fatalf("expected lowest day effective goal calories 2300, got %+v", report.LowestDay)
	}
}

func TestAnalyticsRangeIncludesExerciseOnlyDays(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if _, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "cycling",
		CaloriesBurned: 450,
		PerformedAt:    time.Date(2026, 2, 12, 18, 30, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("add exercise log: %v", err)
	}

	report, err := service.AnalyticsRange(
		db,
		time.Date(2026, 2, 12, 0, 0, 0, 0, time.Local),
		time.Date(2026, 2, 12, 0, 0, 0, 0, time.Local),
		0.10,
	)
	if err != nil {
		t.Fatalf("analytics range: %v", err)
	}

	if len(report.Days) != 1 {
		t.Fatalf("expected one day in report, got %d", len(report.Days))
	}
	day := report.Days[0]
	if day.IntakeCalories != 0 || day.ExerciseCalories != 450 || day.NetCalories != -450 {
		t.Fatalf("unexpected exercise-only day: %+v", day)
	}
}

func TestAnalyticsRangeZeroMacroGoalKeepsMacroTargetsStatic(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if err := service.SetGoal(db, service.SetGoalInput{
		Calories:      2000,
		ProteinG:      0,
		CarbsG:        0,
		FatG:          0,
		EffectiveDate: "2026-02-01",
	}); err != nil {
		t.Fatalf("set goal: %v", err)
	}
	if _, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:       "Meal",
		Calories:   1000,
		ProteinG:   0,
		CarbsG:     0,
		FatG:       0,
		Category:   "lunch",
		Consumed:   time.Date(2026, 2, 12, 12, 0, 0, 0, time.Local),
		SourceType: "manual",
	}); err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if _, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "walking",
		CaloriesBurned: 400,
		PerformedAt:    time.Date(2026, 2, 12, 18, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("add exercise log: %v", err)
	}

	report, err := service.AnalyticsRange(
		db,
		time.Date(2026, 2, 12, 0, 0, 0, 0, time.Local),
		time.Date(2026, 2, 12, 0, 0, 0, 0, time.Local),
		0.10,
	)
	if err != nil {
		t.Fatalf("analytics range: %v", err)
	}
	day := report.Days[0]
	if day.EffectiveGoalCalories != 2400 {
		t.Fatalf("expected effective goal calories 2400, got %d", day.EffectiveGoalCalories)
	}
	if day.EffectiveGoalProtein != 0 || day.EffectiveGoalCarbs != 0 || day.EffectiveGoalFat != 0 {
		t.Fatalf("expected unchanged zero macro targets, got %+v", day)
	}
}

func TestAnalyticsInsightsRangeAutoGranularityBoundaries(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	cases := []struct {
		name string
		from time.Time
		to   time.Time
		want service.InsightsGranularity
	}{
		{
			name: "31 days is daily",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
			to:   time.Date(2026, 1, 31, 0, 0, 0, 0, time.Local),
			want: service.InsightsGranularityDay,
		},
		{
			name: "32 days is weekly",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
			to:   time.Date(2026, 2, 1, 0, 0, 0, 0, time.Local),
			want: service.InsightsGranularityWeek,
		},
		{
			name: "120 days is weekly",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
			to:   time.Date(2026, 4, 30, 0, 0, 0, 0, time.Local),
			want: service.InsightsGranularityWeek,
		},
		{
			name: "121 days is monthly",
			from: time.Date(2026, 1, 1, 0, 0, 0, 0, time.Local),
			to:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.Local),
			want: service.InsightsGranularityMonth,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			report, err := service.AnalyticsInsightsRange(db, tc.from, tc.to, 0.10, service.InsightsGranularityAuto)
			if err != nil {
				t.Fatalf("analytics insights range: %v", err)
			}
			if report.Granularity != tc.want {
				t.Fatalf("expected granularity %s, got %s", tc.want, report.Granularity)
			}
		})
	}
}

func TestAnalyticsInsightsRangeComputesDeltasAndSeries(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if _, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:       "Current Day",
		Calories:   1200,
		ProteinG:   100,
		CarbsG:     120,
		FatG:       30,
		Category:   "lunch",
		Consumed:   time.Date(2026, 2, 10, 12, 0, 0, 0, time.Local),
		SourceType: "manual",
	}); err != nil {
		t.Fatalf("create current entry: %v", err)
	}
	if _, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:       "Previous Day",
		Calories:   800,
		ProteinG:   60,
		CarbsG:     90,
		FatG:       20,
		Category:   "lunch",
		Consumed:   time.Date(2026, 2, 9, 12, 0, 0, 0, time.Local),
		SourceType: "manual",
	}); err != nil {
		t.Fatalf("create previous entry: %v", err)
	}
	if _, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "cycling",
		CaloriesBurned: 300,
		PerformedAt:    time.Date(2026, 2, 10, 18, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("add exercise log: %v", err)
	}

	report, err := service.AnalyticsInsightsRange(
		db,
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.Local),
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.Local),
		0.10,
		service.InsightsGranularityDay,
	)
	if err != nil {
		t.Fatalf("analytics insights range: %v", err)
	}

	if report.Current.TotalDays != 1 {
		t.Fatalf("expected total days 1, got %d", report.Current.TotalDays)
	}
	if report.Current.TotalIntakeCalories != 1200 || report.Current.TotalExerciseCalories != 300 || report.Current.TotalNetCalories != 900 {
		t.Fatalf("unexpected current totals: %+v", report.Current)
	}
	if report.Deltas.AvgIntakeCaloriesPerDay.AbsDelta != 400 {
		t.Fatalf("expected intake delta 400, got %+v", report.Deltas.AvgIntakeCaloriesPerDay)
	}
	if report.Deltas.AvgIntakeCaloriesPerDay.PctDelta == nil {
		t.Fatalf("expected non-nil pct delta when previous is non-zero")
	}
	if len(report.Series) != 1 {
		t.Fatalf("expected one series bucket, got %d", len(report.Series))
	}
	if report.Series[0].IntakeCalories != 1200 || report.Series[0].ExerciseCalories != 300 || report.Series[0].NetCalories != 900 {
		t.Fatalf("unexpected series bucket: %+v", report.Series[0])
	}
}

func TestAnalyticsInsightsRangeHandlesMissingPreviousData(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if _, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "running",
		CaloriesBurned: 450,
		PerformedAt:    time.Date(2026, 2, 12, 18, 0, 0, 0, time.Local),
	}); err != nil {
		t.Fatalf("add exercise log: %v", err)
	}

	report, err := service.AnalyticsInsightsRange(
		db,
		time.Date(2026, 2, 12, 0, 0, 0, 0, time.Local),
		time.Date(2026, 2, 12, 0, 0, 0, 0, time.Local),
		0.10,
		service.InsightsGranularityDay,
	)
	if err != nil {
		t.Fatalf("analytics insights range: %v", err)
	}

	if report.Current.TotalDays != 1 || report.Current.ExerciseActiveDays != 1 || report.Current.IntakeActiveDays != 0 {
		t.Fatalf("unexpected activity rates: %+v", report.Current)
	}
	if report.Current.TotalNetCalories != -450 {
		t.Fatalf("expected net calories -450, got %d", report.Current.TotalNetCalories)
	}
	if report.Deltas.AvgNetCaloriesPerDay.PctDelta != nil {
		t.Fatalf("expected nil pct delta when previous avg is zero, got %+v", report.Deltas.AvgNetCaloriesPerDay)
	}
}

func TestAnalyticsInsightsRangeIncludesStreaksRollingAndCategoryTrends(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if err := service.SetGoal(db, service.SetGoalInput{
		Calories:      2000,
		ProteinG:      150,
		CarbsG:        200,
		FatG:          70,
		EffectiveDate: "2026-01-01",
	}); err != nil {
		t.Fatalf("set goal: %v", err)
	}

	previousDays := []struct {
		date string
		kcal int
		cat  string
	}{
		{"2026-02-02", 1200, "lunch"},
		{"2026-02-03", 1250, "lunch"},
		{"2026-02-04", 1300, "lunch"},
		{"2026-02-05", 1350, "lunch"},
		{"2026-02-06", 1400, "lunch"},
		{"2026-02-07", 1450, "lunch"},
		{"2026-02-08", 1500, "lunch"},
		{"2026-02-09", 1550, "lunch"},
	}
	for _, d := range previousDays {
		if _, err := service.CreateEntry(db, service.CreateEntryInput{
			Name:       "Previous",
			Calories:   d.kcal,
			ProteinG:   90,
			CarbsG:     120,
			FatG:       35,
			Category:   d.cat,
			Consumed:   mustDateTime(t, d.date, "12:00"),
			SourceType: "manual",
		}); err != nil {
			t.Fatalf("create previous entry: %v", err)
		}
	}

	currentDays := []struct {
		date string
		kcal int
		cat  string
		ex   int
	}{
		{"2026-02-10", 2100, "lunch", 0},    // fail goal
		{"2026-02-11", 1900, "lunch", 0},    // pass
		{"2026-02-12", 1800, "lunch", 0},    // pass
		{"2026-02-13", 1700, "lunch", 200},  // pass + exercise
		{"2026-02-14", 2300, "dinner", 250}, // fail
		{"2026-02-15", 1600, "dinner", 300}, // pass
		{"2026-02-16", 1500, "dinner", 320}, // pass
		{"2026-02-17", 1550, "dinner", 340}, // pass
	}
	for _, d := range currentDays {
		if _, err := service.CreateEntry(db, service.CreateEntryInput{
			Name:       "Current",
			Calories:   d.kcal,
			ProteinG:   100,
			CarbsG:     130,
			FatG:       40,
			Category:   d.cat,
			Consumed:   mustDateTime(t, d.date, "12:00"),
			SourceType: "manual",
		}); err != nil {
			t.Fatalf("create current entry: %v", err)
		}
		if d.ex > 0 {
			if _, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
				ExerciseType:   "running",
				CaloriesBurned: d.ex,
				PerformedAt:    mustDateTime(t, d.date, "18:00"),
			}); err != nil {
				t.Fatalf("add exercise log: %v", err)
			}
		}
	}

	report, err := service.AnalyticsInsightsRange(
		db,
		time.Date(2026, 2, 10, 0, 0, 0, 0, time.Local),
		time.Date(2026, 2, 17, 0, 0, 0, 0, time.Local),
		0.10,
		service.InsightsGranularityDay,
	)
	if err != nil {
		t.Fatalf("analytics insights range: %v", err)
	}

	if report.Streaks.Logging.Current != 8 || report.Streaks.Logging.Longest != 8 {
		t.Fatalf("unexpected logging streaks: %+v", report.Streaks.Logging)
	}
	if report.Streaks.Exercise.Current != 5 || report.Streaks.Exercise.Longest != 5 {
		t.Fatalf("unexpected exercise streaks: %+v", report.Streaks.Exercise)
	}
	if report.RollingWindows.Window7.Latest == nil {
		t.Fatalf("expected 7-day rolling latest point")
	}
	if report.RollingWindows.Window30.Latest != nil {
		t.Fatalf("expected nil 30-day latest point for 8-day range")
	}
	if len(report.CategoryTrends) == 0 {
		t.Fatalf("expected category trends")
	}
	if report.CategoryTrends[0].Category == "" {
		t.Fatalf("expected category name in trend: %+v", report.CategoryTrends[0])
	}
}

func mustDateTime(t *testing.T, date, clock string) time.Time {
	t.Helper()
	out, err := time.ParseInLocation("2006-01-02 15:04", date+" "+clock, time.Local)
	if err != nil {
		t.Fatalf("parse datetime %s %s: %v", date, clock, err)
	}
	return out
}
