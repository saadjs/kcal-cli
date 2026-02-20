package model

import "time"

type Category struct {
	ID        int64
	Name      string
	IsDefault bool
	CreatedAt time.Time
}

type Entry struct {
	ID             int64
	Name           string
	Calories       int
	ProteinG       float64
	CarbsG         float64
	FatG           float64
	FiberG         float64
	SugarG         float64
	SodiumMg       float64
	CategoryID     int64
	Category       string
	ConsumedAt     time.Time
	Notes          string
	SourceType     string
	SourceID       *int64
	Metadata       string
	Micronutrients string
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

type ExerciseLog struct {
	ID             int64
	ExerciseType   string
	CaloriesBurned int
	DurationMin    *int
	Distance       *float64
	DistanceUnit   string
	PerformedAt    time.Time
	Notes          string
	Metadata       string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type SavedFood struct {
	ID                int64
	Name              string
	NameNorm          string
	Brand             string
	DefaultCategoryID int64
	DefaultCategory   string
	Calories          int
	ProteinG          float64
	CarbsG            float64
	FatG              float64
	FiberG            float64
	SugarG            float64
	SodiumMg          float64
	Micronutrients    string
	ServingAmount     float64
	ServingUnit       string
	SourceType        string
	SourceProvider    string
	SourceRef         string
	Notes             string
	Metadata          string
	UsageCount        int
	LastUsedAt        *time.Time
	ArchivedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SavedMeal struct {
	ID                int64
	Name              string
	NameNorm          string
	DefaultCategoryID int64
	DefaultCategory   string
	Notes             string
	CaloriesTotal     int
	ProteinTotalG     float64
	CarbsTotalG       float64
	FatTotalG         float64
	FiberTotalG       float64
	SugarTotalG       float64
	SodiumTotalMg     float64
	Micronutrients    string
	UsageCount        int
	LastUsedAt        *time.Time
	ArchivedAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type SavedMealComponent struct {
	ID             int64
	SavedMealID    int64
	SavedFoodID    *int64
	Position       int
	Name           string
	Quantity       float64
	Unit           string
	Calories       int
	ProteinG       float64
	CarbsG         float64
	FatG           float64
	FiberG         float64
	SugarG         float64
	SodiumMg       float64
	Micronutrients string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}
