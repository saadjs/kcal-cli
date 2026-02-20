package service

import (
	"database/sql"
	"fmt"
	"strings"
)

func validateNonNegativeInt(name string, value int) error {
	if value < 0 {
		return fmt.Errorf("%s must be >= 0", name)
	}
	return nil
}

func validateNonNegativeFloat(name string, value float64) error {
	if value < 0 {
		return fmt.Errorf("%s must be >= 0", name)
	}
	return nil
}

func normalizeName(name string) string {
	return strings.TrimSpace(strings.ToLower(name))
}

func categoryIDByName(db *sql.DB, category string) (int64, error) {
	name := normalizeName(category)
	if name == "" {
		return 0, fmt.Errorf("category name is required")
	}
	var id int64
	if err := db.QueryRow(`SELECT id FROM categories WHERE name = ?`, name).Scan(&id); err != nil {
		if err == sql.ErrNoRows {
			return 0, fmt.Errorf("category %q does not exist", name)
		}
		return 0, fmt.Errorf("lookup category %q: %w", name, err)
	}
	return id, nil
}
