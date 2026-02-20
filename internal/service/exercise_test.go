package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
)

func TestExerciseLogCRUD(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	distance := 5.5
	duration := 40
	id, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "running",
		CaloriesBurned: 400,
		DurationMin:    &duration,
		Distance:       &distance,
		DistanceUnit:   "km",
		PerformedAt:    time.Date(2026, 2, 20, 18, 0, 0, 0, time.Local),
		Notes:          "easy run",
		Metadata:       `{"intensity":"easy"}`,
	})
	if err != nil {
		t.Fatalf("create exercise log: %v", err)
	}

	items, err := service.ListExerciseLogs(db, service.ListExerciseFilter{Date: "2026-02-20"})
	if err != nil {
		t.Fatalf("list exercise logs: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 exercise log, got %d", len(items))
	}
	if items[0].CaloriesBurned != 400 || items[0].ExerciseType != "running" {
		t.Fatalf("unexpected exercise row: %+v", items[0])
	}

	duration = 50
	distance = 6.1
	if err := service.UpdateExerciseLog(db, service.UpdateExerciseInput{
		ID: id,
		ExerciseLogInput: service.ExerciseLogInput{
			ExerciseType:   "running",
			CaloriesBurned: 480,
			DurationMin:    &duration,
			Distance:       &distance,
			DistanceUnit:   "km",
			PerformedAt:    time.Date(2026, 2, 20, 19, 0, 0, 0, time.Local),
			Notes:          "progression",
		},
	}); err != nil {
		t.Fatalf("update exercise log: %v", err)
	}

	items, err = service.ListExerciseLogs(db, service.ListExerciseFilter{Date: "2026-02-20"})
	if err != nil {
		t.Fatalf("list exercise logs after update: %v", err)
	}
	if len(items) != 1 || items[0].CaloriesBurned != 480 {
		t.Fatalf("expected updated calories 480, got: %+v", items)
	}

	if err := service.DeleteExerciseLog(db, id); err != nil {
		t.Fatalf("delete exercise log: %v", err)
	}
	items, err = service.ListExerciseLogs(db, service.ListExerciseFilter{Date: "2026-02-20"})
	if err != nil {
		t.Fatalf("list exercise logs after delete: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected no exercise logs after delete, got %d", len(items))
	}
}

func TestExerciseLogValidation(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "",
		CaloriesBurned: 100,
	})
	if err == nil || !strings.Contains(err.Error(), "exercise type is required") {
		t.Fatalf("expected missing type error, got: %v", err)
	}

	_, err = service.CreateExerciseLog(db, service.ExerciseLogInput{
		ExerciseType:   "cycling",
		CaloriesBurned: 200,
		DistanceUnit:   "km",
	})
	if err == nil || !strings.Contains(err.Error(), "distance must be provided") {
		t.Fatalf("expected distance pairing error, got: %v", err)
	}
}
