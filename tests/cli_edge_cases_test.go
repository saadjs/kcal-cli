package tests

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func buildKcalBinary(t *testing.T) string {
	t.Helper()
	repoRoot, err := filepath.Abs("..")
	if err != nil {
		t.Fatalf("resolve repo root: %v", err)
	}
	binPath := filepath.Join(t.TempDir(), "kcal")
	cmd := exec.Command("go", "build", "-o", binPath, ".")
	cmd.Dir = repoRoot
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build kcal binary: %v\n%s", err, string(out))
	}
	return binPath
}

func runKcal(t *testing.T, binPath, dbPath string, args ...string) (string, string, int) {
	t.Helper()
	allArgs := append([]string{"--db", dbPath}, args...)
	cmd := exec.Command(binPath, allArgs...)
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), stderr.String(), 0
	}
	exitErr, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("run kcal command: %v", err)
	}
	return stdout.String(), stderr.String(), exitErr.ExitCode()
}

func initDB(t *testing.T, binPath, dbPath string) {
	t.Helper()
	_, stderr, exit := runKcal(t, binPath, dbPath, "init")
	if exit != 0 {
		t.Fatalf("init db failed: exit=%d stderr=%s", exit, stderr)
	}
}

func TestCLIRejectsNegativeCalories(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"entry", "add",
		"--name", "x",
		"--calories", "-1",
		"--protein", "1",
		"--carbs", "1",
		"--fat", "1",
		"--category", "breakfast",
	)

	if exit == 0 {
		t.Fatalf("expected non-zero exit for negative calories")
	}
	if !strings.Contains(stderr, "calories must be >= 0") {
		t.Fatalf("expected validation error in stderr, got: %s", stderr)
	}
}

func TestCLIRejectsTimeWithoutDate(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"entry", "add",
		"--name", "x",
		"--calories", "10",
		"--protein", "1",
		"--carbs", "1",
		"--fat", "1",
		"--category", "breakfast",
		"--time", "09:00",
	)

	if exit == 0 {
		t.Fatalf("expected non-zero exit when --time is passed without --date")
	}
	if !strings.Contains(stderr, "--date is required when --time is set") {
		t.Fatalf("expected date/time validation error in stderr, got: %s", stderr)
	}
}

func TestCLICategoryDeleteRequiresReassignWhenEntriesExist(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath, "category", "add", "supper")
	if exit != 0 {
		t.Fatalf("add category failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath,
		"entry", "add",
		"--name", "x",
		"--calories", "10",
		"--protein", "1",
		"--carbs", "1",
		"--fat", "1",
		"--category", "supper",
	)
	if exit != 0 {
		t.Fatalf("seed entry failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "category", "delete", "supper")
	if exit == 0 {
		t.Fatalf("expected non-zero exit when deleting category with entries and no reassign")
	}
	if !strings.Contains(stderr, "use --reassign") {
		t.Fatalf("expected reassign hint in stderr, got: %s", stderr)
	}
}

func TestCLIAnalyticsRangeRejectsInvalidDate(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"analytics", "range",
		"--from", "2026-02-30",
		"--to", "2026-02-20",
	)

	if exit == 0 {
		t.Fatalf("expected non-zero exit for invalid range date")
	}
	if !strings.Contains(stderr, "invalid --from date") {
		t.Fatalf("expected invalid from date error in stderr, got: %s", stderr)
	}
}

func TestCLIRecipeShowRejectsPartialNumericIdentifier(t *testing.T) {
	binPath := buildKcalBinary(t)
	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	initDB(t, binPath, dbPath)

	_, stderr, exit := runKcal(t, binPath, dbPath,
		"recipe", "add",
		"--name", "oats",
		"--calories", "100",
		"--protein", "10",
		"--carbs", "10",
		"--fat", "5",
		"--servings", "1",
	)
	if exit != 0 {
		t.Fatalf("add recipe failed: exit=%d stderr=%s", exit, stderr)
	}

	_, stderr, exit = runKcal(t, binPath, dbPath, "recipe", "show", "1abc")
	if exit == 0 {
		t.Fatalf("expected non-zero exit for partial numeric identifier")
	}
	if !strings.Contains(stderr, `recipe "1abc" not found`) {
		t.Fatalf("expected strict identifier not-found message, got: %s", stderr)
	}
}

func TestMain(m *testing.M) {
	os.Exit(m.Run())
}
