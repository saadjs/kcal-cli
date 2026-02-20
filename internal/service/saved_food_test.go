package service_test

import (
	"strconv"
	"testing"
	"time"

	"github.com/saad/kcal-cli/internal/service"
)

func TestSavedFoodLifecycleAndLog(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	id, err := service.CreateSavedFood(db, service.CreateSavedFoodInput{
		Name:       "Greek Yogurt",
		Category:   "breakfast",
		Calories:   150,
		ProteinG:   15,
		CarbsG:     10,
		FatG:       5,
		ServingAmt: 1,
	})
	if err != nil {
		t.Fatalf("create saved food: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected saved food id > 0")
	}

	items, err := service.ListSavedFoods(db, service.ListSavedFoodsFilter{})
	if err != nil {
		t.Fatalf("list saved foods: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 saved food, got %d", len(items))
	}

	entryID, err := service.LogSavedFood(db, service.LogSavedFoodInput{
		Identifier: "Greek Yogurt",
		Servings:   2,
		ConsumedAt: time.Date(2026, 2, 20, 8, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("log saved food: %v", err)
	}
	if entryID <= 0 {
		t.Fatalf("expected entry id > 0")
	}
	entry, err := service.EntryByID(db, entryID)
	if err != nil {
		t.Fatalf("entry by id: %v", err)
	}
	if entry.SourceType != "saved_food" {
		t.Fatalf("expected source_type=saved_food, got %s", entry.SourceType)
	}
	if entry.SourceID == nil || *entry.SourceID != id {
		t.Fatalf("expected source id %d, got %+v", id, entry.SourceID)
	}
	if entry.Calories != 300 {
		t.Fatalf("expected calories 300, got %d", entry.Calories)
	}

	food, err := service.ResolveSavedFood(db, "Greek Yogurt")
	if err != nil {
		t.Fatalf("resolve saved food: %v", err)
	}
	if food.UsageCount != 1 {
		t.Fatalf("expected usage_count 1, got %d", food.UsageCount)
	}

	if err := service.ArchiveSavedFood(db, "Greek Yogurt"); err != nil {
		t.Fatalf("archive saved food: %v", err)
	}
	items, err = service.ListSavedFoods(db, service.ListSavedFoodsFilter{})
	if err != nil {
		t.Fatalf("list after archive: %v", err)
	}
	if len(items) != 0 {
		t.Fatalf("expected archived food hidden from default list, got %d", len(items))
	}

	if err := service.RestoreSavedFood(db, "Greek Yogurt"); err != nil {
		t.Fatalf("restore saved food: %v", err)
	}
	items, err = service.ListSavedFoods(db, service.ListSavedFoodsFilter{})
	if err != nil {
		t.Fatalf("list after restore: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected restored food visible, got %d", len(items))
	}
}

func TestCreateSavedFoodFromEntrySnapshot(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	entryID, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:     "Chicken Bowl",
		Calories: 550,
		ProteinG: 45,
		CarbsG:   40,
		FatG:     18,
		Category: "lunch",
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}

	savedID, err := service.CreateSavedFoodFromEntry(db, entryID, "", "")
	if err != nil {
		t.Fatalf("create saved food from entry: %v", err)
	}
	food, err := service.ResolveSavedFood(db, strconv.FormatInt(savedID, 10))
	if err != nil {
		t.Fatalf("resolve saved food: %v", err)
	}
	if food.Calories != 550 || food.ProteinG != 45 {
		t.Fatalf("expected snapshot nutrition from entry, got kcal=%d protein=%.1f", food.Calories, food.ProteinG)
	}
	if food.SourceType != "entry" {
		t.Fatalf("expected source type entry, got %s", food.SourceType)
	}
}
