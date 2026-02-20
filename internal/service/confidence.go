package service

import (
	"fmt"
	"math"
	"regexp"
	"sort"
	"strings"
)

const DefaultVerifiedMinScore = 0.80

type ConfidenceScore struct {
	Score      float64  `json:"score"`
	IsVerified bool     `json:"is_verified"`
	Reasons    []string `json:"reasons,omitempty"`
}

func ScoreBarcodeConfidence(r BarcodeLookupResult, minScore float64) ConfidenceScore {
	if minScore <= 0 {
		minScore = DefaultVerifiedMinScore
	}
	if strings.EqualFold(strings.TrimSpace(r.SourceTier), "override") {
		return ConfidenceScore{
			Score:      1,
			IsVerified: true,
			Reasons: []string{
				"provider_trust=1.00 (override)",
				"identity_quality=1.00 (exact barcode match)",
				"score=1.00",
			},
		}
	}

	providerTrust := providerBaseConfidence(r.Provider)
	nutritionQuality := scoreNutritionQuality(r)
	servingQuality := scoreServingQuality(r.ServingAmount, r.ServingUnit)
	identityQuality, identityReason := scoreBarcodeIdentityQuality(r)

	score := 0.45*providerTrust + 0.25*nutritionQuality + 0.15*servingQuality + 0.15*identityQuality
	score = clamp01(score)
	verified := score >= minScore

	return ConfidenceScore{
		Score:      score,
		IsVerified: verified,
		Reasons: []string{
			fmt.Sprintf("provider_trust=%.2f", providerTrust),
			fmt.Sprintf("nutrition_quality=%.2f", nutritionQuality),
			fmt.Sprintf("serving_quality=%.2f", servingQuality),
			fmt.Sprintf("identity_quality=%.2f (%s)", identityQuality, identityReason),
			fmt.Sprintf("score=%.2f", score),
			fmt.Sprintf("verified_threshold=%.2f", minScore),
		},
	}
}

func ScoreSearchConfidence(r BarcodeLookupResult, query string, minScore float64) ConfidenceScore {
	if minScore <= 0 {
		minScore = DefaultVerifiedMinScore
	}
	providerTrust := providerBaseConfidence(r.Provider)
	nutritionQuality := scoreNutritionQuality(r)
	servingQuality := scoreServingQuality(r.ServingAmount, r.ServingUnit)
	identityQuality, identityReason := scoreSearchIdentityQuality(query, r.Description, r.Brand)

	score := 0.45*providerTrust + 0.25*nutritionQuality + 0.15*servingQuality + 0.15*identityQuality
	score = clamp01(score)
	verified := score >= minScore && identityQuality >= 0.7
	thresholdReason := fmt.Sprintf("verified_threshold=%.2f", minScore)
	if identityQuality < 0.7 {
		thresholdReason += " + identity_guard(identity>=0.70)"
	}

	return ConfidenceScore{
		Score:      score,
		IsVerified: verified,
		Reasons: []string{
			fmt.Sprintf("provider_trust=%.2f", providerTrust),
			fmt.Sprintf("nutrition_quality=%.2f", nutritionQuality),
			fmt.Sprintf("serving_quality=%.2f", servingQuality),
			fmt.Sprintf("identity_quality=%.2f (%s)", identityQuality, identityReason),
			fmt.Sprintf("score=%.2f", score),
			thresholdReason,
		},
	}
}

func scoreNutritionQuality(r BarcodeLookupResult) float64 {
	macroCount := 0
	if r.ProteinG > 0 {
		macroCount++
	}
	if r.CarbsG > 0 {
		macroCount++
	}
	if r.FatG > 0 {
		macroCount++
	}

	if r.Calories > 0 && macroCount == 3 {
		return 1.0
	}
	if r.Calories > 0 && macroCount >= 2 {
		return 0.7
	}
	if r.Calories > 0 || macroCount > 0 || r.FiberG > 0 || r.SugarG > 0 || r.SodiumMg > 0 || len(r.Micronutrients) > 0 {
		return 0.4
	}
	return 0.2
}

func scoreServingQuality(amount float64, unit string) float64 {
	hasAmount := amount > 0
	hasUnit := strings.TrimSpace(unit) != ""
	switch {
	case hasAmount && hasUnit:
		return 1.0
	case hasAmount || hasUnit:
		return 0.5
	default:
		return 0.0
	}
}

func scoreBarcodeIdentityQuality(r BarcodeLookupResult) (float64, string) {
	if r.ExactMatch {
		return 1.0, "exact barcode match"
	}
	if r.SourceID > 0 {
		return 0.8, "source id present"
	}
	return 0.5, "weak identity evidence"
}

func scoreSearchIdentityQuality(query, description, brand string) (float64, string) {
	queryTokens := tokenize(query)
	descTokens := tokenize(description)
	brandTokens := tokenize(brand)
	if len(queryTokens) == 0 {
		return 0.4, "empty query tokens"
	}

	querySet := map[string]bool{}
	for _, t := range queryTokens {
		querySet[t] = true
	}
	descSet := map[string]bool{}
	for _, t := range descTokens {
		descSet[t] = true
	}
	brandSet := map[string]bool{}
	for _, t := range brandTokens {
		brandSet[t] = true
	}

	matched := 0
	brandMatched := false
	for t := range querySet {
		if descSet[t] {
			matched++
		}
		if brandSet[t] {
			brandMatched = true
		}
	}
	overlap := float64(matched) / math.Max(1, float64(len(querySet)))

	if overlap >= 0.75 && brandMatched {
		return 1.0, "high token overlap with brand match"
	}
	if overlap >= 0.5 && brandMatched {
		return 0.7, "moderate token overlap with brand match"
	}
	return 0.4, "weak token overlap"
}

func tokenize(s string) []string {
	s = strings.ToLower(strings.TrimSpace(s))
	if s == "" {
		return nil
	}
	re := regexp.MustCompile(`[^a-z0-9]+`)
	s = re.ReplaceAllString(s, " ")
	parts := strings.Fields(s)
	seen := map[string]bool{}
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p == "" || seen[p] {
			continue
		}
		seen[p] = true
		out = append(out, p)
	}
	sort.Strings(out)
	return out
}

func clamp01(v float64) float64 {
	if v < 0 {
		return 0
	}
	if v > 1 {
		return 1
	}
	return math.Round(v*1000) / 1000
}
