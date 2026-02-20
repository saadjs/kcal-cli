package tests

import (
	"os"
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

func TestExerciseCommandsAndAnalyticsJSONFields(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"goal", "set",
		"--calories", "2200",
		"--protein", "160",
		"--carbs", "240",
		"--fat", "70",
		"--effective-date", "2026-02-01",
	)
	if exit != 0 {
		t.Fatalf("goal set failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"entry", "add",
		"--name", "Lunch",
		"--calories", "900",
		"--protein", "50",
		"--carbs", "100",
		"--fat", "25",
		"--category", "lunch",
		"--date", "2026-02-20",
		"--time", "12:00",
	)
	if exit != 0 {
		t.Fatalf("entry add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"exercise", "add",
		"--type", "cycling",
		"--calories", "500",
		"--duration-min", "60",
		"--distance", "22.5",
		"--distance-unit", "km",
		"--date", "2026-02-20",
		"--time", "18:00",
		"--notes", "tempo ride",
	)
	if exit != 0 {
		t.Fatalf("exercise add failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "exercise", "list", "--date", "2026-02-20")
	if exit != 0 {
		t.Fatalf("exercise list failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "cycling") || !strings.Contains(stdout, "\t500\t") {
		t.Fatalf("expected exercise list output, got:\n%s", stdout)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"exercise", "update", "1",
		"--type", "cycling",
		"--calories", "520",
		"--duration-min", "62",
		"--distance", "23.2",
		"--distance-unit", "km",
		"--date", "2026-02-20",
		"--time", "18:10",
	)
	if exit != 0 {
		t.Fatalf("exercise update failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit = runKcal(t, binPath, dbPath, "analytics", "range", "--from", "2026-02-20", "--to", "2026-02-20", "--json")
	if exit != 0 {
		t.Fatalf("analytics json failed: exit=%d stderr=%s", exit, stderr)
	}
	for _, want := range []string{
		`"total_intake_calories"`,
		`"total_exercise_calories"`,
		`"total_net_calories"`,
		`"avg_intake_calories_per_day"`,
		`"avg_exercise_calories_per_day"`,
		`"avg_net_calories_per_day"`,
		`"intake_calories"`,
		`"exercise_calories"`,
		`"net_calories"`,
		`"effective_goal_calories"`,
		`"effective_goal_protein_g"`,
		`"effective_goal_carbs_g"`,
		`"effective_goal_fat_g"`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected analytics json to contain %q, got:\n%s", want, stdout)
		}
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "exercise", "delete", "1")
	if exit != 0 {
		t.Fatalf("exercise delete failed: exit=%d stderr=%s", exit, stderr)
	}
}

func TestAnalyticsInsightsCommandsAndOutputModes(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"goal", "set",
		"--calories", "2200",
		"--protein", "160",
		"--carbs", "240",
		"--fat", "70",
		"--effective-date", "2026-02-01",
	)
	if exit != 0 {
		t.Fatalf("goal set failed: exit=%d stderr=%s", exit, stderr)
	}

	for _, args := range [][]string{
		{"entry", "add", "--name", "Meal A", "--calories", "900", "--protein", "50", "--carbs", "100", "--fat", "25", "--category", "lunch", "--date", "2026-02-20", "--time", "12:00"},
		{"entry", "add", "--name", "Meal B", "--calories", "700", "--protein", "40", "--carbs", "80", "--fat", "20", "--category", "dinner", "--date", "2026-02-19", "--time", "19:00"},
	} {
		_, stderr, exit = runKcal(t, binPath, dbPath, args...)
		if exit != 0 {
			t.Fatalf("entry add failed: exit=%d stderr=%s args=%v", exit, stderr, args)
		}
	}
	_, stderr, exit = runKcal(t, binPath, dbPath,
		"exercise", "add",
		"--type", "running",
		"--calories", "300",
		"--date", "2026-02-20",
		"--time", "18:00",
	)
	if exit != 0 {
		t.Fatalf("exercise add failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath,
		"analytics", "insights", "range",
		"--from", "2026-02-20",
		"--to", "2026-02-20",
	)
	if exit != 0 {
		t.Fatalf("analytics insights range failed: exit=%d stderr=%s", exit, stderr)
	}
	for _, want := range []string{
		"Key Metrics",
		"Consistency",
		"Trends",
		"Streaks",
		"Rolling Windows",
		"Category Trends",
		"Charts",
		"Macro Sparklines",
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected insights output to contain %q, got:\n%s", want, stdout)
		}
	}

	stdout, stderr, exit = runKcal(t, binPath, dbPath,
		"analytics", "insights", "range",
		"--from", "2026-02-20",
		"--to", "2026-02-20",
		"--no-charts",
	)
	if exit != 0 {
		t.Fatalf("analytics insights range --no-charts failed: exit=%d stderr=%s", exit, stderr)
	}
	if strings.Contains(stdout, "Charts") {
		t.Fatalf("expected charts section to be suppressed with --no-charts, got:\n%s", stdout)
	}

	stdout, stderr, exit = runKcal(t, binPath, dbPath,
		"analytics", "insights", "range",
		"--from", "2026-02-20",
		"--to", "2026-02-20",
		"--json",
	)
	if exit != 0 {
		t.Fatalf("analytics insights json failed: exit=%d stderr=%s", exit, stderr)
	}
	for _, want := range []string{
		`"from_date"`,
		`"previous_from_date"`,
		`"granularity"`,
		`"current"`,
		`"deltas"`,
		`"consistency"`,
		`"trends"`,
		`"extremes"`,
		`"macro_balance"`,
		`"streaks"`,
		`"rolling_windows"`,
		`"category_trends"`,
		`"series"`,
	} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected insights json to contain %q, got:\n%s", want, stdout)
		}
	}

	mdOut := filepath.Join(t.TempDir(), "insights_report.md")
	stdout, stderr, exit = runKcal(t, binPath, dbPath,
		"analytics", "insights", "range",
		"--from", "2026-02-20",
		"--to", "2026-02-20",
		"--out", mdOut,
		"--out-format", "markdown",
		"--no-charts",
	)
	if exit != 0 {
		t.Fatalf("analytics insights markdown export failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "Saved insights report to") {
		t.Fatalf("expected saved-report message in stdout, got:\n%s", stdout)
	}
	mdRaw, err := os.ReadFile(mdOut)
	if err != nil {
		t.Fatalf("read markdown output: %v", err)
	}
	md := string(mdRaw)
	if !strings.Contains(md, "# Analytics Insights Report") || !strings.Contains(md, "## Rolling Windows") {
		t.Fatalf("expected markdown report sections, got:\n%s", md)
	}

	jsonOut := filepath.Join(t.TempDir(), "insights_report.json")
	stdout, stderr, exit = runKcal(t, binPath, dbPath,
		"analytics", "insights", "range",
		"--from", "2026-02-20",
		"--to", "2026-02-20",
		"--out", jsonOut,
		"--out-format", "json",
	)
	if exit != 0 {
		t.Fatalf("analytics insights json export failed: exit=%d stderr=%s", exit, stderr)
	}
	jsonRaw, err := os.ReadFile(jsonOut)
	if err != nil {
		t.Fatalf("read json output: %v", err)
	}
	if !strings.Contains(string(jsonRaw), `"rolling_windows"`) || !strings.Contains(string(jsonRaw), `"category_trends"`) {
		t.Fatalf("expected exported json report fields, got:\n%s", string(jsonRaw))
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

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "entry", "list", "--date", "2026-02-20", "--with-metadata")
	if exit != 0 {
		t.Fatalf("entry list failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "barcode") {
		t.Fatalf("expected barcode source in entry list, got: %s", stdout)
	}
	if !strings.Contains(stdout, "\t150\t") {
		t.Fatalf("expected scaled calories 150 in entry list, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"provider":"openfoodfacts"`) {
		t.Fatalf("expected barcode provider metadata, got: %s", stdout)
	}
	if !strings.Contains(stdout, `"servings":1.5`) {
		t.Fatalf("expected servings metadata, got: %s", stdout)
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

func TestEntryAddWithBarcodeFallbackSkipsMissingUSDAKey(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "lookup", "override", "set", "3017620422003",
		"--provider", "openfoodfacts",
		"--name", "Fallback Snack",
		"--brand", "Test",
		"--serving-amount", "20",
		"--serving-unit", "g",
		"--calories", "110",
		"--protein", "2",
		"--carbs", "12",
		"--fat", "6",
	)
	if exit != 0 {
		t.Fatalf("set override failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "entry", "add",
		"--barcode", "3017620422003",
		"--fallback-order", "usda,openfoodfacts",
		"--category", "snacks",
		"--date", "2026-02-20",
		"--time", "12:00",
	)
	if exit != 0 {
		t.Fatalf("entry add fallback failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "entry", "list", "--date", "2026-02-20")
	if exit != 0 {
		t.Fatalf("entry list failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "Fallback Snack") {
		t.Fatalf("expected entry from fallback provider in list output, got: %s", stdout)
	}
}

func TestEntryShowAndMetadataUpdate(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "entry", "add",
		"--name", "Metadata Meal",
		"--calories", "420",
		"--protein", "25",
		"--carbs", "35",
		"--fat", "18",
		"--category", "lunch",
		"--metadata-json", `{"imported":true}`,
		"--date", "2026-02-20",
		"--time", "12:15",
	)
	if exit != 0 {
		t.Fatalf("entry add failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "Added entry") {
		t.Fatalf("unexpected add output: %s", stdout)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "entry", "metadata", "1", "--metadata-json", `{"edited":"yes"}`)
	if exit != 0 {
		t.Fatalf("entry metadata update failed: exit=%d stderr=%s", exit, stderr)
	}
	showOut, stderr, exit := runKcal(t, binPath, dbPath, "entry", "show", "1")
	if exit != 0 {
		t.Fatalf("entry show failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(showOut, `Metadata: {"edited":"yes"}`) {
		t.Fatalf("expected updated metadata in show output, got: %s", showOut)
	}
}

func TestConfigSetAndBarcodeDefaults(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "config", "set",
		"--barcode-provider", "openfoodfacts",
		"--fallback-order", "openfoodfacts,upcitemdb",
		"--api-key-hint", "set KCAL_USDA_API_KEY",
	)
	if exit != 0 {
		t.Fatalf("config set failed: exit=%d stderr=%s", exit, stderr)
	}
	cfgOut, stderr, exit := runKcal(t, binPath, dbPath, "config", "get")
	if exit != 0 {
		t.Fatalf("config get failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(cfgOut, "barcode_provider\topenfoodfacts") {
		t.Fatalf("expected barcode provider config, got: %s", cfgOut)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "lookup", "override", "set", "3017620422003",
		"--provider", "openfoodfacts",
		"--name", "Config Snack",
		"--brand", "Test",
		"--serving-amount", "20",
		"--serving-unit", "g",
		"--calories", "120",
		"--protein", "2",
		"--carbs", "12",
		"--fat", "7",
	)
	if exit != 0 {
		t.Fatalf("override set failed: exit=%d stderr=%s", exit, stderr)
	}

	lookupOut, stderr, exit := runKcal(t, binPath, dbPath, "lookup", "barcode", "3017620422003")
	if exit != 0 {
		t.Fatalf("lookup barcode using config defaults failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(lookupOut, "Provider: openfoodfacts (override)") {
		t.Fatalf("expected default provider from config, got: %s", lookupOut)
	}
}

func TestExportImportJSONRoundTrip(t *testing.T) {
	binPath := buildKcalBinary(t)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kcal.db")
	importDB := filepath.Join(dir, "kcal-import.db")
	exportPath := filepath.Join(dir, "export.json")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "entry", "add",
		"--name", "Portable Meal",
		"--calories", "500",
		"--protein", "30",
		"--carbs", "50",
		"--fat", "20",
		"--category", "dinner",
		"--metadata-json", `{"portable":true}`,
		"--date", "2026-02-20",
		"--time", "19:00",
	)
	if exit != 0 {
		t.Fatalf("seed entry failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "export", "--format", "json", "--out", exportPath)
	if exit != 0 {
		t.Fatalf("export json failed: exit=%d stderr=%s", exit, stderr)
	}
	if _, err := os.Stat(exportPath); err != nil {
		t.Fatalf("expected export file to exist: %v", err)
	}

	initDB(t, binPath, importDB)
	_, stderr, exit = runKcal(t, binPath, importDB, "import", "--format", "json", "--in", exportPath)
	if exit != 0 {
		t.Fatalf("import json failed: exit=%d stderr=%s", exit, stderr)
	}
	listOut, stderr, exit := runKcal(t, binPath, importDB, "entry", "list", "--with-metadata")
	if exit != 0 {
		t.Fatalf("entry list after import failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(listOut, "Portable Meal") || !strings.Contains(listOut, `"portable":true`) {
		t.Fatalf("expected imported entry and metadata in list output, got: %s", listOut)
	}
}

func TestImportDryRunDoesNotWrite(t *testing.T) {
	binPath := buildKcalBinary(t)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kcal.db")
	importDB := filepath.Join(dir, "kcal-import.db")
	exportPath := filepath.Join(dir, "export.json")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "entry", "add",
		"--name", "DryRun Meal",
		"--calories", "250",
		"--protein", "20",
		"--carbs", "20",
		"--fat", "10",
		"--category", "lunch",
		"--date", "2026-02-20",
		"--time", "13:00",
	)
	if exit != 0 {
		t.Fatalf("seed entry failed: exit=%d stderr=%s", exit, stderr)
	}
	_, stderr, exit = runKcal(t, binPath, dbPath, "export", "--format", "json", "--out", exportPath)
	if exit != 0 {
		t.Fatalf("export json failed: exit=%d stderr=%s", exit, stderr)
	}

	initDB(t, binPath, importDB)
	_, stderr, exit = runKcal(t, binPath, importDB, "import", "--format", "json", "--in", exportPath, "--dry-run")
	if exit != 0 {
		t.Fatalf("import dry-run failed: exit=%d stderr=%s", exit, stderr)
	}
	listOut, stderr, exit := runKcal(t, binPath, importDB, "entry", "list")
	if exit != 0 {
		t.Fatalf("entry list failed: exit=%d stderr=%s", exit, stderr)
	}
	if strings.Contains(listOut, "DryRun Meal") {
		t.Fatalf("expected dry-run import to not write data, got: %s", listOut)
	}
}

func TestEntryRichNutrientsDisplay(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "entry", "add",
		"--name", "Micronutrient Meal",
		"--calories", "320",
		"--protein", "20",
		"--carbs", "35",
		"--fat", "10",
		"--fiber", "8",
		"--sugar", "6",
		"--sodium", "350",
		"--micros-json", `{"vitamin_c":{"value":40,"unit":"mg"}}`,
		"--category", "lunch",
		"--date", "2026-02-20",
		"--time", "12:30",
	)
	if exit != 0 {
		t.Fatalf("entry add failed: exit=%d stderr=%s", exit, stderr)
	}

	listOut, stderr, exit := runKcal(t, binPath, dbPath, "entry", "list", "--with-nutrients")
	if exit != 0 {
		t.Fatalf("entry list --with-nutrients failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(listOut, "FIBER_G") || !strings.Contains(listOut, "SODIUM_MG") || !strings.Contains(listOut, "vitamin_c") {
		t.Fatalf("expected richer nutrient columns and values in list output, got: %s", listOut)
	}

	showOut, stderr, exit := runKcal(t, binPath, dbPath, "entry", "show", "1")
	if exit != 0 {
		t.Fatalf("entry show failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(showOut, "Fiber: 8.0g") || !strings.Contains(showOut, "Sodium: 350.0mg") || !strings.Contains(showOut, "vitamin_c") {
		t.Fatalf("expected richer nutrient fields in show output, got: %s", showOut)
	}
}

func TestBackupCreateAndRestore(t *testing.T) {
	binPath := buildKcalBinary(t)
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "entry", "add",
		"--name", "Backup Meal",
		"--calories", "300",
		"--protein", "25",
		"--carbs", "20",
		"--fat", "12",
		"--category", "dinner",
		"--date", "2026-02-20",
		"--time", "18:30",
	)
	if exit != 0 {
		t.Fatalf("seed entry failed: exit=%d stderr=%s", exit, stderr)
	}

	backupFile := filepath.Join(dir, "snapshots", "bk.db")
	_, stderr, exit = runKcal(t, binPath, dbPath, "backup", "create", "--out", backupFile)
	if exit != 0 {
		t.Fatalf("backup create failed: exit=%d stderr=%s", exit, stderr)
	}
	if _, err := os.Stat(backupFile); err != nil {
		t.Fatalf("expected backup file: %v", err)
	}
	if _, err := os.Stat(backupFile + ".sha256"); err != nil {
		t.Fatalf("expected checksum file: %v", err)
	}

	restoredDB := filepath.Join(dir, "restored.db")
	_, stderr, exit = runKcal(t, binPath, restoredDB, "backup", "restore", "--file", backupFile)
	if exit != 0 {
		t.Fatalf("backup restore failed: exit=%d stderr=%s", exit, stderr)
	}
	listOut, stderr, exit := runKcal(t, binPath, restoredDB, "entry", "list")
	if exit != 0 {
		t.Fatalf("entry list on restored db failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(listOut, "Backup Meal") {
		t.Fatalf("expected restored entry in db, got: %s", listOut)
	}
}
