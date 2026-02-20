package service

import (
	"database/sql"
	"encoding/json"
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

type BodyPoint struct {
	Date       string   `json:"date"`
	WeightKg   float64  `json:"weight_kg"`
	BodyFatPct *float64 `json:"body_fat_pct,omitempty"`
	LeanMassKg *float64 `json:"lean_mass_kg,omitempty"`
}

type BodyGoalProgress struct {
	ActiveGoalEffectiveDate string   `json:"active_goal_effective_date,omitempty"`
	TargetWeightKg          float64  `json:"target_weight_kg,omitempty"`
	TargetBodyFatPct        *float64 `json:"target_body_fat_pct,omitempty"`
	LatestWeightKg          float64  `json:"latest_weight_kg,omitempty"`
	LatestBodyFatPct        *float64 `json:"latest_body_fat_pct,omitempty"`
	WeightDeltaKg           float64  `json:"weight_delta_kg,omitempty"`
	BodyFatDeltaPct         *float64 `json:"body_fat_delta_pct,omitempty"`
}

type BodySummary struct {
	MeasurementsCount int               `json:"measurements_count"`
	StartWeightKg     float64           `json:"start_weight_kg,omitempty"`
	EndWeightKg       float64           `json:"end_weight_kg,omitempty"`
	WeightChangeKg    float64           `json:"weight_change_kg,omitempty"`
	AvgWeeklyChangeKg float64           `json:"avg_weekly_change_kg,omitempty"`
	StartBodyFatPct   *float64          `json:"start_body_fat_pct,omitempty"`
	EndBodyFatPct     *float64          `json:"end_body_fat_pct,omitempty"`
	BodyFatChangePct  *float64          `json:"body_fat_change_pct,omitempty"`
	StartLeanMassKg   *float64          `json:"start_lean_mass_kg,omitempty"`
	EndLeanMassKg     *float64          `json:"end_lean_mass_kg,omitempty"`
	LeanMassChangeKg  *float64          `json:"lean_mass_change_kg,omitempty"`
	GoalProgress      *BodyGoalProgress `json:"goal_progress,omitempty"`
	Points            []BodyPoint       `json:"points"`
}

type ConfidenceStats struct {
	Count int     `json:"count"`
	Avg   float64 `json:"avg"`
	Min   float64 `json:"min"`
	Max   float64 `json:"max"`
}

type MetadataSummary struct {
	SourceCounts      map[string]int  `json:"source_counts"`
	BarcodeTierCounts map[string]int  `json:"barcode_tier_counts"`
	Confidence        ConfidenceStats `json:"confidence"`
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
	Body                  BodySummary         `json:"body"`
	Metadata              MetadataSummary     `json:"metadata"`
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

	body, err := calculateBodySummary(db, from, to)
	if err != nil {
		return nil, err
	}
	report.Body = body

	metadata, err := calculateMetadataSummary(db, from, to)
	if err != nil {
		return nil, err
	}
	report.Metadata = metadata

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

func calculateBodySummary(db *sql.DB, from, to time.Time) (BodySummary, error) {
	rows, err := db.Query(`
SELECT measured_at, weight_kg, body_fat_pct
FROM body_measurements
WHERE measured_at >= ? AND measured_at < ?
ORDER BY measured_at ASC
`, from.Format(time.RFC3339), to.Add(24*time.Hour).Format(time.RFC3339))
	if err != nil {
		return BodySummary{}, fmt.Errorf("query body measurements: %w", err)
	}
	defer rows.Close()

	summary := BodySummary{Points: make([]BodyPoint, 0)}
	for rows.Next() {
		var measured string
		var weight float64
		var bodyFat sql.NullFloat64
		if err := rows.Scan(&measured, &weight, &bodyFat); err != nil {
			return BodySummary{}, fmt.Errorf("scan body measurement analytics row: %w", err)
		}
		t, err := time.Parse(time.RFC3339, measured)
		if err != nil {
			return BodySummary{}, fmt.Errorf("parse measured_at analytics row: %w", err)
		}
		p := BodyPoint{Date: t.Format("2006-01-02"), WeightKg: weight}
		if bodyFat.Valid {
			bf := bodyFat.Float64
			p.BodyFatPct = &bf
			lean := leanMassKg(weight, bf)
			p.LeanMassKg = &lean
		}
		summary.Points = append(summary.Points, p)
	}
	if err := rows.Err(); err != nil {
		return BodySummary{}, fmt.Errorf("iterate body measurements analytics rows: %w", err)
	}

	summary.MeasurementsCount = len(summary.Points)
	if summary.MeasurementsCount == 0 {
		return summary, nil
	}

	start := summary.Points[0]
	end := summary.Points[summary.MeasurementsCount-1]
	summary.StartWeightKg = start.WeightKg
	summary.EndWeightKg = end.WeightKg
	summary.WeightChangeKg = end.WeightKg - start.WeightKg

	daysSpan := to.Sub(from).Hours() / 24
	if daysSpan >= 7 {
		summary.AvgWeeklyChangeKg = summary.WeightChangeKg / (daysSpan / 7)
	}

	if start.BodyFatPct != nil {
		summary.StartBodyFatPct = start.BodyFatPct
		summary.StartLeanMassKg = start.LeanMassKg
	}
	if end.BodyFatPct != nil {
		summary.EndBodyFatPct = end.BodyFatPct
		summary.EndLeanMassKg = end.LeanMassKg
	}
	if summary.StartBodyFatPct != nil && summary.EndBodyFatPct != nil {
		delta := *summary.EndBodyFatPct - *summary.StartBodyFatPct
		summary.BodyFatChangePct = &delta
	}
	if summary.StartLeanMassKg != nil && summary.EndLeanMassKg != nil {
		delta := *summary.EndLeanMassKg - *summary.StartLeanMassKg
		summary.LeanMassChangeKg = &delta
	}

	latest := end
	goal, err := CurrentBodyGoal(db, end.Date)
	if err != nil {
		return BodySummary{}, err
	}
	if goal != nil {
		progress := &BodyGoalProgress{
			ActiveGoalEffectiveDate: goal.EffectiveDate,
			TargetWeightKg:          goal.TargetWeightKg,
			TargetBodyFatPct:        goal.TargetBodyFatPct,
			LatestWeightKg:          latest.WeightKg,
			LatestBodyFatPct:        latest.BodyFatPct,
			WeightDeltaKg:           latest.WeightKg - goal.TargetWeightKg,
		}
		if goal.TargetBodyFatPct != nil && latest.BodyFatPct != nil {
			delta := *latest.BodyFatPct - *goal.TargetBodyFatPct
			progress.BodyFatDeltaPct = &delta
		}
		summary.GoalProgress = progress
	}

	return summary, nil
}

func leanMassKg(weightKg, bodyFatPct float64) float64 {
	return weightKg * (1 - (bodyFatPct / 100))
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

func calculateMetadataSummary(db *sql.DB, from, to time.Time) (MetadataSummary, error) {
	rows, err := db.Query(`
SELECT source_type, IFNULL(metadata_json, '')
FROM entries
WHERE consumed_at >= ? AND consumed_at < ?
`, from.Format(time.RFC3339), to.Add(24*time.Hour).Format(time.RFC3339))
	if err != nil {
		return MetadataSummary{}, fmt.Errorf("query metadata analytics: %w", err)
	}
	defer rows.Close()

	out := MetadataSummary{
		SourceCounts:      map[string]int{},
		BarcodeTierCounts: map[string]int{},
	}
	confMinSet := false
	for rows.Next() {
		var sourceType string
		var metadataRaw string
		if err := rows.Scan(&sourceType, &metadataRaw); err != nil {
			return MetadataSummary{}, fmt.Errorf("scan metadata analytics: %w", err)
		}
		out.SourceCounts[sourceType]++

		if metadataRaw == "" {
			continue
		}
		var meta map[string]any
		if err := json.Unmarshal([]byte(metadataRaw), &meta); err != nil {
			continue
		}
		if tier, ok := meta["source_tier"].(string); ok && tier != "" {
			out.BarcodeTierCounts[tier]++
		}
		if v, ok := meta["provider_confidence"]; ok {
			conf, ok := numericValue(v)
			if ok {
				out.Confidence.Count++
				out.Confidence.Avg += conf
				if !confMinSet || conf < out.Confidence.Min {
					out.Confidence.Min = conf
					confMinSet = true
				}
				if conf > out.Confidence.Max {
					out.Confidence.Max = conf
				}
			}
		}
	}
	if err := rows.Err(); err != nil {
		return MetadataSummary{}, fmt.Errorf("iterate metadata analytics: %w", err)
	}
	if out.Confidence.Count > 0 {
		out.Confidence.Avg = out.Confidence.Avg / float64(out.Confidence.Count)
	}
	return out, nil
}

func numericValue(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	default:
		return 0, false
	}
}
