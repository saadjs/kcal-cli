package service

import (
	"database/sql"
	"fmt"
	"sort"
	"time"
)

type CategoryBreakdown struct {
	Category string  `json:"category"`
	Calories int     `json:"calories"`
	Protein  float64 `json:"protein_g"`
	Carbs    float64 `json:"carbs_g"`
	Fat      float64 `json:"fat_g"`
}

type DaySummary struct {
	Date     string  `json:"date"`
	Calories int     `json:"calories"`
	Protein  float64 `json:"protein_g"`
	Carbs    float64 `json:"carbs_g"`
	Fat      float64 `json:"fat_g"`
}

type AnalyticsReport struct {
	FromDate              string              `json:"from_date"`
	ToDate                string              `json:"to_date"`
	TotalCalories         int                 `json:"total_calories"`
	TotalProtein          float64             `json:"total_protein_g"`
	TotalCarbs            float64             `json:"total_carbs_g"`
	TotalFat              float64             `json:"total_fat_g"`
	DaysWithEntries       int                 `json:"days_with_entries"`
	AverageCaloriesPerDay float64             `json:"avg_calories_per_day"`
	AverageProteinPerDay  float64             `json:"avg_protein_per_day"`
	AverageCarbsPerDay    float64             `json:"avg_carbs_per_day"`
	AverageFatPerDay      float64             `json:"avg_fat_per_day"`
	HighestDay            *DaySummary         `json:"highest_day,omitempty"`
	LowestDay             *DaySummary         `json:"lowest_day,omitempty"`
	Adherence             AdherenceSummary    `json:"adherence"`
	ByCategory            []CategoryBreakdown `json:"by_category"`
	Days                  []DaySummary        `json:"days"`
}

type AdherenceSummary struct {
	EvaluatedDays   int     `json:"evaluated_days"`
	WithinGoalDays  int     `json:"within_goal_days"`
	PercentWithin   float64 `json:"percent_within_goal"`
	SkippedGoalDays int     `json:"days_without_goal"`
}

func AnalyticsRange(db *sql.DB, from, to time.Time, tolerance float64) (*AnalyticsReport, error) {
	if from.After(to) {
		return nil, fmt.Errorf("from date must be <= to date")
	}
	from = beginningOfDay(from)
	to = beginningOfDay(to)

	report := &AnalyticsReport{
		FromDate: from.Format("2006-01-02"),
		ToDate:   to.Format("2006-01-02"),
	}

	days, err := loadDaySummaries(db, from, to)
	if err != nil {
		return nil, err
	}
	report.Days = days
	report.DaysWithEntries = len(days)

	for i := range days {
		report.TotalCalories += days[i].Calories
		report.TotalProtein += days[i].Protein
		report.TotalCarbs += days[i].Carbs
		report.TotalFat += days[i].Fat
	}
	if report.DaysWithEntries > 0 {
		div := float64(report.DaysWithEntries)
		report.AverageCaloriesPerDay = float64(report.TotalCalories) / div
		report.AverageProteinPerDay = report.TotalProtein / div
		report.AverageCarbsPerDay = report.TotalCarbs / div
		report.AverageFatPerDay = report.TotalFat / div
		report.HighestDay, report.LowestDay = extremeDays(days)
	}

	categories, err := loadCategoryBreakdown(db, from, to)
	if err != nil {
		return nil, err
	}
	report.ByCategory = categories

	adherence, err := calculateAdherence(db, days, tolerance)
	if err != nil {
		return nil, err
	}
	report.Adherence = adherence

	return report, nil
}

func loadDaySummaries(db *sql.DB, from, to time.Time) ([]DaySummary, error) {
	rows, err := db.Query(`
SELECT substr(consumed_at, 1, 10) as day, SUM(calories), SUM(protein_g), SUM(carbs_g), SUM(fat_g)
FROM entries
WHERE consumed_at >= ? AND consumed_at < ?
GROUP BY day
ORDER BY day ASC
`, from.Format(time.RFC3339), to.Add(24*time.Hour).Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query day summaries: %w", err)
	}
	defer rows.Close()

	items := make([]DaySummary, 0)
	for rows.Next() {
		var d DaySummary
		if err := rows.Scan(&d.Date, &d.Calories, &d.Protein, &d.Carbs, &d.Fat); err != nil {
			return nil, fmt.Errorf("scan day summary: %w", err)
		}
		items = append(items, d)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate day summaries: %w", err)
	}
	return items, nil
}

func loadCategoryBreakdown(db *sql.DB, from, to time.Time) ([]CategoryBreakdown, error) {
	rows, err := db.Query(`
SELECT c.name, SUM(e.calories), SUM(e.protein_g), SUM(e.carbs_g), SUM(e.fat_g)
FROM entries e
JOIN categories c ON c.id = e.category_id
WHERE e.consumed_at >= ? AND e.consumed_at < ?
GROUP BY c.name
ORDER BY SUM(e.calories) DESC
`, from.Format(time.RFC3339), to.Add(24*time.Hour).Format(time.RFC3339))
	if err != nil {
		return nil, fmt.Errorf("query category breakdown: %w", err)
	}
	defer rows.Close()

	items := make([]CategoryBreakdown, 0)
	for rows.Next() {
		var c CategoryBreakdown
		if err := rows.Scan(&c.Category, &c.Calories, &c.Protein, &c.Carbs, &c.Fat); err != nil {
			return nil, fmt.Errorf("scan category breakdown: %w", err)
		}
		items = append(items, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate category breakdown: %w", err)
	}
	return items, nil
}

func calculateAdherence(db *sql.DB, days []DaySummary, tolerance float64) (AdherenceSummary, error) {
	out := AdherenceSummary{}
	for _, d := range days {
		goal, err := CurrentGoal(db, d.Date)
		if err != nil {
			return out, err
		}
		if goal == nil {
			out.SkippedGoalDays++
			continue
		}
		out.EvaluatedDays++
		if float64(d.Calories) <= float64(goal.Calories) &&
			AdherenceWithin(d.Protein, goal.ProteinG, tolerance) &&
			AdherenceWithin(d.Carbs, goal.CarbsG, tolerance) &&
			AdherenceWithin(d.Fat, goal.FatG, tolerance) {
			out.WithinGoalDays++
		}
	}
	if out.EvaluatedDays > 0 {
		out.PercentWithin = (float64(out.WithinGoalDays) / float64(out.EvaluatedDays)) * 100
	}
	return out, nil
}

func extremeDays(days []DaySummary) (*DaySummary, *DaySummary) {
	if len(days) == 0 {
		return nil, nil
	}
	copied := make([]DaySummary, len(days))
	copy(copied, days)
	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].Calories < copied[j].Calories
	})
	low := copied[0]
	high := copied[len(copied)-1]
	return &high, &low
}

func beginningOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, time.Local)
}
