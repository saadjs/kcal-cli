package service_test

import (
	"testing"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
)

func TestRecipeLogSnapshotsEntry(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	recipeID, err := service.CreateRecipe(db, service.RecipeInput{
		Name:          "Overnight oats",
		CaloriesTotal: 400,
		ProteinTotalG: 20,
		CarbsTotalG:   50,
		FatTotalG:     10,
		Servings:      2,
	})
	if err != nil {
		t.Fatalf("create recipe: %v", err)
	}

	entryID, err := service.LogRecipe(db, service.LogRecipeInput{
		RecipeIdentifier: "Overnight oats",
		Servings:         1,
		Category:         "breakfast",
		ConsumedAt:       time.Date(2026, 2, 20, 8, 0, 0, 0, time.Local),
	})
	if err != nil {
		t.Fatalf("log recipe: %v", err)
	}
	if entryID <= 0 {
		t.Fatalf("expected logged entry id > 0")
	}

	if err := service.UpdateRecipe(db, "Overnight oats", service.RecipeInput{
		Name:          "Overnight oats",
		CaloriesTotal: 600,
		ProteinTotalG: 30,
		CarbsTotalG:   80,
		FatTotalG:     12,
		Servings:      2,
	}); err != nil {
		t.Fatalf("update recipe: %v", err)
	}

	entries, err := service.ListEntries(db, service.ListEntriesFilter{Category: "breakfast"})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one logged recipe entry, got %d", len(entries))
	}
	if entries[0].Calories != 200 {
		t.Fatalf("expected snapshot calories 200, got %d", entries[0].Calories)
	}
	if entries[0].SourceID == nil || *entries[0].SourceID != recipeID {
		t.Fatalf("expected source id %d, got %+v", recipeID, entries[0].SourceID)
	}
}

func TestResolveRecipeRejectsPartialNumericIdentifier(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.CreateRecipe(db, service.RecipeInput{
		Name:          "oats",
		CaloriesTotal: 100,
		ProteinTotalG: 10,
		CarbsTotalG:   10,
		FatTotalG:     5,
		Servings:      1,
	})
	if err != nil {
		t.Fatalf("create recipe: %v", err)
	}

	_, err = service.ResolveRecipe(db, "1abc")
	if err == nil {
		t.Fatalf("expected invalid partial numeric identifier to fail")
	}
}
