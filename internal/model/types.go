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
