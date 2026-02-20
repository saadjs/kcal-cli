package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/model"
)

type BodyMeasurementInput struct {
	Weight     float64
	Unit       string
	BodyFatPct *float64
	MeasuredAt time.Time
	Notes      string
}

type BodyMeasurementFilter struct {
	Date     string
	FromDate string
	ToDate   string
	Limit    int
}

type UpdateBodyMeasurementInput struct {
	ID int64
	BodyMeasurementInput
}

func AddBodyMeasurement(db *sql.DB, in BodyMeasurementInput) (int64, error) {
	weightKg, err := convertWeightToKg(in.Weight, in.Unit)
	if err != nil {
		return 0, err
	}
	if in.BodyFatPct != nil {
		if *in.BodyFatPct < 0 || *in.BodyFatPct > 100 {
			return 0, fmt.Errorf("body-fat must be between 0 and 100")
		}
	}
	if in.MeasuredAt.IsZero() {
		in.MeasuredAt = time.Now()
	}
	res, err := db.Exec(`
INSERT INTO body_measurements(measured_at, weight_kg, body_fat_pct, notes)
VALUES(?, ?, ?, ?)
`, in.MeasuredAt.Format(time.RFC3339), weightKg, in.BodyFatPct, strings.TrimSpace(in.Notes))
	if err != nil {
		return 0, fmt.Errorf("add body measurement: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve body measurement id: %w", err)
	}
	return id, nil
}

func ListBodyMeasurements(db *sql.DB, f BodyMeasurementFilter) ([]model.BodyMeasurement, error) {
	if strings.TrimSpace(f.Date) != "" && (strings.TrimSpace(f.FromDate) != "" || strings.TrimSpace(f.ToDate) != "") {
		return nil, fmt.Errorf("--date cannot be combined with --from or --to")
	}
	query := `SELECT id, measured_at, weight_kg, body_fat_pct, IFNULL(notes, '') FROM body_measurements WHERE 1=1`
	args := make([]any, 0)

	if strings.TrimSpace(f.Date) != "" {
		start, end, err := dayBounds(f.Date)
		if err != nil {
			return nil, err
		}
		query += ` AND measured_at >= ? AND measured_at < ?`
		args = append(args, start, end)
	}
	if strings.TrimSpace(f.FromDate) != "" {
		from, err := parseDateStart(f.FromDate)
		if err != nil {
			return nil, err
		}
		query += ` AND measured_at >= ?`
		args = append(args, from)
	}
	if strings.TrimSpace(f.ToDate) != "" {
		to, err := parseDateEndExclusive(f.ToDate)
		if err != nil {
			return nil, err
		}
		query += ` AND measured_at < ?`
		args = append(args, to)
	}

	query += ` ORDER BY measured_at DESC`
	if f.Limit <= 0 {
		f.Limit = 50
	}
	query += ` LIMIT ?`
	args = append(args, f.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list body measurements: %w", err)
	}
	defer rows.Close()

	items := make([]model.BodyMeasurement, 0)
	for rows.Next() {
		var m model.BodyMeasurement
		var measuredAtRaw string
		var bodyFat sql.NullFloat64
		if err := rows.Scan(&m.ID, &measuredAtRaw, &m.WeightKg, &bodyFat, &m.Notes); err != nil {
			return nil, fmt.Errorf("scan body measurement: %w", err)
		}
		measured, err := time.Parse(time.RFC3339, measuredAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse measured_at: %w", err)
		}
		m.MeasuredAt = measured
		if bodyFat.Valid {
			v := bodyFat.Float64
			m.BodyFatPct = &v
		}
		items = append(items, m)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate body measurements: %w", err)
	}
	return items, nil
}

func UpdateBodyMeasurement(db *sql.DB, in UpdateBodyMeasurementInput) error {
	if in.ID <= 0 {
		return fmt.Errorf("measurement id must be > 0")
	}
	weightKg, err := convertWeightToKg(in.Weight, in.Unit)
	if err != nil {
		return err
	}
	if in.BodyFatPct != nil {
		if *in.BodyFatPct < 0 || *in.BodyFatPct > 100 {
			return fmt.Errorf("body-fat must be between 0 and 100")
		}
	}
	if in.MeasuredAt.IsZero() {
		return fmt.Errorf("measurement date/time is required")
	}
	res, err := db.Exec(`
UPDATE body_measurements
SET measured_at = ?, weight_kg = ?, body_fat_pct = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, in.MeasuredAt.Format(time.RFC3339), weightKg, in.BodyFatPct, strings.TrimSpace(in.Notes), in.ID)
	if err != nil {
		return fmt.Errorf("update body measurement %d: %w", in.ID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("body measurement %d not found", in.ID)
	}
	return nil
}

func DeleteBodyMeasurement(db *sql.DB, id int64) error {
	if id <= 0 {
		return fmt.Errorf("measurement id must be > 0")
	}
	res, err := db.Exec(`DELETE FROM body_measurements WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete body measurement %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("body measurement %d not found", id)
	}
	return nil
}

func convertWeightToKg(value float64, unit string) (float64, error) {
	if value <= 0 {
		return 0, fmt.Errorf("weight must be > 0")
	}
	u := strings.ToLower(strings.TrimSpace(unit))
	if u == "" {
		u = "kg"
	}
	switch u {
	case "kg":
		return value, nil
	case "lb", "lbs":
		return value * 0.45359237, nil
	default:
		return 0, fmt.Errorf("invalid weight unit %q (use kg or lb)", unit)
	}
}

func weightFromKg(weightKg float64, unit string) (float64, error) {
	u := strings.ToLower(strings.TrimSpace(unit))
	if u == "" {
		u = "kg"
	}
	switch u {
	case "kg":
		return weightKg, nil
	case "lb", "lbs":
		return weightKg / 0.45359237, nil
	default:
		return 0, fmt.Errorf("invalid weight unit %q (use kg or lb)", unit)
	}
}

func WeightFromKg(weightKg float64, unit string) (float64, error) {
	return weightFromKg(weightKg, unit)
}

func ToKg(weight float64, unit string) (float64, error) {
	return convertWeightToKg(weight, unit)
}
