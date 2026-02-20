package service

import (
	"database/sql"
	"fmt"
	"strings"
)

const (
	ConfigBarcodeProvider      = "barcode_provider"
	ConfigBarcodeFallbackOrder = "barcode_fallback_order"
	ConfigAPIKeyHint           = "barcode_api_key_hint"
)

func SetConfig(db *sql.DB, key, value string) error {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return fmt.Errorf("config key is required")
	}
	_, err := db.Exec(`
INSERT INTO app_config(key, value, updated_at)
VALUES(?, ?, CURRENT_TIMESTAMP)
ON CONFLICT(key) DO UPDATE SET value=excluded.value, updated_at=excluded.updated_at
`, key, strings.TrimSpace(value))
	if err != nil {
		return fmt.Errorf("set config %q: %w", key, err)
	}
	return nil
}

func GetConfig(db *sql.DB, key string) (string, bool, error) {
	key = strings.TrimSpace(strings.ToLower(key))
	if key == "" {
		return "", false, fmt.Errorf("config key is required")
	}
	var value string
	err := db.QueryRow(`SELECT value FROM app_config WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, fmt.Errorf("get config %q: %w", key, err)
	}
	return value, true, nil
}

func ListConfig(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query(`SELECT key, value FROM app_config ORDER BY key ASC`)
	if err != nil {
		return nil, fmt.Errorf("list config: %w", err)
	}
	defer rows.Close()
	out := map[string]string{}
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan config: %w", err)
		}
		out[key] = value
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate config: %w", err)
	}
	return out, nil
}
