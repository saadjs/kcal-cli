package kcal

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/app"
	"github.com/saadjs/kcal-cli/internal/db"
)

func withDB(run func(*sql.DB) error) error {
	path, err := resolveDBPath()
	if err != nil {
		return err
	}
	if err := app.EnsureDBDir(path); err != nil {
		return err
	}
	sqldb, err := db.Open(path)
	if err != nil {
		return err
	}
	defer sqldb.Close()

	if err := db.ApplyMigrations(sqldb); err != nil {
		return err
	}
	return run(sqldb)
}

func parseInt64Arg(name, value string) (int64, error) {
	v, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s %q", name, value)
	}
	if v <= 0 {
		return 0, fmt.Errorf("%s must be > 0", name)
	}
	return v, nil
}

func parseDateTimeOrNow(date, timeStr string) (time.Time, error) {
	date = strings.TrimSpace(date)
	timeStr = strings.TrimSpace(timeStr)
	if date == "" && timeStr == "" {
		return time.Now(), nil
	}
	if date == "" {
		return time.Time{}, fmt.Errorf("--date is required when --time is set")
	}
	if timeStr == "" {
		t, err := time.ParseInLocation("2006-01-02", date, time.Local)
		if err != nil {
			return time.Time{}, fmt.Errorf("invalid --date %q (expected YYYY-MM-DD)", date)
		}
		return t, nil
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --date/--time (expected YYYY-MM-DD and HH:MM)")
	}
	return t, nil
}

func parseDateTime(date, timeStr string) (time.Time, error) {
	date = strings.TrimSpace(date)
	timeStr = strings.TrimSpace(timeStr)
	if date == "" || timeStr == "" {
		return time.Time{}, fmt.Errorf("both --date and --time are required")
	}
	t, err := time.ParseInLocation("2006-01-02 15:04", date+" "+timeStr, time.Local)
	if err != nil {
		return time.Time{}, fmt.Errorf("invalid --date/--time (expected YYYY-MM-DD and HH:MM)")
	}
	return t, nil
}
