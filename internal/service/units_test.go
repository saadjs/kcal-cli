package service_test

import (
	"math"
	"testing"

	"github.com/saad/kcal-cli/internal/service"
)

func TestConvertIngredientAmountSameDimension(t *testing.T) {
	t.Parallel()
	out, err := service.ConvertIngredientAmount(100, "g", "oz", 0)
	if err != nil {
		t.Fatalf("convert mass units: %v", err)
	}
	if math.Abs(out-3.5274) > 0.01 {
		t.Fatalf("expected ~3.53 oz, got %.4f", out)
	}
}

func TestConvertIngredientAmountCrossDimensionRequiresDensity(t *testing.T) {
	t.Parallel()
	_, err := service.ConvertIngredientAmount(1, "cup", "g", 0)
	if err == nil {
		t.Fatalf("expected density requirement error")
	}
}

func TestConvertIngredientAmountCrossDimensionWithDensity(t *testing.T) {
	t.Parallel()
	out, err := service.ConvertIngredientAmount(1, "cup", "g", 1.05)
	if err != nil {
		t.Fatalf("convert volume to mass with density: %v", err)
	}
	if math.Abs(out-248.4) > 0.5 {
		t.Fatalf("expected ~248.4 g, got %.4f", out)
	}
}

func TestScaleIngredientMacros(t *testing.T) {
	t.Parallel()
	scaled, err := service.ScaleIngredientMacros(service.ScaleIngredientMacrosInput{
		Amount:      150,
		Unit:        "g",
		RefAmount:   100,
		RefUnit:     "g",
		RefCalories: 130,
		RefProteinG: 2.4,
		RefCarbsG:   28,
		RefFatG:     0.3,
	})
	if err != nil {
		t.Fatalf("scale macros: %v", err)
	}
	if scaled.Calories != 195 {
		t.Fatalf("expected calories 195, got %d", scaled.Calories)
	}
	if math.Abs(scaled.ProteinG-3.6) > 0.01 {
		t.Fatalf("expected protein 3.6, got %.3f", scaled.ProteinG)
	}
}

func TestScaleIngredientMacrosUnsupportedUnit(t *testing.T) {
	t.Parallel()
	_, err := service.ScaleIngredientMacros(service.ScaleIngredientMacrosInput{
		Amount:      1,
		Unit:        "banana",
		RefAmount:   1,
		RefUnit:     "g",
		RefCalories: 10,
		RefProteinG: 1,
		RefCarbsG:   1,
		RefFatG:     1,
	})
	if err == nil {
		t.Fatalf("expected unsupported unit error")
	}
}
