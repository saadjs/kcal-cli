package app

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	appDirName = "kcal"
	dbFileName = "kcal.db"
)

func DefaultDBPath() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, appDirName, dbFileName), nil
}

func EnsureDBDir(path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create db directory: %w", err)
	}
	return nil
}
