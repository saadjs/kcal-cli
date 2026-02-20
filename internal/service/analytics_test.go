package service_test

import (
	"testing"
	"time"

	"github.com/saad/kcal-cli/internal/service"
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
