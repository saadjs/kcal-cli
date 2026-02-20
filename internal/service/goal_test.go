package service_test

import (
	"testing"

	"github.com/saad/kcal-cli/internal/service"
)

func TestGoalVersioningByEffectiveDate(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if err := service.SetGoal(db, service.SetGoalInput{
		Calories:      2000,
		ProteinG:      150,
		CarbsG:        220,
		FatG:          70,
		EffectiveDate: "2026-01-01",
	}); err != nil {
		t.Fatalf("set first goal: %v", err)
	}
	if err := service.SetGoal(db, service.SetGoalInput{
		Calories:      1800,
		ProteinG:      160,
		CarbsG:        180,
		FatG:          60,
		EffectiveDate: "2026-02-01",
	}); err != nil {
		t.Fatalf("set second goal: %v", err)
	}

	january, err := service.CurrentGoal(db, "2026-01-15")
	if err != nil {
		t.Fatalf("current january goal: %v", err)
	}
	if january == nil || january.Calories != 2000 {
		t.Fatalf("expected january goal calories 2000, got %+v", january)
	}

	february, err := service.CurrentGoal(db, "2026-02-10")
	if err != nil {
		t.Fatalf("current february goal: %v", err)
	}
	if february == nil || february.Calories != 1800 {
		t.Fatalf("expected february goal calories 1800, got %+v", february)
	}
}
