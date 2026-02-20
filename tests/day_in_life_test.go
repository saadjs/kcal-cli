package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestDayInTheLifeFlow(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")

	_, stderr, exit := runKcal(t, binPath, dbPath, "init")
	if exit != 0 {
		t.Fatalf("init failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
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

	_, stderr, exit = runKcal(t, binPath, dbPath, "category", "add", "supper")
	if exit != 0 {
		t.Fatalf("category add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"entry", "add",
		"--name", "Chicken bowl",
		"--calories", "550",
		"--protein", "45",
		"--carbs", "40",
		"--fat", "18",
		"--category", "supper",
		"--date", "2026-02-20",
		"--time", "19:30",
	)
	if exit != 0 {
		t.Fatalf("entry add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"recipe", "add",
		"--name", "Overnight oats",
		"--calories", "400",
		"--protein", "20",
		"--carbs", "50",
		"--fat", "10",
		"--servings", "2",
	)
	if exit != 0 {
		t.Fatalf("recipe add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"recipe", "log", "Overnight oats",
		"--servings", "1",
		"--category", "breakfast",
		"--date", "2026-02-20",
		"--time", "08:00",
	)
	if exit != 0 {
		t.Fatalf("recipe log failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath,
		"analytics", "range",
		"--from", "2026-02-20",
		"--to", "2026-02-20",
	)
	if exit != 0 {
		t.Fatalf("analytics range failed: exit=%d stderr=%s", exit, stderr)
	}

	checks := []string{
		"Range: 2026-02-20 to 2026-02-20",
		"Totals: kcal=750 P=55.0 C=65.0 F=23.0",
		"supper",
		"breakfast",
	}
	for _, want := range checks {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected analytics output to contain %q, got:\n%s", want, stdout)
		}
	}
}
