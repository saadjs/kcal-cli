package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/model"
)

type SetBodyGoalInput struct {
	TargetWeight  float64
	Unit          string
	TargetBodyFat *float64
	TargetDate    string
	EffectiveDate string
}

func SetBodyGoal(db *sql.DB, in SetBodyGoalInput) error {
	weightKg, err := convertWeightToKg(in.TargetWeight, in.Unit)
	if err != nil {
		return err
	}
	if in.TargetBodyFat != nil {
		if *in.TargetBodyFat < 0 || *in.TargetBodyFat > 100 {
			return fmt.Errorf("target body-fat must be between 0 and 100")
		}
	}
	in.EffectiveDate = strings.TrimSpace(in.EffectiveDate)
	if in.EffectiveDate == "" {
		in.EffectiveDate = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", in.EffectiveDate); err != nil {
		return fmt.Errorf("invalid effective date %q (expected YYYY-MM-DD)", in.EffectiveDate)
	}
	in.TargetDate = strings.TrimSpace(in.TargetDate)
	if in.TargetDate != "" {
		if _, err := time.Parse("2006-01-02", in.TargetDate); err != nil {
			return fmt.Errorf("invalid target date %q (expected YYYY-MM-DD)", in.TargetDate)
		}
	}

	_, err = db.Exec(`
INSERT INTO body_goals(target_weight_kg, target_body_fat_pct, target_date, effective_date)
VALUES(?, ?, ?, ?)
ON CONFLICT(effective_date) DO UPDATE SET
  target_weight_kg=excluded.target_weight_kg,
  target_body_fat_pct=excluded.target_body_fat_pct,
  target_date=excluded.target_date
`, weightKg, in.TargetBodyFat, in.TargetDate, in.EffectiveDate)
	if err != nil {
		return fmt.Errorf("set body goal: %w", err)
	}
	return nil
}

func CurrentBodyGoal(db *sql.DB, date string) (*model.BodyGoal, error) {
	date = strings.TrimSpace(date)
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", date)
	}
	var out model.BodyGoal
	var bodyFat sql.NullFloat64
	var targetDate sql.NullString
	err := db.QueryRow(`
SELECT id, target_weight_kg, target_body_fat_pct, target_date, effective_date, created_at
FROM body_goals
WHERE effective_date <= ?
ORDER BY effective_date DESC
LIMIT 1
`, date).Scan(&out.ID, &out.TargetWeightKg, &bodyFat, &targetDate, &out.EffectiveDate, &out.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("current body goal for %s: %w", date, err)
	}
	if bodyFat.Valid {
		v := bodyFat.Float64
		out.TargetBodyFatPct = &v
	}
	if targetDate.Valid {
		out.TargetDate = targetDate.String
	}
	return &out, nil
}

func BodyGoalHistory(db *sql.DB) ([]model.BodyGoal, error) {
	rows, err := db.Query(`
SELECT id, target_weight_kg, target_body_fat_pct, target_date, effective_date, created_at
FROM body_goals
ORDER BY effective_date DESC
`)
	if err != nil {
		return nil, fmt.Errorf("list body goal history: %w", err)
	}
	defer rows.Close()
	items := make([]model.BodyGoal, 0)
	for rows.Next() {
		var out model.BodyGoal
		var bodyFat sql.NullFloat64
		var targetDate sql.NullString
		if err := rows.Scan(&out.ID, &out.TargetWeightKg, &bodyFat, &targetDate, &out.EffectiveDate, &out.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan body goal: %w", err)
		}
		if bodyFat.Valid {
			v := bodyFat.Float64
			out.TargetBodyFatPct = &v
		}
		if targetDate.Valid {
			out.TargetDate = targetDate.String
		}
		items = append(items, out)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate body goals: %w", err)
	}
	return items, nil
}
