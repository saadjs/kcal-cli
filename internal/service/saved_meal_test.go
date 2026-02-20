package service_test

import (
	"testing"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
)

func TestSavedMealComponentsRecalcAndLog(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	foodID, err := service.CreateSavedFood(db, service.CreateSavedFoodInput{
		Name:     "Oats",
		Category: "breakfast",
		Calories: 150,
		ProteinG: 5,
		CarbsG:   27,
		FatG:     3,
	})
	if err != nil {
		t.Fatalf("create saved food: %v", err)
	}
	if foodID <= 0 {
		t.Fatalf("expected saved food id > 0")
	}

	mealID, err := service.CreateSavedMeal(db, service.CreateSavedMealInput{
		Name:     "Overnight oats",
		Category: "breakfast",
	})
	if err != nil {
		t.Fatalf("create saved meal: %v", err)
	}

	if _, err := service.AddSavedMealComponent(db, "Overnight oats", service.SavedMealComponentInput{
		SavedFoodIdentifier: "Oats",
		Quantity:            1,
		Unit:                "serving",
		Position:            1,
	}); err != nil {
		t.Fatalf("add component from saved food: %v", err)
	}
	if _, err := service.AddSavedMealComponent(db, "Overnight oats", service.SavedMealComponentInput{
		Name:     "Milk",
		Quantity: 200,
		Unit:     "ml",
		Calories: 80,
		ProteinG: 5,
		CarbsG:   10,
		FatG:     2,
		Position: 2,
	}); err != nil {
		t.Fatalf("add manual component: %v", err)
	}

	meal, err := service.ResolveSavedMeal(db, "Overnight oats")
	if err != nil {
		t.Fatalf("resolve meal: %v", err)
	}
	if meal.CaloriesTotal != 230 {
		t.Fatalf("expected meal calories 230, got %d", meal.CaloriesTotal)
	}
	if meal.ProteinTotalG != 10 {
		t.Fatalf("expected meal protein 10, got %.1f", meal.ProteinTotalG)
	}

	entryID, err := service.LogSavedMeal(db, service.LogSavedMealInput{
		Identifier: "Overnight oats",
		Servings:   2,
		ConsumedAt: time.Date(2026, 2, 20, 8, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("log saved meal: %v", err)
	}
	entry, err := service.EntryByID(db, entryID)
	if err != nil {
		t.Fatalf("entry by id: %v", err)
	}
	if entry.SourceType != "saved_meal" {
		t.Fatalf("expected saved_meal source type, got %s", entry.SourceType)
	}
	if entry.SourceID == nil || *entry.SourceID != mealID {
		t.Fatalf("expected source id %d, got %+v", mealID, entry.SourceID)
	}
	if entry.Calories != 460 {
		t.Fatalf("expected logged calories 460, got %d", entry.Calories)
	}

	if err := service.ArchiveSavedMeal(db, "Overnight oats"); err != nil {
		t.Fatalf("archive meal: %v", err)
	}
	items, err := service.ListSavedMeals(db, service.ListSavedMealsFilter{})
	if err != nil {
		t.Fatalf("list meals: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected archived meal hidden, got %d", len(items))
	}
}
