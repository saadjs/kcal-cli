package tests

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCLISavedFoodAndSavedMealFlow(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"saved-food", "add",
		"--name", "Greek Yogurt",
		"--category", "breakfast",
		"--calories", "150",
		"--protein", "15",
		"--carbs", "10",
		"--fat", "5",
	)
	if exit != 0 {
		t.Fatalf("saved-food add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"saved-food", "log", "Greek Yogurt",
		"--servings", "2",
		"--date", "2026-02-20",
		"--time", "08:00",
	)
	if exit != 0 {
		t.Fatalf("saved-food log failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit := runKcal(t, binPath, dbPath, "entry", "list", "--date", "2026-02-20")
	if exit != 0 {
		t.Fatalf("entry list failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "saved_food") {
		t.Fatalf("expected logged saved_food source in entry list, got:\n%s", stdout)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"saved-meal", "add",
		"--name", "Yogurt Bowl",
		"--category", "breakfast",
	)
	if exit != 0 {
		t.Fatalf("saved-meal add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"saved-meal", "component", "add", "Yogurt Bowl",
		"--saved-food", "Greek Yogurt",
	)
	if exit != 0 {
		t.Fatalf("saved-meal component add failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"saved-meal", "log", "Yogurt Bowl",
		"--servings", "1",
		"--date", "2026-02-20",
		"--time", "09:00",
	)
	if exit != 0 {
		t.Fatalf("saved-meal log failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit = runKcal(t, binPath, dbPath, "entry", "list", "--date", "2026-02-20")
	if exit != 0 {
		t.Fatalf("entry list #2 failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "saved_meal") {
		t.Fatalf("expected logged saved_meal source in entry list, got:\n%s", stdout)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "saved-food", "archive", "Greek Yogurt")
	if exit != 0 {
		t.Fatalf("saved-food archive failed: exit=%d stderr=%s", exit, stderr)
	}

	stdout, stderr, exit = runKcal(t, binPath, dbPath, "saved-food", "list")
	if exit != 0 {
		t.Fatalf("saved-food list failed: exit=%d stderr=%s", exit, stderr)
	}
	if strings.Contains(stdout, "Greek Yogurt") {
		t.Fatalf("expected archived saved food hidden in default list, got:\n%s", stdout)
	}

	stdout, stderr, exit = runKcal(t, binPath, dbPath, "saved-food", "list", "--include-archived")
	if exit != 0 {
		t.Fatalf("saved-food list --include-archived failed: exit=%d stderr=%s", exit, stderr)
	}
	if !strings.Contains(stdout, "Greek Yogurt") {
		t.Fatalf("expected archived saved food in include-archived list, got:\n%s", stdout)
	}
}
