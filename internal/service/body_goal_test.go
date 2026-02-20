package service_test

import (
	"testing"

	"github.com/saad/kcal-cli/internal/service"
)

func TestBodyGoalVersioning(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if err := service.SetBodyGoal(db, service.SetBodyGoalInput{TargetWeight: 80, Unit: "kg", TargetBodyFat: floatPtr(20), EffectiveDate: "2026-01-01"}); err != nil {
		t.Fatalf("set first body goal: %v", err)
	}
	if err := service.SetBodyGoal(db, service.SetBodyGoalInput{TargetWeight: 75, Unit: "kg", TargetBodyFat: floatPtr(18), EffectiveDate: "2026-02-01"}); err != nil {
		t.Fatalf("set second body goal: %v", err)
	}

	jan, err := service.CurrentBodyGoal(db, "2026-01-15")
	if err != nil {
		t.Fatalf("current jan body goal: %v", err)
	}
	if jan == nil || jan.TargetWeightKg != 80 {
		t.Fatalf("expected jan target weight 80kg, got %+v", jan)
	}

	feb, err := service.CurrentBodyGoal(db, "2026-02-20")
	if err != nil {
		t.Fatalf("current feb body goal: %v", err)
	}
	if feb == nil || feb.TargetWeightKg != 75 {
		t.Fatalf("expected feb target weight 75kg, got %+v", feb)
	}
}
