package model

import "time"

type Category struct {
	ID        int64
	Name      string
	IsDefault bool
	CreatedAt time.Time
}

type Entry struct {
	ID         int64
	Name       string
	Calories   int
	ProteinG   float64
	CarbsG     float64
	FatG       float64
	CategoryID int64
	Category   string
	ConsumedAt time.Time
	Notes      string
	SourceType string
	SourceID   *int64
}

type Goal struct {
	ID            int64
	Calories      int
	ProteinG      float64
	CarbsG        float64
	FatG          float64
	EffectiveDate string
	CreatedAt     time.Time
}

type Recipe struct {
	ID            int64
	Name          string
	CaloriesTotal int
	ProteinTotalG float64
	CarbsTotalG   float64
	FatTotalG     float64
	Servings      float64
	Notes         string
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

type BodyMeasurement struct {
	ID         int64
	MeasuredAt time.Time
	WeightKg   float64
	BodyFatPct *float64
	Notes      string
}

type BodyGoal struct {
	ID               int64
	TargetWeightKg   float64
	TargetBodyFatPct *float64
	TargetDate       string
	EffectiveDate    string
	CreatedAt        time.Time
}

type RecipeIngredient struct {
	ID         int64
	RecipeID   int64
	Name       string
	Amount     float64
	AmountUnit string
	Calories   int
	ProteinG   float64
	CarbsG     float64
	FatG       float64
	CreatedAt  time.Time
	UpdatedAt  time.Time
}
