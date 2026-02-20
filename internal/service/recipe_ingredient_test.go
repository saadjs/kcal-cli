package service_test

import (
	"testing"

	"github.com/saadjs/kcal-cli/internal/service"
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

func TestRecipeIngredientScalingOutputCanBePersisted(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.CreateRecipe(db, service.RecipeInput{
		Name:          "Smoothie",
		CaloriesTotal: 0,
		ProteinTotalG: 0,
		CarbsTotalG:   0,
		FatTotalG:     0,
		Servings:      1,
	})
	if err != nil {
		t.Fatalf("create recipe: %v", err)
	}

	scaled, err := service.ScaleIngredientMacros(service.ScaleIngredientMacrosInput{
		Amount:      2,
		Unit:        "tbsp",
		RefAmount:   32,
		RefUnit:     "g",
		RefCalories: 190,
		RefProteinG: 7,
		RefCarbsG:   8,
		RefFatG:     16,
		DensityGML:  1.05,
	})
	if err != nil {
		t.Fatalf("scale ingredient macros: %v", err)
	}

	_, err = service.AddRecipeIngredient(db, "Smoothie", service.RecipeIngredientInput{
		Name:       "Peanut Butter",
		Amount:     2,
		AmountUnit: "tbsp",
		Calories:   scaled.Calories,
		ProteinG:   scaled.ProteinG,
		CarbsG:     scaled.CarbsG,
		FatG:       scaled.FatG,
	})
	if err != nil {
		t.Fatalf("add scaled ingredient: %v", err)
	}
}
