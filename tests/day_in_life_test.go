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
	_, stderr, exit = runKcal(t, binPath, dbPath,
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
		"--time", "07:00",
	)
	if exit != 0 {
		t.Fatalf("body add failed: exit=%d stderr=%s", exit, stderr)
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
		"recipe", "ingredient", "add", "Overnight oats",
		"--name", "Oats",
		"--amount", "40",
		"--unit", "g",
		"--calories", "150",
		"--protein", "5",
		"--carbs", "27",
		"--fat", "3",
	)
	if exit != 0 {
		t.Fatalf("recipe ingredient add failed: exit=%d stderr=%s", exit, stderr)
	}
	_, stderr, exit = runKcal(t, binPath, dbPath,
		"recipe", "ingredient", "add", "Overnight oats",
		"--name", "Milk",
		"--amount", "200",
		"--unit", "ml",
		"--calories", "80",
		"--protein", "5",
		"--carbs", "10",
		"--fat", "2",
	)
	if exit != 0 {
		t.Fatalf("recipe ingredient add #2 failed: exit=%d stderr=%s", exit, stderr)
	}
	_, stderr, exit = runKcal(t, binPath, dbPath, "recipe", "recalc", "Overnight oats")
	if exit != 0 {
		t.Fatalf("recipe recalc failed: exit=%d stderr=%s", exit, stderr)
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

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"exercise", "add",
		"--type", "running",
		"--calories", "300",
		"--duration-min", "35",
		"--date", "2026-02-20",
		"--time", "18:30",
	)
	if exit != 0 {
		t.Fatalf("exercise add failed: exit=%d stderr=%s", exit, stderr)
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
		"Totals: intake=665 exercise=300 net=365",
		"supper",
		"breakfast",
		"Body",
	}
	for _, want := range checks {
		if !strings.Contains(stdout, want) {
			t.Fatalf("expected analytics output to contain %q, got:\n%s", want, stdout)
		}
	}
}
