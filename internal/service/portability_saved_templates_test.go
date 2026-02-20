package service_test

import (
	"path/filepath"
	"testing"

	"github.com/saadjs/kcal-cli/internal/db"
	"github.com/saadjs/kcal-cli/internal/service"
)

func TestExportImportSavedTemplates(t *testing.T) {
	t.Parallel()
	src := newTestDB(t)
	defer src.Close()

	_, err := service.CreateSavedFood(src, service.CreateSavedFoodInput{
		Name:     "Greek Yogurt",
		Category: "breakfast",
		Calories: 150,
		ProteinG: 15,
		CarbsG:   10,
		FatG:     5,
	})
	if err != nil {
		t.Fatalf("create saved food: %v", err)
	}
	_, err = service.CreateSavedMeal(src, service.CreateSavedMealInput{Name: "Yogurt bowl", Category: "breakfast"})
	if err != nil {
		t.Fatalf("create saved meal: %v", err)
	}
	if _, err := service.AddSavedMealComponent(src, "Yogurt bowl", service.SavedMealComponentInput{SavedFoodIdentifier: "Greek Yogurt", Position: 1}); err != nil {
		t.Fatalf("add component: %v", err)
	}

	exported, err := service.ExportDataSnapshot(src)
	if err != nil {
		t.Fatalf("export snapshot: %v", err)
	}
	if len(exported.SavedFoods) == 0 || len(exported.SavedMeals) == 0 || len(exported.SavedMealComponents) == 0 {
		t.Fatalf("expected saved templates in export payload")
	}

	dstPath := filepath.Join(t.TempDir(), "dst.db")
	dst, err := db.Open(dstPath)
	if err != nil {
		t.Fatalf("open dst db: %v", err)
	}
	defer dst.Close()
	if err := db.ApplyMigrations(dst); err != nil {
		t.Fatalf("apply migrations on dst: %v", err)
	}

	report, err := service.ImportDataSnapshotWithOptions(dst, exported, service.ImportOptions{Mode: service.ImportModeMerge})
	if err != nil {
		t.Fatalf("import snapshot: %v", err)
	}
	if report.Conflicts != 0 {
		t.Fatalf("expected no conflicts, got %d", report.Conflicts)
	}

	foods, err := service.ListSavedFoods(dst, service.ListSavedFoodsFilter{})
	if err != nil {
		t.Fatalf("list saved foods in dst: %v", err)
	}
	if len(foods) != 1 {
		t.Fatalf("expected 1 saved food in dst, got %d", len(foods))
	}
	meals, err := service.ListSavedMeals(dst, service.ListSavedMealsFilter{})
	if err != nil {
		t.Fatalf("list saved meals in dst: %v", err)
	}
	if len(meals) != 1 {
		t.Fatalf("expected 1 saved meal in dst, got %d", len(meals))
	}
	comps, err := service.ListSavedMealComponents(dst, "Yogurt bowl")
	if err != nil {
		t.Fatalf("list meal components in dst: %v", err)
	}
	if len(comps) != 1 {
		t.Fatalf("expected 1 component in dst, got %d", len(comps))
	}
}
