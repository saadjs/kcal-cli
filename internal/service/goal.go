package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/model"
)

type SetGoalInput struct {
	Calories      int
	ProteinG      float64
	CarbsG        float64
	FatG          float64
	EffectiveDate string
}

func SetGoal(db *sql.DB, in SetGoalInput) error {
	if err := validateNonNegativeInt("calories", in.Calories); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("protein", in.ProteinG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("carbs", in.CarbsG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("fat", in.FatG); err != nil {
		return err
	}
	in.EffectiveDate = strings.TrimSpace(in.EffectiveDate)
	if in.EffectiveDate == "" {
		in.EffectiveDate = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", in.EffectiveDate); err != nil {
		return fmt.Errorf("invalid effective date %q (expected YYYY-MM-DD)", in.EffectiveDate)
	}

	_, err := db.Exec(`
INSERT INTO goals(calories, protein_g, carbs_g, fat_g, effective_date)
VALUES(?, ?, ?, ?, ?)
ON CONFLICT(effective_date) DO UPDATE SET
  calories=excluded.calories,
  protein_g=excluded.protein_g,
  carbs_g=excluded.carbs_g,
  fat_g=excluded.fat_g
`, in.Calories, in.ProteinG, in.CarbsG, in.FatG, in.EffectiveDate)
	if err != nil {
		return fmt.Errorf("set goal: %w", err)
	}
	return nil
}

func CurrentGoal(db *sql.DB, date string) (*model.Goal, error) {
	date = strings.TrimSpace(date)
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	if _, err := time.Parse("2006-01-02", date); err != nil {
		return nil, fmt.Errorf("invalid date %q (expected YYYY-MM-DD)", date)
	}

	var g model.Goal
	err := db.QueryRow(`
SELECT id, calories, protein_g, carbs_g, fat_g, effective_date, created_at
FROM goals
WHERE effective_date <= ?
ORDER BY effective_date DESC
LIMIT 1
`, date).Scan(&g.ID, &g.Calories, &g.ProteinG, &g.CarbsG, &g.FatG, &g.EffectiveDate, &g.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("current goal for %s: %w", date, err)
	}
	return &g, nil
}

func GoalHistory(db *sql.DB) ([]model.Goal, error) {
	rows, err := db.Query(`
SELECT id, calories, protein_g, carbs_g, fat_g, effective_date, created_at
FROM goals
ORDER BY effective_date DESC
`)
	if err != nil {
		return nil, fmt.Errorf("list goal history: %w", err)
	}
	defer rows.Close()

	goals := make([]model.Goal, 0)
	for rows.Next() {
		var g model.Goal
		if err := rows.Scan(&g.ID, &g.Calories, &g.ProteinG, &g.CarbsG, &g.FatG, &g.EffectiveDate, &g.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan goal history: %w", err)
		}
		goals = append(goals, g)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate goal history: %w", err)
	}
	return goals, nil
}

func AdherenceWithin(actual float64, target float64, tolerance float64) bool {
	if target == 0 {
		return actual == 0
	}
	lower := target * (1 - tolerance)
	upper := target * (1 + tolerance)
	return actual >= lower && actual <= upper
}
