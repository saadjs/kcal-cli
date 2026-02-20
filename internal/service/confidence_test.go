package service

import "testing"

func TestScoreBarcodeConfidenceOverrideIsAlwaysVerified(t *testing.T) {
	r := BarcodeLookupResult{Provider: BarcodeProviderUSDA, SourceTier: "override", Calories: 100}
	out := ScoreBarcodeConfidence(r, DefaultVerifiedMinScore)
	if out.Score != 1 {
		t.Fatalf("expected score 1 for override, got %.3f", out.Score)
	}
	if !out.IsVerified {
		t.Fatalf("expected override to be verified")
	}
}

func TestScoreSearchConfidenceThreshold(t *testing.T) {
	r := BarcodeLookupResult{
		Provider:      BarcodeProviderUSDA,
		Description:   "Greek Yogurt",
		Brand:         "Fage",
		ServingAmount: 170,
		ServingUnit:   "g",
		Calories:      120,
		ProteinG:      15,
		CarbsG:        8,
		FatG:          3,
	}
	out := ScoreSearchConfidence(r, "fage greek yogurt", 0.80)
	if out.Score < 0.80 {
		t.Fatalf("expected score >= 0.80, got %.3f", out.Score)
	}
	if !out.IsVerified {
		t.Fatalf("expected verified search result")
	}
}

func TestScoreBarcodeConfidenceUsesSourceIDWhenNotExact(t *testing.T) {
	r := BarcodeLookupResult{
		Provider:      BarcodeProviderUSDA,
		Barcode:       "012345678905",
		ExactMatch:    false,
		SourceID:      123,
		Description:   "Fallback",
		ServingAmount: 100,
		ServingUnit:   "g",
		Calories:      100,
		ProteinG:      10,
		CarbsG:        5,
		FatG:          2,
	}
	out := ScoreBarcodeConfidence(r, DefaultVerifiedMinScore)
	if out.Score >= 0.95 {
		t.Fatalf("expected lower score than exact match, got %.3f", out.Score)
	}
}

func TestScoreSearchConfidenceIdentityGuard(t *testing.T) {
	r := BarcodeLookupResult{
		Provider:      BarcodeProviderUSDA,
		Description:   "Unrelated Dressing With Yogurt Base",
		Brand:         "Brand",
		ServingAmount: 30,
		ServingUnit:   "g",
		Calories:      120,
		ProteinG:      8,
		CarbsG:        9,
		FatG:          5,
	}
	out := ScoreSearchConfidence(r, "greek yogurt", 0.80)
	if out.IsVerified {
		t.Fatalf("expected identity guard to prevent verified result: %+v", out)
	}
}
