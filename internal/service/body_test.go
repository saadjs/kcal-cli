package service_test

import (
	"testing"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
)

func TestBodyMeasurementCRUDAndUnitConversion(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	id, err := service.AddBodyMeasurement(db, service.BodyMeasurementInput{
		Weight:     180,
		Unit:       "lb",
		BodyFatPct: floatPtr(22.5),
		MeasuredAt: time.Date(2026, 2, 20, 8, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("add body measurement: %v", err)
	}

	items, err := service.ListBodyMeasurements(db, service.BodyMeasurementFilter{Date: "2026-02-20"})
	if err != nil {
		t.Fatalf("list body measurements: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 body measurement, got %d", len(items))
	}
	if items[0].ID != id {
		t.Fatalf("expected measurement id %d, got %d", id, items[0].ID)
	}
	if items[0].WeightKg < 81 || items[0].WeightKg > 82 {
		t.Fatalf("expected converted weight around 81.6kg, got %.4f", items[0].WeightKg)
	}

	if err := service.UpdateBodyMeasurement(db, service.UpdateBodyMeasurementInput{
		ID: id,
		BodyMeasurementInput: service.BodyMeasurementInput{
			Weight:     80,
			Unit:       "kg",
			BodyFatPct: floatPtr(20),
			MeasuredAt: time.Date(2026, 2, 21, 8, 0, 0, 0, time.Local),
		},
	}); err != nil {
		t.Fatalf("update body measurement: %v", err)
	}

	if err := service.DeleteBodyMeasurement(db, id); err != nil {
		t.Fatalf("delete body measurement: %v", err)
	}
}

func TestBodyMeasurementRejectsInvalidBodyFat(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.AddBodyMeasurement(db, service.BodyMeasurementInput{Weight: 80, Unit: "kg", BodyFatPct: floatPtr(101)})
	if err == nil {
		t.Fatalf("expected invalid body-fat to fail")
	}
}

func floatPtr(v float64) *float64 {
	return &v
}
