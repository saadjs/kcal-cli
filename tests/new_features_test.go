package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestBodyCommandsAndAnalyticsJSONBodySection(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"body-goal", "set",
		"--target-weight", "170",
		"--unit", "lb",
		"--target-body-fat", "18",
		"--effective-date", "2026-02-01",
	)
	if exit != 0 {
		t.Fatalf("body-goal set failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"body", "add",
		"--weight", "172",
		"--unit", "lb",
		"--body-fat", "20",
		"--date", "2026-02-20",
		"--time", "07:30",
	)
	if exit != 0 {
		t.Fatalf("body add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"body", "add",
		"--weight", "171",
		"--unit", "lb",
		"--body-fat", "19.5",
		"--date", "2026-02-21",
		"--time", "07:30",
	)
	if exit != 0 {
		t.Fatalf("body add #2 failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "body", "list", "--unit", "lb")
	if exit != 0 {
		t.Fatalf("body list failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "WEIGHT") || !strings.Contains(stdout, "lb") {
		t.Fatalf("expected body list output in lb, got:\n%s", stdout)
	}

	stdout, stderr, exit = runKcal(t, binPath, dbPath,
		"analytics", "range", "--from", "2026-02-20", "--to", "2026-02-21", "--json",
	)
	if exit != 0 {
		t.Fatalf("analytics json failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, `"body"`) || !strings.Contains(stdout, `"measurements_count"`) {
		t.Fatalf("expected body section in analytics json, got:\n%s", stdout)
	}
}

func TestRecipeIngredientLifecycleAndRecalc(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"recipe", "add",
		"--name", "Burrito",
		"--calories", "0",
		"--protein", "0",
		"--carbs", "0",
		"--fat", "0",
		"--servings", "2",
	)
	if exit != 0 {
		t.Fatalf("recipe add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"recipe", "ingredient", "add", "Burrito",
		"--name", "Rice", "--amount", "100", "--unit", "g",
		"--calories", "130", "--protein", "2.4", "--carbs", "28", "--fat", "0.3",
	)
	if exit != 0 {
		t.Fatalf("ingredient add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"recipe", "ingredient", "add", "Burrito",
		"--name", "Chicken", "--amount", "150", "--unit", "g",
		"--calories", "240", "--protein", "45", "--carbs", "0", "--fat", "5",
	)
	if exit != 0 {
		t.Fatalf("ingredient add #2 failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "recipe", "recalc", "Burrito")
	if exit != 0 {
		t.Fatalf("recipe recalc failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "recipe", "show", "Burrito")
	if exit != 0 {
		t.Fatalf("recipe show failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "Calories Total: 370") {
		t.Fatalf("expected recalculated calories in recipe show output, got:\n%s", stdout)
	}
}

func TestRecipeIngredientScalingAndDensityValidation(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"recipe", "add",
		"--name", "PB Smoothie",
		"--calories", "0",
		"--protein", "0",
		"--carbs", "0",
		"--fat", "0",
		"--servings", "1",
	)
	if exit != 0 {
		t.Fatalf("recipe add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"recipe", "ingredient", "add", "PB Smoothie",
		"--name", "Peanut Butter",
		"--amount", "2",
		"--unit", "tbsp",
		"--ref-amount", "32",
		"--ref-unit", "g",
		"--ref-calories", "190",
		"--ref-protein", "7",
		"--ref-carbs", "8",
		"--ref-fat", "16",
	)
	if exit == 0 {
		t.Fatalf("expected mass/volume scaling without density to fail")
	}
	if !strings.Contains(stderr, "density-g-per-ml must be > 0") {
		t.Fatalf("expected density error, got: %s", stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"recipe", "ingredient", "add", "PB Smoothie",
		"--name", "Peanut Butter",
		"--amount", "2",
		"--unit", "tbsp",
		"--ref-amount", "32",
		"--ref-unit", "g",
		"--ref-calories", "190",
		"--ref-protein", "7",
		"--ref-carbs", "8",
		"--ref-fat", "16",
		"--density-g-per-ml", "1.05",
	)
	if exit != 0 {
		t.Fatalf("expected scaling with density to succeed: exit=%d stderr=%s", exit, stderr)
	}
}

func TestBarcodeOverrideIsUsedByLookup(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"lookup", "override", "set", "3017620422003",
		"--provider", "openfoodfacts",
		"--name", "Nutella Custom",
		"--brand", "Ferrero",
		"--serving-amount", "15",
		"--serving-unit", "g",
		"--calories", "99",
		"--protein", "1",
		"--carbs", "10",
		"--fat", "6",
	)
	if exit != 0 {
		t.Fatalf("override set failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath,
		"lookup", "barcode", "3017620422003", "--provider", "openfoodfacts",
	)
	if exit != 0 {
		t.Fatalf("lookup with override failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "Provider: openfoodfacts (override)") {
		t.Fatalf("expected override source output, got: %s", stdout)
	}
	if !strings.Contains(stdout, "Food: Nutella Custom") || !strings.Contains(stdout, "Calories: 99.0") {
		t.Fatalf("expected override nutrition output, got: %s", stdout)
	}
}

func TestEntryAddWithBarcodeUsesOverrideAndServings(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "lookup", "override", "set", "3017620422003",
		"--provider", "openfoodfacts",
		"--name", "Nutella Custom",
		"--brand", "Ferrero",
		"--serving-amount", "15",
		"--serving-unit", "g",
		"--calories", "100",
		"--protein", "1",
		"--carbs", "10",
		"--fat", "6",
	)
	if exit != 0 {
		t.Fatalf("set override failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "entry", "add",
		"--barcode", "3017620422003",
		"--provider", "openfoodfacts",
		"--servings", "1.5",
		"--category", "snacks",
		"--date", "2026-02-20",
		"--time", "12:00",
	)
	if exit != 0 {
		t.Fatalf("entry add --barcode failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "entry", "list", "--date", "2026-02-20")
	if exit != 0 {
		t.Fatalf("entry list failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "barcode") {
		t.Fatalf("expected barcode source in entry list, got: %s", stdout)
	}
	if !strings.Contains(stdout, "\t150\t") {
		t.Fatalf("expected scaled calories 150 in entry list, got: %s", stdout)
	}
}

func TestEntryAddBarcodeRejectsManualNutritionFlags(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "entry", "add",
		"--barcode", "3017620422003",
		"--provider", "openfoodfacts",
		"--name", "conflict",
		"--category", "snacks",
	)
	if exit == 0 {
		t.Fatalf("expected conflicting barcode/manual flags to fail")
	}
	if !strings.Contains(stderr, "cannot combine --barcode with manual nutrition flags") {
		t.Fatalf("expected conflict validation error, got: %s", stderr)
	}
}
