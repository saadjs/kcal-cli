package service_test

import (
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/saad/kcal-cli/internal/db"
)

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kcal.db")
	sqldb, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.ApplyMigrations(sqldb); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return sqldb
}
