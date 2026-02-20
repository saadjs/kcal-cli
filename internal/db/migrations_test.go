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
	if migrationCount != 11 {
		t.Fatalf("expected 11 migration versions, got %d", migrationCount)
	}

	var metadataColCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('entries') WHERE name = 'metadata_json'`).Scan(&metadataColCount); err != nil {
		t.Fatalf("check entries metadata column: %v", err)
	}
	if metadataColCount != 1 {
		t.Fatalf("expected metadata_json column in entries table")
	}

	var configTableCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'app_config'`).Scan(&configTableCount); err != nil {
		t.Fatalf("check app_config table: %v", err)
	}
	if configTableCount != 1 {
		t.Fatalf("expected app_config table to exist")
	}

	var exerciseTableCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'exercise_logs'`).Scan(&exerciseTableCount); err != nil {
		t.Fatalf("check exercise_logs table: %v", err)
	}
	if exerciseTableCount != 1 {
		t.Fatalf("expected exercise_logs table to exist")
	}

	var exerciseMetadataColCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('exercise_logs') WHERE name = 'metadata_json'`).Scan(&exerciseMetadataColCount); err != nil {
		t.Fatalf("check exercise_logs metadata column: %v", err)
	}
	if exerciseMetadataColCount != 1 {
		t.Fatalf("expected metadata_json column in exercise_logs table")
	}

	var entriesFiberColCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('entries') WHERE name = 'fiber_g'`).Scan(&entriesFiberColCount); err != nil {
		t.Fatalf("check entries fiber_g column: %v", err)
	}
	if entriesFiberColCount != 1 {
		t.Fatalf("expected fiber_g column in entries table")
	}

	var entriesMicrosColCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('entries') WHERE name = 'micronutrients_json'`).Scan(&entriesMicrosColCount); err != nil {
		t.Fatalf("check entries micronutrients_json column: %v", err)
	}
	if entriesMicrosColCount != 1 {
		t.Fatalf("expected micronutrients_json column in entries table")
	}

	var cacheMicrosColCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('barcode_cache') WHERE name = 'micronutrients_json'`).Scan(&cacheMicrosColCount); err != nil {
		t.Fatalf("check barcode_cache micronutrients_json column: %v", err)
	}
	if cacheMicrosColCount != 1 {
		t.Fatalf("expected micronutrients_json column in barcode_cache table")
	}

	var overrideMicrosColCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM pragma_table_info('barcode_overrides') WHERE name = 'micronutrients_json'`).Scan(&overrideMicrosColCount); err != nil {
		t.Fatalf("check barcode_overrides micronutrients_json column: %v", err)
	}
	if overrideMicrosColCount != 1 {
		t.Fatalf("expected micronutrients_json column in barcode_overrides table")
	}

	var searchCacheTableCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'provider_search_cache'`).Scan(&searchCacheTableCount); err != nil {
		t.Fatalf("check provider_search_cache table: %v", err)
	}
	if searchCacheTableCount != 1 {
		t.Fatalf("expected provider_search_cache table to exist")
	}

	var searchCacheIndexCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND name = 'idx_provider_search_cache_expires_at'`).Scan(&searchCacheIndexCount); err != nil {
		t.Fatalf("check provider_search_cache expires index: %v", err)
	}
	if searchCacheIndexCount != 1 {
		t.Fatalf("expected idx_provider_search_cache_expires_at index to exist")
	}

	var searchCacheQueryIndexCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'index' AND name = 'idx_provider_search_cache_query_norm'`).Scan(&searchCacheQueryIndexCount); err != nil {
		t.Fatalf("check provider_search_cache query_norm index: %v", err)
	}
	if searchCacheQueryIndexCount != 1 {
		t.Fatalf("expected idx_provider_search_cache_query_norm index to exist")
	}

	var savedFoodsTableCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'saved_foods'`).Scan(&savedFoodsTableCount); err != nil {
		t.Fatalf("check saved_foods table: %v", err)
	}
	if savedFoodsTableCount != 1 {
		t.Fatalf("expected saved_foods table to exist")
	}

	var savedMealsTableCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'saved_meals'`).Scan(&savedMealsTableCount); err != nil {
		t.Fatalf("check saved_meals table: %v", err)
	}
	if savedMealsTableCount != 1 {
		t.Fatalf("expected saved_meals table to exist")
	}

	var savedMealComponentsTableCount int
	if err := sqldb.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'saved_meal_components'`).Scan(&savedMealComponentsTableCount); err != nil {
		t.Fatalf("check saved_meal_components table: %v", err)
	}
	if savedMealComponentsTableCount != 1 {
		t.Fatalf("expected saved_meal_components table to exist")
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
