package db_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/saad/kcal-cli/internal/db"
)

func TestApplyMigrationsIdempotentAndSeedsDefaults(t *testing.T) {
	t.Parallel()

	dbPath := filepath.Join(t.TempDir(), "kcal.db")
	sqldb, err := db.Open(dbPath)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer sqldb.Close()

	if err := db.ApplyMigrations(sqldb); err != nil {
		t.Fatalf("first apply migrations: %v", err)
	}
	if err := db.ApplyMigrations(sqldb); err != nil {
		t.Fatalf("second apply migrations: %v", err)
	}

	var migrationCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM schema_migrations`).Scan(&migrationCount); err != nil {
		t.Fatalf("count migrations: %v", err)
	}
	if migrationCount != 1 {
		t.Fatalf("expected 1 migration version, got %d", migrationCount)
	}

	var categoryCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM categories`).Scan(&categoryCount); err != nil {
		t.Fatalf("count categories: %v", err)
	}
	if categoryCount < 4 {
		t.Fatalf("expected at least 4 seeded categories, got %d", categoryCount)
	}

	if _, err := os.Stat(dbPath); err != nil {
		t.Fatalf("expected db file to exist: %v", err)
	}
}
