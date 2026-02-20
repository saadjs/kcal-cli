package service_test

import (
	"testing"

	"github.com/saad/kcal-cli/internal/service"
)

func TestRecipeIngredientRecalculation(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.CreateRecipe(db, service.RecipeInput{
		Name:          "Burrito",
		CaloriesTotal: 0,
		ProteinTotalG: 0,
		CarbsTotalG:   0,
		FatTotalG:     0,
		Servings:      2,
	})
	if err != nil {
		t.Fatalf("create recipe: %v", err)
	}

	_, err = service.AddRecipeIngredient(db, "Burrito", service.RecipeIngredientInput{Name: "Rice", Amount: 100, AmountUnit: "g", Calories: 130, ProteinG: 2.4, CarbsG: 28, FatG: 0.3})
	if err != nil {
		t.Fatalf("add ingredient 1: %v", err)
	}
	_, err = service.AddRecipeIngredient(db, "Burrito", service.RecipeIngredientInput{Name: "Chicken", Amount: 150, AmountUnit: "g", Calories: 240, ProteinG: 45, CarbsG: 0, FatG: 5})
	if err != nil {
		t.Fatalf("add ingredient 2: %v", err)
	}

	if err := service.RecalculateRecipeTotals(db, "Burrito"); err != nil {
		t.Fatalf("recalculate recipe totals: %v", err)
	}

	recipe, err := service.ResolveRecipe(db, "Burrito")
	if err != nil {
		t.Fatalf("resolve recipe: %v", err)
	}
	if recipe.CaloriesTotal != 370 {
		t.Fatalf("expected recipe calories 370, got %d", recipe.CaloriesTotal)
	}
	if recipe.ProteinTotalG < 47 || recipe.ProteinTotalG > 48 {
		t.Fatalf("unexpected recipe protein total %.2f", recipe.ProteinTotalG)
	}
}
