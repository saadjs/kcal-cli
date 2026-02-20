package service

import (
	"database/sql"
	"time"
)

type TodayStatus struct {
	Date              string  `json:"date"`
	IntakeCalories    int     `json:"intake_calories"`
	ExerciseCalories  int     `json:"exercise_calories"`
	NetCalories       int     `json:"net_calories"`
	ProteinG          float64 `json:"protein_g"`
	CarbsG            float64 `json:"carbs_g"`
	FatG              float64 `json:"fat_g"`
	GoalCalories      int     `json:"goal_calories,omitempty"`
	GoalProteinG      float64 `json:"goal_protein_g,omitempty"`
	GoalCarbsG        float64 `json:"goal_carbs_g,omitempty"`
	GoalFatG          float64 `json:"goal_fat_g,omitempty"`
	RemainingCalories int     `json:"remaining_calories,omitempty"`
	RemainingProteinG float64 `json:"remaining_protein_g,omitempty"`
	RemainingCarbsG   float64 `json:"remaining_carbs_g,omitempty"`
	RemainingFatG     float64 `json:"remaining_fat_g,omitempty"`
	HasGoal           bool    `json:"has_goal"`
}

func TodaySummary(db *sql.DB, date time.Time) (*TodayStatus, error) {
	start := beginningOfDay(date)
	report, err := AnalyticsRange(db, start, start, 0.10)
	if err != nil {
		return nil, err
	}
	status := &TodayStatus{Date: start.Format("2006-01-02")}
	status.IntakeCalories = report.TotalIntakeCalories
	status.ExerciseCalories = report.TotalExerciseCalories
	status.NetCalories = report.TotalNetCalories
	status.ProteinG = report.TotalProtein
	status.CarbsG = report.TotalCarbs
	status.FatG = report.TotalFat

	goal, err := CurrentGoal(db, status.Date)
	if err != nil {
		return nil, err
	}
	if goal != nil {
		status.HasGoal = true
		status.GoalCalories = goal.Calories
		status.GoalProteinG = goal.ProteinG
		status.GoalCarbsG = goal.CarbsG
		status.GoalFatG = goal.FatG
		status.RemainingCalories = goal.Calories - status.NetCalories
		status.RemainingProteinG = goal.ProteinG - status.ProteinG
		status.RemainingCarbsG = goal.CarbsG - status.CarbsG
		status.RemainingFatG = goal.FatG - status.FatG
	}
	return status, nil
}
