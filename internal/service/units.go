package service

import (
	"fmt"
	"math"
	"strings"
)

type unitKind string

const (
	unitKindMass   unitKind = "mass"
	unitKindVolume unitKind = "volume"
)

type unitDef struct {
	kind       unitKind
	toBaseUnit float64
}

var unitTable = map[string]unitDef{
	// mass (base = g)
	"mg": {kind: unitKindMass, toBaseUnit: 0.001},
	"g":  {kind: unitKindMass, toBaseUnit: 1},
	"kg": {kind: unitKindMass, toBaseUnit: 1000},
	"oz": {kind: unitKindMass, toBaseUnit: 28.349523125},
	"lb": {kind: unitKindMass, toBaseUnit: 453.59237},
	"lbs": {
		kind:       unitKindMass,
		toBaseUnit: 453.59237,
	},

	// volume (base = ml)
	"ml":    {kind: unitKindVolume, toBaseUnit: 1},
	"l":     {kind: unitKindVolume, toBaseUnit: 1000},
	"tsp":   {kind: unitKindVolume, toBaseUnit: 4.92892159375},
	"tbsp":  {kind: unitKindVolume, toBaseUnit: 14.78676478125},
	"cup":   {kind: unitKindVolume, toBaseUnit: 236.5882365},
	"fl-oz": {kind: unitKindVolume, toBaseUnit: 29.5735295625},
}

type ScaledMacros struct {
	Calories int
	ProteinG float64
	CarbsG   float64
	FatG     float64
}

type ScaleIngredientMacrosInput struct {
	Amount      float64
	Unit        string
	RefAmount   float64
	RefUnit     string
	RefCalories int
	RefProteinG float64
	RefCarbsG   float64
	RefFatG     float64
	DensityGML  float64
}

func ScaleIngredientMacros(in ScaleIngredientMacrosInput) (ScaledMacros, error) {
	if in.Amount <= 0 {
		return ScaledMacros{}, fmt.Errorf("ingredient amount must be > 0")
	}
	if in.RefAmount <= 0 {
		return ScaledMacros{}, fmt.Errorf("reference amount must be > 0")
	}
	if err := validateNonNegativeInt("reference calories", in.RefCalories); err != nil {
		return ScaledMacros{}, err
	}
	if err := validateNonNegativeFloat("reference protein", in.RefProteinG); err != nil {
		return ScaledMacros{}, err
	}
	if err := validateNonNegativeFloat("reference carbs", in.RefCarbsG); err != nil {
		return ScaledMacros{}, err
	}
	if err := validateNonNegativeFloat("reference fat", in.RefFatG); err != nil {
		return ScaledMacros{}, err
	}

	targetInRefUnit, err := ConvertIngredientAmount(in.Amount, in.Unit, in.RefUnit, in.DensityGML)
	if err != nil {
		return ScaledMacros{}, err
	}
	factor := targetInRefUnit / in.RefAmount

	return ScaledMacros{
		Calories: int(math.Round(float64(in.RefCalories) * factor)),
		ProteinG: in.RefProteinG * factor,
		CarbsG:   in.RefCarbsG * factor,
		FatG:     in.RefFatG * factor,
	}, nil
}

func ConvertIngredientAmount(value float64, fromUnit, toUnit string, densityGML float64) (float64, error) {
	if value <= 0 {
		return 0, fmt.Errorf("amount must be > 0")
	}
	from, ok := resolveUnit(fromUnit)
	if !ok {
		return 0, fmt.Errorf("unsupported unit %q", fromUnit)
	}
	to, ok := resolveUnit(toUnit)
	if !ok {
		return 0, fmt.Errorf("unsupported unit %q", toUnit)
	}

	if from.kind == to.kind {
		base := value * from.toBaseUnit
		return base / to.toBaseUnit, nil
	}

	if densityGML <= 0 {
		return 0, fmt.Errorf("density-g-per-ml must be > 0 for mass/volume conversion")
	}

	var grams float64
	switch from.kind {
	case unitKindMass:
		grams = value * from.toBaseUnit
	case unitKindVolume:
		ml := value * from.toBaseUnit
		grams = ml * densityGML
	default:
		return 0, fmt.Errorf("unsupported source unit kind")
	}

	switch to.kind {
	case unitKindMass:
		return grams / to.toBaseUnit, nil
	case unitKindVolume:
		ml := grams / densityGML
		return ml / to.toBaseUnit, nil
	default:
		return 0, fmt.Errorf("unsupported target unit kind")
	}
}

func resolveUnit(unit string) (unitDef, bool) {
	u := strings.ToLower(strings.TrimSpace(unit))
	def, ok := unitTable[u]
	return def, ok
}
