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
}
