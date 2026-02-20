package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/model"
)

type ExerciseLogInput struct {
	ExerciseType   string
	CaloriesBurned int
	DurationMin    *int
	Distance       *float64
	DistanceUnit   string
	PerformedAt    time.Time
	Notes          string
	Metadata       string
}

type ListExerciseFilter struct {
	Date         string
	FromDate     string
	ToDate       string
	ExerciseType string
	Limit        int
}

type UpdateExerciseInput struct {
	ID int64
	ExerciseLogInput
}

func CreateExerciseLog(db *sql.DB, in ExerciseLogInput) (int64, error) {
	normalized, err := normalizeExerciseInput(in, false)
	if err != nil {
		return 0, err
	}
	res, err := db.Exec(`
INSERT INTO exercise_logs(exercise_type, calories_burned, duration_min, distance, distance_unit, performed_at, notes, metadata_json)
VALUES(?, ?, ?, ?, ?, ?, ?, ?)
`, normalized.ExerciseType, normalized.CaloriesBurned, normalized.DurationMin, normalized.Distance, nullableString(normalized.DistanceUnit), normalized.PerformedAt.Format(time.RFC3339), normalized.Notes, normalized.Metadata)
	if err != nil {
		return 0, fmt.Errorf("add exercise log: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve exercise log id: %w", err)
	}
	return id, nil
}

func ListExerciseLogs(db *sql.DB, f ListExerciseFilter) ([]model.ExerciseLog, error) {
	if strings.TrimSpace(f.Date) != "" && (strings.TrimSpace(f.FromDate) != "" || strings.TrimSpace(f.ToDate) != "") {
		return nil, fmt.Errorf("--date cannot be combined with --from or --to")
	}

	query := `SELECT id, exercise_type, calories_burned, duration_min, distance, IFNULL(distance_unit, ''), performed_at, IFNULL(notes, ''), IFNULL(metadata_json, ''), created_at, updated_at FROM exercise_logs WHERE 1=1`
	args := make([]any, 0)
	if strings.TrimSpace(f.Date) != "" {
		start, end, err := dayBounds(f.Date)
		if err != nil {
			return nil, err
		}
		query += ` AND performed_at >= ? AND performed_at < ?`
		args = append(args, start, end)
	}
	if strings.TrimSpace(f.FromDate) != "" {
		from, err := parseDateStart(f.FromDate)
		if err != nil {
			return nil, err
		}
		query += ` AND performed_at >= ?`
		args = append(args, from)
	}
	if strings.TrimSpace(f.ToDate) != "" {
		to, err := parseDateEndExclusive(f.ToDate)
		if err != nil {
			return nil, err
		}
		query += ` AND performed_at < ?`
		args = append(args, to)
	}
	if strings.TrimSpace(f.ExerciseType) != "" {
		query += ` AND exercise_type = ?`
		args = append(args, strings.ToLower(strings.TrimSpace(f.ExerciseType)))
	}

	query += ` ORDER BY performed_at DESC`
	if f.Limit <= 0 {
		f.Limit = 50
	}
	query += ` LIMIT ?`
	args = append(args, f.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list exercise logs: %w", err)
	}
	defer rows.Close()

	items := make([]model.ExerciseLog, 0)
	for rows.Next() {
		var item model.ExerciseLog
		var duration sql.NullInt64
		var distance sql.NullFloat64
		var performedAtRaw string
		var createdRaw string
		var updatedRaw string
		if err := rows.Scan(&item.ID, &item.ExerciseType, &item.CaloriesBurned, &duration, &distance, &item.DistanceUnit, &performedAtRaw, &item.Notes, &item.Metadata, &createdRaw, &updatedRaw); err != nil {
			return nil, fmt.Errorf("scan exercise log: %w", err)
		}
		performedAt, err := time.Parse(time.RFC3339, performedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse performed_at: %w", err)
		}
		item.PerformedAt = performedAt
		if duration.Valid {
			v := int(duration.Int64)
			item.DurationMin = &v
		}
		if distance.Valid {
			v := distance.Float64
			item.Distance = &v
		}
		item.CreatedAt, _ = time.Parse(time.RFC3339, createdRaw)
		item.UpdatedAt, _ = time.Parse(time.RFC3339, updatedRaw)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate exercise logs: %w", err)
	}
	return items, nil
}

func UpdateExerciseLog(db *sql.DB, in UpdateExerciseInput) error {
	if in.ID <= 0 {
		return fmt.Errorf("exercise id must be > 0")
	}
	normalized, err := normalizeExerciseInput(in.ExerciseLogInput, true)
	if err != nil {
		return err
	}
	res, err := db.Exec(`
UPDATE exercise_logs
SET exercise_type = ?, calories_burned = ?, duration_min = ?, distance = ?, distance_unit = ?, performed_at = ?, notes = ?, metadata_json = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, normalized.ExerciseType, normalized.CaloriesBurned, normalized.DurationMin, normalized.Distance, nullableString(normalized.DistanceUnit), normalized.PerformedAt.Format(time.RFC3339), normalized.Notes, normalized.Metadata, in.ID)
	if err != nil {
		return fmt.Errorf("update exercise log %d: %w", in.ID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("exercise log %d not found", in.ID)
	}
	return nil
}

func DeleteExerciseLog(db *sql.DB, id int64) error {
	if id <= 0 {
		return fmt.Errorf("exercise id must be > 0")
	}
	res, err := db.Exec(`DELETE FROM exercise_logs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete exercise log %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("exercise log %d not found", id)
	}
	return nil
}

func normalizeExerciseInput(in ExerciseLogInput, requirePerformedAt bool) (ExerciseLogInput, error) {
	in.ExerciseType = strings.ToLower(strings.TrimSpace(in.ExerciseType))
	if in.ExerciseType == "" {
		return ExerciseLogInput{}, fmt.Errorf("exercise type is required")
	}
	if in.CaloriesBurned <= 0 {
		return ExerciseLogInput{}, fmt.Errorf("calories burned must be > 0")
	}
	if in.DurationMin != nil && *in.DurationMin <= 0 {
		return ExerciseLogInput{}, fmt.Errorf("duration must be > 0")
	}

	unit := strings.ToLower(strings.TrimSpace(in.DistanceUnit))
	if in.Distance == nil && unit != "" {
		return ExerciseLogInput{}, fmt.Errorf("distance must be provided when distance unit is set")
	}
	if in.Distance != nil && unit == "" {
		return ExerciseLogInput{}, fmt.Errorf("distance unit is required when distance is set")
	}
	if in.Distance != nil && *in.Distance <= 0 {
		return ExerciseLogInput{}, fmt.Errorf("distance must be > 0")
	}
	if unit != "" && unit != "km" && unit != "mi" {
		return ExerciseLogInput{}, fmt.Errorf("invalid distance unit %q (use km or mi)", in.DistanceUnit)
	}
	in.DistanceUnit = unit

	if requirePerformedAt && in.PerformedAt.IsZero() {
		return ExerciseLogInput{}, fmt.Errorf("exercise date/time is required")
	}
	if !requirePerformedAt && in.PerformedAt.IsZero() {
		in.PerformedAt = time.Now()
	}

	metadata, err := normalizeEntryMetadata(in.Metadata)
	if err != nil {
		return ExerciseLogInput{}, err
	}
	in.Metadata = metadata
	in.Notes = strings.TrimSpace(in.Notes)
	return in, nil
}

func nullableString(value string) any {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return value
}
