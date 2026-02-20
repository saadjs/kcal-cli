package db

import (
	"database/sql"
	"fmt"
)

type migration struct {
	version int
	name    string
	sql     string
}

var migrations = []migration{
	{
		version: 1,
		name:    "initial_schema",
		sql: `
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS categories (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  is_default INTEGER NOT NULL DEFAULT 0,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  archived_at DATETIME
);

CREATE TABLE IF NOT EXISTS goals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  calories INTEGER NOT NULL CHECK(calories >= 0),
  protein_g REAL NOT NULL CHECK(protein_g >= 0),
  carbs_g REAL NOT NULL CHECK(carbs_g >= 0),
  fat_g REAL NOT NULL CHECK(fat_g >= 0),
  effective_date TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(effective_date)
);

CREATE TABLE IF NOT EXISTS recipes (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL UNIQUE,
  calories_total INTEGER NOT NULL CHECK(calories_total >= 0),
  protein_total_g REAL NOT NULL CHECK(protein_total_g >= 0),
  carbs_total_g REAL NOT NULL CHECK(carbs_total_g >= 0),
  fat_total_g REAL NOT NULL CHECK(fat_total_g >= 0),
  servings REAL NOT NULL CHECK(servings > 0),
  notes TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE TABLE IF NOT EXISTS entries (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  name TEXT NOT NULL,
  calories INTEGER NOT NULL CHECK(calories >= 0),
  protein_g REAL NOT NULL CHECK(protein_g >= 0),
  carbs_g REAL NOT NULL CHECK(carbs_g >= 0),
  fat_g REAL NOT NULL CHECK(fat_g >= 0),
  category_id INTEGER NOT NULL,
  consumed_at DATETIME NOT NULL,
  notes TEXT,
  source_type TEXT NOT NULL DEFAULT 'manual',
  source_id INTEGER,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(category_id) REFERENCES categories(id)
);

CREATE INDEX IF NOT EXISTS idx_entries_consumed_at ON entries(consumed_at);
CREATE INDEX IF NOT EXISTS idx_entries_category_id ON entries(category_id);
`,
	},
	{
		version: 2,
		name:    "body_tracking",
		sql: `
CREATE TABLE IF NOT EXISTS body_measurements (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  measured_at DATETIME NOT NULL,
  weight_kg REAL NOT NULL CHECK(weight_kg > 0),
  body_fat_pct REAL CHECK(body_fat_pct >= 0 AND body_fat_pct <= 100),
  notes TEXT,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_body_measurements_measured_at ON body_measurements(measured_at);

CREATE TABLE IF NOT EXISTS body_goals (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  target_weight_kg REAL NOT NULL CHECK(target_weight_kg > 0),
  target_body_fat_pct REAL CHECK(target_body_fat_pct >= 0 AND target_body_fat_pct <= 100),
  target_date TEXT,
  effective_date TEXT NOT NULL,
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE(effective_date)
);
`,
	},
	{
		version: 3,
		name:    "recipe_ingredients",
		sql: `
CREATE TABLE IF NOT EXISTS recipe_ingredients (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  recipe_id INTEGER NOT NULL,
  name TEXT NOT NULL,
  amount REAL NOT NULL CHECK(amount > 0),
  amount_unit TEXT NOT NULL,
  calories INTEGER NOT NULL CHECK(calories >= 0),
  protein_g REAL NOT NULL CHECK(protein_g >= 0),
  carbs_g REAL NOT NULL CHECK(carbs_g >= 0),
  fat_g REAL NOT NULL CHECK(fat_g >= 0),
  created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
  FOREIGN KEY(recipe_id) REFERENCES recipes(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_recipe_ingredients_recipe_id ON recipe_ingredients(recipe_id);
`,
	},
	{
		version: 4,
		name:    "barcode_cache",
		sql: `
CREATE TABLE IF NOT EXISTS barcode_cache (
  provider TEXT NOT NULL,
  barcode TEXT NOT NULL,
  description TEXT NOT NULL,
  brand TEXT NOT NULL DEFAULT '',
  serving_amount REAL NOT NULL DEFAULT 0,
  serving_unit TEXT NOT NULL DEFAULT '',
  calories REAL NOT NULL DEFAULT 0,
  protein_g REAL NOT NULL DEFAULT 0,
  carbs_g REAL NOT NULL DEFAULT 0,
  fat_g REAL NOT NULL DEFAULT 0,
  source_id INTEGER NOT NULL DEFAULT 0,
  raw_json TEXT,
  fetched_at DATETIME NOT NULL,
  expires_at DATETIME NOT NULL,
  PRIMARY KEY(provider, barcode)
);
`,
	},
	{
		version: 5,
		name:    "barcode_overrides",
		sql: `
CREATE TABLE IF NOT EXISTS barcode_overrides (
  provider TEXT NOT NULL,
  barcode TEXT NOT NULL,
  description TEXT NOT NULL,
  brand TEXT NOT NULL DEFAULT '',
  serving_amount REAL NOT NULL CHECK(serving_amount > 0),
  serving_unit TEXT NOT NULL,
  calories REAL NOT NULL DEFAULT 0 CHECK(calories >= 0),
  protein_g REAL NOT NULL DEFAULT 0 CHECK(protein_g >= 0),
  carbs_g REAL NOT NULL DEFAULT 0 CHECK(carbs_g >= 0),
  fat_g REAL NOT NULL DEFAULT 0 CHECK(fat_g >= 0),
  source_id INTEGER NOT NULL DEFAULT 0,
  notes TEXT NOT NULL DEFAULT '',
  updated_at DATETIME NOT NULL,
  PRIMARY KEY(provider, barcode)
);
`,
	},
	{
		version: 6,
		name:    "entry_metadata",
		sql: `
ALTER TABLE entries ADD COLUMN metadata_json TEXT NOT NULL DEFAULT '';
`,
	},
}

var defaultCategories = []string{"breakfast", "lunch", "dinner", "snacks"}

func ApplyMigrations(db *sql.DB) error {
	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
  version INTEGER PRIMARY KEY,
  name TEXT NOT NULL,
  applied_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);
`); err != nil {
		return fmt.Errorf("ensure schema_migrations table: %w", err)
	}

	for _, m := range migrations {
		var exists int
		err := db.QueryRow(`SELECT 1 FROM schema_migrations WHERE version = ?`, m.version).Scan(&exists)
		if err == nil {
			continue
		}
		if err != sql.ErrNoRows {
			return fmt.Errorf("check migration version %d: %w", m.version, err)
		}

		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin migration tx: %w", err)
		}

		if _, err := tx.Exec(m.sql); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("apply migration version %d (%s): %w", m.version, m.name, err)
		}
		if _, err := tx.Exec(`INSERT INTO schema_migrations(version, name) VALUES(?, ?)`, m.version, m.name); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("record migration version %d: %w", m.version, err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration version %d: %w", m.version, err)
		}
	}

	for _, name := range defaultCategories {
		if _, err := db.Exec(`INSERT OR IGNORE INTO categories(name, is_default) VALUES(?, 1)`, name); err != nil {
			return fmt.Errorf("seed default category %s: %w", name, err)
		}
	}

	return nil
}
