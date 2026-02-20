package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/provider/openfoodfacts"
	"github.com/saadjs/kcal-cli/internal/provider/upcitemdb"
	"github.com/saadjs/kcal-cli/internal/provider/usda"
)

const (
	BarcodeProviderUSDA          = "usda"
	BarcodeProviderOpenFoodFacts = "openfoodfacts"
	BarcodeProviderUPCItemDB     = "upcitemdb"
	defaultBarcodeTTL            = 30 * 24 * time.Hour
)

type BarcodeLookupResult struct {
	Provider              string         `json:"provider"`
	Barcode               string         `json:"barcode"`
	Description           string         `json:"description"`
	Brand                 string         `json:"brand"`
	ServingAmount         float64        `json:"serving_amount"`
	ServingUnit           string         `json:"serving_unit"`
	Calories              float64        `json:"calories"`
	ProteinG              float64        `json:"protein_g"`
	CarbsG                float64        `json:"carbs_g"`
	FatG                  float64        `json:"fat_g"`
	FiberG                float64        `json:"fiber_g"`
	SugarG                float64        `json:"sugar_g"`
	SodiumMg              float64        `json:"sodium_mg"`
	Micronutrients        Micronutrients `json:"micronutrients,omitempty"`
	SourceID              int64          `json:"source_id"`
	SourceTier            string         `json:"source_tier,omitempty"`
	ExactMatch            bool           `json:"exact_match,omitempty"`
	ConfidenceScore       float64        `json:"confidence_score,omitempty"`
	IsVerified            bool           `json:"is_verified,omitempty"`
	VerificationReasons   []string       `json:"verification_reasons,omitempty"`
	ProviderConfidence    float64        `json:"provider_confidence,omitempty"`
	NutritionCompleteness string         `json:"nutrition_completeness,omitempty"`
	LookupTrail           []string       `json:"lookup_trail,omitempty"`
	FromOverride          bool           `json:"from_override"`
	FromCache             bool           `json:"from_cache"`
}

type BarcodeOverrideInput struct {
	Description    string
	Brand          string
	ServingAmount  float64
	ServingUnit    string
	Calories       float64
	ProteinG       float64
	CarbsG         float64
	FatG           float64
	FiberG         float64
	SugarG         float64
	SodiumMg       float64
	Micronutrients string
	Notes          string
}

type BarcodeLookupOptions struct {
	APIKey     string
	APIKeyType string
}

type BarcodeLookupCandidate struct {
	Provider string
	Options  BarcodeLookupOptions
}

type BarcodeCacheItem struct {
	Provider    string    `json:"provider"`
	Barcode     string    `json:"barcode"`
	Description string    `json:"description"`
	Brand       string    `json:"brand"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type barcodeClient interface {
	LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error)
}

func LookupBarcodeUSDA(db *sql.DB, apiKey, barcode string) (BarcodeLookupResult, error) {
	return LookupBarcode(db, BarcodeProviderUSDA, barcode, BarcodeLookupOptions{APIKey: apiKey})
}

func LookupBarcodeOpenFoodFacts(db *sql.DB, barcode string) (BarcodeLookupResult, error) {
	return LookupBarcode(db, BarcodeProviderOpenFoodFacts, barcode, BarcodeLookupOptions{})
}

func LookupBarcode(db *sql.DB, provider, barcode string, options BarcodeLookupOptions) (BarcodeLookupResult, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "", BarcodeProviderUSDA:
		return lookupBarcodeWithClient(db, BarcodeProviderUSDA, &usdaClientAdapter{client: &usda.Client{APIKey: options.APIKey}}, barcode)
	case BarcodeProviderOpenFoodFacts, "off":
		return lookupBarcodeWithClient(db, BarcodeProviderOpenFoodFacts, &openFoodFactsClientAdapter{client: &openfoodfacts.Client{}}, barcode)
	case BarcodeProviderUPCItemDB, "upc":
		return lookupBarcodeWithClient(db, BarcodeProviderUPCItemDB, &upcItemDBClientAdapter{client: &upcitemdb.Client{APIKey: options.APIKey, APIKeyType: options.APIKeyType}}, barcode)
	default:
		return BarcodeLookupResult{}, fmt.Errorf("unsupported barcode provider %q", provider)
	}
}

func RefreshBarcodeCache(db *sql.DB, provider, barcode string, options BarcodeLookupOptions) (BarcodeLookupResult, error) {
	provider = normalizeBarcodeProvider(provider)
	barcode = strings.TrimSpace(barcode)
	if provider == "" {
		return BarcodeLookupResult{}, fmt.Errorf("provider is required")
	}
	if !isValidBarcode(barcode) {
		return BarcodeLookupResult{}, fmt.Errorf("invalid barcode %q (expected 8-14 digits)", barcode)
	}

	if _, err := db.Exec(`DELETE FROM barcode_cache WHERE provider = ? AND barcode = ?`, provider, barcode); err != nil {
		return BarcodeLookupResult{}, fmt.Errorf("delete barcode cache row: %w", err)
	}
	var client barcodeClient
	switch provider {
	case BarcodeProviderUSDA:
		client = &usdaClientAdapter{client: &usda.Client{APIKey: options.APIKey}}
	case BarcodeProviderOpenFoodFacts:
		client = &openFoodFactsClientAdapter{client: &openfoodfacts.Client{}}
	case BarcodeProviderUPCItemDB:
		client = &upcItemDBClientAdapter{client: &upcitemdb.Client{APIKey: options.APIKey, APIKeyType: options.APIKeyType}}
	default:
		return BarcodeLookupResult{}, fmt.Errorf("unsupported barcode provider %q", provider)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, raw, err := client.LookupBarcode(ctx, barcode)
	if err != nil {
		return BarcodeLookupResult{}, err
	}
	result.Provider = provider
	result.Barcode = barcode
	result.SourceTier = "provider"
	result.NutritionCompleteness = deriveNutritionCompleteness(result)
	applyBarcodeConfidence(&result, DefaultVerifiedMinScore)
	if err := upsertBarcodeCache(db, result, raw, time.Now().Add(defaultBarcodeTTL)); err != nil {
		return BarcodeLookupResult{}, err
	}
	return result, nil
}

func ListBarcodeCache(db *sql.DB, provider string, limit int) ([]BarcodeCacheItem, error) {
	provider = normalizeBarcodeProvider(provider)
	if limit <= 0 {
		limit = 100
	}
	base := `SELECT provider, barcode, description, brand, expires_at FROM barcode_cache`
	args := make([]any, 0, 2)
	if provider != "" {
		base += ` WHERE provider = ?`
		args = append(args, provider)
	}
	base += ` ORDER BY fetched_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.Query(base, args...)
	if err != nil {
		return nil, fmt.Errorf("list barcode cache: %w", err)
	}
	defer rows.Close()
	out := make([]BarcodeCacheItem, 0)
	for rows.Next() {
		var item BarcodeCacheItem
		var expires string
		if err := rows.Scan(&item.Provider, &item.Barcode, &item.Description, &item.Brand, &expires); err != nil {
			return nil, fmt.Errorf("scan barcode cache: %w", err)
		}
		item.ExpiresAt, _ = time.Parse(time.RFC3339, expires)
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate barcode cache: %w", err)
	}
	return out, nil
}

func PurgeBarcodeCache(db *sql.DB, provider, barcode string, purgeAll bool) (int64, error) {
	provider = normalizeBarcodeProvider(provider)
	barcode = strings.TrimSpace(barcode)

	var (
		res sql.Result
		err error
	)
	switch {
	case purgeAll:
		res, err = db.Exec(`DELETE FROM barcode_cache`)
	case provider != "" && barcode != "":
		res, err = db.Exec(`DELETE FROM barcode_cache WHERE provider = ? AND barcode = ?`, provider, barcode)
	case provider != "":
		res, err = db.Exec(`DELETE FROM barcode_cache WHERE provider = ?`, provider)
	case barcode != "":
		res, err = db.Exec(`DELETE FROM barcode_cache WHERE barcode = ?`, barcode)
	default:
		return 0, fmt.Errorf("specify --all, --provider, --barcode, or provider+barcode")
	}
	if err != nil {
		return 0, fmt.Errorf("purge barcode cache: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return 0, fmt.Errorf("purge barcode cache rows affected: %w", err)
	}
	return affected, nil
}

func LookupBarcodeWithFallback(db *sql.DB, barcode string, candidates []BarcodeLookupCandidate) (BarcodeLookupResult, error) {
	if len(candidates) == 0 {
		return BarcodeLookupResult{}, fmt.Errorf("no lookup providers configured")
	}
	attempts := make([]string, 0, len(candidates))
	errs := make([]string, 0, len(candidates))
	for _, c := range candidates {
		provider := normalizeBarcodeProvider(c.Provider)
		if provider == "" {
			continue
		}
		attempts = append(attempts, provider)
		result, err := LookupBarcode(db, provider, barcode, c.Options)
		if err == nil {
			result.LookupTrail = attempts
			if result.NutritionCompleteness == "" {
				result.NutritionCompleteness = deriveNutritionCompleteness(result)
			}
			if result.ConfidenceScore == 0 {
				applyBarcodeConfidence(&result, DefaultVerifiedMinScore)
			}
			return result, nil
		}
		errs = append(errs, fmt.Sprintf("%s: %v", provider, err))
	}
	return BarcodeLookupResult{}, fmt.Errorf("lookup failed for %q across providers [%s]", barcode, strings.Join(errs, "; "))
}

func lookupBarcodeWithClient(db *sql.DB, provider string, client barcodeClient, barcode string) (BarcodeLookupResult, error) {
	barcode = strings.TrimSpace(barcode)
	if !isValidBarcode(barcode) {
		return BarcodeLookupResult{}, fmt.Errorf("invalid barcode %q (expected 8-14 digits)", barcode)
	}
	overridden, found, err := lookupBarcodeOverride(db, provider, barcode)
	if err != nil {
		return BarcodeLookupResult{}, err
	}
	if found {
		overridden.FromOverride = true
		overridden.SourceTier = "override"
		overridden.ExactMatch = true
		overridden.NutritionCompleteness = deriveNutritionCompleteness(overridden)
		applyBarcodeConfidence(&overridden, DefaultVerifiedMinScore)
		return overridden, nil
	}

	cached, found, err := lookupBarcodeCache(db, provider, barcode)
	if err != nil {
		return BarcodeLookupResult{}, err
	}
	if found {
		cached.FromCache = true
		cached.SourceTier = "cache"
		cached.ExactMatch = provider != BarcodeProviderUSDA
		cached.NutritionCompleteness = deriveNutritionCompleteness(cached)
		applyBarcodeConfidence(&cached, DefaultVerifiedMinScore)
		return cached, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	result, raw, err := client.LookupBarcode(ctx, barcode)
	if err != nil {
		return BarcodeLookupResult{}, err
	}
	result.Provider = provider
	result.Barcode = barcode
	result.SourceTier = "provider"
	if provider != BarcodeProviderUSDA {
		result.ExactMatch = true
	}
	result.NutritionCompleteness = deriveNutritionCompleteness(result)
	applyBarcodeConfidence(&result, DefaultVerifiedMinScore)
	if err := upsertBarcodeCache(db, result, raw, time.Now().Add(defaultBarcodeTTL)); err != nil {
		return BarcodeLookupResult{}, err
	}
	return result, nil
}

type usdaClientAdapter struct {
	client *usda.Client
}

func (a *usdaClientAdapter) LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error) {
	food, raw, err := a.client.LookupBarcode(ctx, barcode)
	if err != nil {
		return BarcodeLookupResult{}, nil, err
	}
	return BarcodeLookupResult{
		Description:    food.Description,
		Brand:          food.Brand,
		ServingAmount:  food.ServingAmount,
		ServingUnit:    food.ServingUnit,
		Calories:       food.Calories,
		ProteinG:       food.ProteinG,
		CarbsG:         food.CarbsG,
		FatG:           food.FatG,
		FiberG:         food.FiberG,
		SugarG:         food.SugarG,
		SodiumMg:       food.SodiumMg,
		Micronutrients: convertUSDAMicros(food.Micronutrients),
		SourceID:       food.FDCID,
		ExactMatch:     food.ExactBarcodeMatch,
	}, raw, nil
}

func (a *usdaClientAdapter) SearchFoods(ctx context.Context, query string, limit int) ([]BarcodeLookupResult, []byte, error) {
	foods, raw, err := a.client.SearchFoods(ctx, query, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]BarcodeLookupResult, 0, len(foods))
	for _, food := range foods {
		out = append(out, BarcodeLookupResult{
			Description:    food.Description,
			Brand:          food.Brand,
			ServingAmount:  food.ServingAmount,
			ServingUnit:    food.ServingUnit,
			Calories:       food.Calories,
			ProteinG:       food.ProteinG,
			CarbsG:         food.CarbsG,
			FatG:           food.FatG,
			FiberG:         food.FiberG,
			SugarG:         food.SugarG,
			SodiumMg:       food.SodiumMg,
			Micronutrients: convertUSDAMicros(food.Micronutrients),
			SourceID:       food.FDCID,
		})
	}
	return out, raw, nil
}

type openFoodFactsClientAdapter struct {
	client *openfoodfacts.Client
}

func (a *openFoodFactsClientAdapter) LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error) {
	food, raw, err := a.client.LookupBarcode(ctx, barcode)
	if err != nil {
		return BarcodeLookupResult{}, nil, err
	}
	return BarcodeLookupResult{
		Description:    food.Description,
		Brand:          food.Brand,
		ServingAmount:  food.ServingAmount,
		ServingUnit:    food.ServingUnit,
		Calories:       food.Calories,
		ProteinG:       food.ProteinG,
		CarbsG:         food.CarbsG,
		FatG:           food.FatG,
		FiberG:         food.FiberG,
		SugarG:         food.SugarG,
		SodiumMg:       food.SodiumMg,
		Micronutrients: convertOpenFoodFactsMicros(food.Micronutrients),
		SourceID:       food.SourceID,
		ExactMatch:     true,
	}, raw, nil
}

func (a *openFoodFactsClientAdapter) SearchFoods(ctx context.Context, query string, limit int) ([]BarcodeLookupResult, []byte, error) {
	foods, raw, err := a.client.SearchFoods(ctx, query, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]BarcodeLookupResult, 0, len(foods))
	for _, food := range foods {
		out = append(out, BarcodeLookupResult{
			Description:    food.Description,
			Brand:          food.Brand,
			ServingAmount:  food.ServingAmount,
			ServingUnit:    food.ServingUnit,
			Calories:       food.Calories,
			ProteinG:       food.ProteinG,
			CarbsG:         food.CarbsG,
			FatG:           food.FatG,
			FiberG:         food.FiberG,
			SugarG:         food.SugarG,
			SodiumMg:       food.SodiumMg,
			Micronutrients: convertOpenFoodFactsMicros(food.Micronutrients),
			SourceID:       food.SourceID,
		})
	}
	return out, raw, nil
}

type upcItemDBClientAdapter struct {
	client *upcitemdb.Client
}

func (a *upcItemDBClientAdapter) LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error) {
	food, raw, err := a.client.LookupBarcode(ctx, barcode)
	if err != nil {
		return BarcodeLookupResult{}, nil, err
	}
	return BarcodeLookupResult{
		Description:    food.Description,
		Brand:          food.Brand,
		ServingAmount:  food.ServingAmount,
		ServingUnit:    food.ServingUnit,
		Calories:       food.Calories,
		ProteinG:       food.ProteinG,
		CarbsG:         food.CarbsG,
		FatG:           food.FatG,
		FiberG:         food.FiberG,
		SugarG:         food.SugarG,
		SodiumMg:       food.SodiumMg,
		Micronutrients: convertUPCItemDBMicros(food.Micronutrients),
		SourceID:       food.SourceID,
		ExactMatch:     true,
	}, raw, nil
}

func (a *upcItemDBClientAdapter) SearchFoods(ctx context.Context, query string, limit int) ([]BarcodeLookupResult, []byte, error) {
	foods, raw, err := a.client.SearchFoods(ctx, query, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]BarcodeLookupResult, 0, len(foods))
	for _, food := range foods {
		out = append(out, BarcodeLookupResult{
			Description:    food.Description,
			Brand:          food.Brand,
			ServingAmount:  food.ServingAmount,
			ServingUnit:    food.ServingUnit,
			Calories:       food.Calories,
			ProteinG:       food.ProteinG,
			CarbsG:         food.CarbsG,
			FatG:           food.FatG,
			FiberG:         food.FiberG,
			SugarG:         food.SugarG,
			SodiumMg:       food.SodiumMg,
			Micronutrients: convertUPCItemDBMicros(food.Micronutrients),
			SourceID:       food.SourceID,
		})
	}
	return out, raw, nil
}

func isValidBarcode(code string) bool {
	return regexp.MustCompile(`^\d{8,14}$`).MatchString(code)
}

func lookupBarcodeCache(db *sql.DB, provider, barcode string) (BarcodeLookupResult, bool, error) {
	var row BarcodeLookupResult
	var expiresAtRaw string
	var microsRaw string
	err := db.QueryRow(`
SELECT provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, IFNULL(micronutrients_json,''), source_id, expires_at
FROM barcode_cache
WHERE provider = ? AND barcode = ?
`, provider, barcode).Scan(
		&row.Provider, &row.Barcode, &row.Description, &row.Brand,
		&row.ServingAmount, &row.ServingUnit,
		&row.Calories, &row.ProteinG, &row.CarbsG, &row.FatG, &row.FiberG, &row.SugarG, &row.SodiumMg, &microsRaw,
		&row.SourceID, &expiresAtRaw,
	)
	if err == sql.ErrNoRows {
		return BarcodeLookupResult{}, false, nil
	}
	if err != nil {
		return BarcodeLookupResult{}, false, fmt.Errorf("lookup barcode cache: %w", err)
	}
	micros, err := decodeMicronutrientsJSON(microsRaw)
	if err != nil {
		return BarcodeLookupResult{}, false, fmt.Errorf("decode barcode cache micronutrients: %w", err)
	}
	row.Micronutrients = micros
	expiresAt, err := time.Parse(time.RFC3339, expiresAtRaw)
	if err != nil {
		return BarcodeLookupResult{}, false, fmt.Errorf("parse barcode cache expiry: %w", err)
	}
	if time.Now().After(expiresAt) {
		return BarcodeLookupResult{}, false, nil
	}
	return row, true, nil
}

func upsertBarcodeCache(db *sql.DB, result BarcodeLookupResult, raw []byte, expiresAt time.Time) error {
	rawStr := ""
	if json.Valid(raw) {
		rawStr = string(raw)
	}
	microsJSON, err := EncodeMicronutrientsJSON(result.Micronutrients)
	if err != nil {
		return fmt.Errorf("encode barcode cache micronutrients: %w", err)
	}
	_, err = db.Exec(`
INSERT INTO barcode_cache(provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, micronutrients_json, source_id, raw_json, fetched_at, expires_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, barcode) DO UPDATE SET
  description=excluded.description,
  brand=excluded.brand,
  serving_amount=excluded.serving_amount,
  serving_unit=excluded.serving_unit,
  calories=excluded.calories,
  protein_g=excluded.protein_g,
  carbs_g=excluded.carbs_g,
  fat_g=excluded.fat_g,
  fiber_g=excluded.fiber_g,
  sugar_g=excluded.sugar_g,
  sodium_mg=excluded.sodium_mg,
  micronutrients_json=excluded.micronutrients_json,
  source_id=excluded.source_id,
  raw_json=excluded.raw_json,
  fetched_at=excluded.fetched_at,
  expires_at=excluded.expires_at
`, result.Provider, result.Barcode, result.Description, result.Brand, result.ServingAmount, result.ServingUnit, result.Calories, result.ProteinG, result.CarbsG, result.FatG, result.FiberG, result.SugarG, result.SodiumMg, microsJSON, result.SourceID, rawStr, time.Now().Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert barcode cache: %w", err)
	}
	return nil
}

func SetBarcodeOverride(db *sql.DB, provider, barcode string, in BarcodeOverrideInput) error {
	provider = normalizeBarcodeProvider(provider)
	barcode = strings.TrimSpace(barcode)
	if !isValidBarcode(barcode) {
		return fmt.Errorf("invalid barcode %q (expected 8-14 digits)", barcode)
	}
	if provider == "" {
		return fmt.Errorf("provider is required")
	}
	if strings.TrimSpace(in.Description) == "" {
		return fmt.Errorf("description is required")
	}
	if in.ServingAmount <= 0 {
		return fmt.Errorf("serving amount must be > 0")
	}
	if strings.TrimSpace(in.ServingUnit) == "" {
		return fmt.Errorf("serving unit is required")
	}
	if in.Calories < 0 || in.ProteinG < 0 || in.CarbsG < 0 || in.FatG < 0 || in.FiberG < 0 || in.SugarG < 0 || in.SodiumMg < 0 {
		return fmt.Errorf("calories and nutrients must be >= 0")
	}
	microsJSON, err := normalizeMicronutrientsJSON(in.Micronutrients)
	if err != nil {
		return err
	}

	_, err = db.Exec(`
INSERT INTO barcode_overrides(provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, micronutrients_json, notes, updated_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, barcode) DO UPDATE SET
  description=excluded.description,
  brand=excluded.brand,
  serving_amount=excluded.serving_amount,
  serving_unit=excluded.serving_unit,
  calories=excluded.calories,
  protein_g=excluded.protein_g,
  carbs_g=excluded.carbs_g,
  fat_g=excluded.fat_g,
  fiber_g=excluded.fiber_g,
  sugar_g=excluded.sugar_g,
  sodium_mg=excluded.sodium_mg,
  micronutrients_json=excluded.micronutrients_json,
  notes=excluded.notes,
  updated_at=excluded.updated_at
`, provider, barcode, strings.TrimSpace(in.Description), strings.TrimSpace(in.Brand), in.ServingAmount, strings.TrimSpace(in.ServingUnit), in.Calories, in.ProteinG, in.CarbsG, in.FatG, in.FiberG, in.SugarG, in.SodiumMg, microsJSON, strings.TrimSpace(in.Notes), time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("set barcode override: %w", err)
	}
	return nil
}

func GetBarcodeOverride(db *sql.DB, provider, barcode string) (BarcodeLookupResult, bool, error) {
	provider = normalizeBarcodeProvider(provider)
	return lookupBarcodeOverride(db, provider, strings.TrimSpace(barcode))
}

func DeleteBarcodeOverride(db *sql.DB, provider, barcode string) error {
	provider = normalizeBarcodeProvider(provider)
	barcode = strings.TrimSpace(barcode)
	res, err := db.Exec(`DELETE FROM barcode_overrides WHERE provider = ? AND barcode = ?`, provider, barcode)
	if err != nil {
		return fmt.Errorf("delete barcode override: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete barcode override rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("barcode override not found for provider=%s barcode=%s", provider, barcode)
	}
	return nil
}

func ListBarcodeOverrides(db *sql.DB, provider string, limit int) ([]BarcodeLookupResult, error) {
	provider = normalizeBarcodeProvider(provider)
	if limit <= 0 {
		limit = 100
	}
	base := `
SELECT provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, IFNULL(micronutrients_json,''), source_id
FROM barcode_overrides`
	args := make([]any, 0, 2)
	if provider != "" {
		base += ` WHERE provider = ?`
		args = append(args, provider)
	}
	base += ` ORDER BY updated_at DESC LIMIT ?`
	args = append(args, limit)
	rows, err := db.Query(base, args...)
	if err != nil {
		return nil, fmt.Errorf("list barcode overrides: %w", err)
	}
	defer rows.Close()
	out := make([]BarcodeLookupResult, 0)
	for rows.Next() {
		var r BarcodeLookupResult
		var microsRaw string
		if err := rows.Scan(&r.Provider, &r.Barcode, &r.Description, &r.Brand, &r.ServingAmount, &r.ServingUnit, &r.Calories, &r.ProteinG, &r.CarbsG, &r.FatG, &r.FiberG, &r.SugarG, &r.SodiumMg, &microsRaw, &r.SourceID); err != nil {
			return nil, fmt.Errorf("scan barcode override: %w", err)
		}
		micros, err := decodeMicronutrientsJSON(microsRaw)
		if err != nil {
			return nil, fmt.Errorf("decode barcode override micronutrients: %w", err)
		}
		r.Micronutrients = micros
		r.FromOverride = true
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate barcode overrides: %w", err)
	}
	return out, nil
}

func lookupBarcodeOverride(db *sql.DB, provider, barcode string) (BarcodeLookupResult, bool, error) {
	var row BarcodeLookupResult
	var microsRaw string
	err := db.QueryRow(`
SELECT provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, IFNULL(micronutrients_json,''), source_id
FROM barcode_overrides
WHERE provider = ? AND barcode = ?
`, provider, barcode).Scan(
		&row.Provider, &row.Barcode, &row.Description, &row.Brand,
		&row.ServingAmount, &row.ServingUnit,
		&row.Calories, &row.ProteinG, &row.CarbsG, &row.FatG, &row.FiberG, &row.SugarG, &row.SodiumMg, &microsRaw,
		&row.SourceID,
	)
	if err == sql.ErrNoRows {
		return BarcodeLookupResult{}, false, nil
	}
	if err != nil {
		return BarcodeLookupResult{}, false, fmt.Errorf("lookup barcode override: %w", err)
	}
	micros, err := decodeMicronutrientsJSON(microsRaw)
	if err != nil {
		return BarcodeLookupResult{}, false, fmt.Errorf("decode barcode override micronutrients: %w", err)
	}
	row.Micronutrients = micros
	return row, true, nil
}

func normalizeBarcodeProvider(provider string) string {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "off":
		return BarcodeProviderOpenFoodFacts
	default:
		return p
	}
}

func providerBaseConfidence(provider string) float64 {
	switch normalizeBarcodeProvider(provider) {
	case BarcodeProviderUSDA:
		return 0.90
	case BarcodeProviderOpenFoodFacts:
		return 0.72
	case BarcodeProviderUPCItemDB:
		return 0.68
	default:
		return 0.50
	}
}

func deriveNutritionCompleteness(r BarcodeLookupResult) string {
	if strings.TrimSpace(r.Description) == "" {
		return "unknown"
	}
	hasNutrition := r.Calories > 0 || r.ProteinG > 0 || r.CarbsG > 0 || r.FatG > 0 || r.FiberG > 0 || r.SugarG > 0 || r.SodiumMg > 0 || len(r.Micronutrients) > 0
	if r.ServingAmount > 0 && strings.TrimSpace(r.ServingUnit) != "" && hasNutrition {
		return "complete"
	}
	return "partial"
}

func convertUSDAMicros(in usda.Micronutrients) Micronutrients {
	out := Micronutrients{}
	for k, v := range in {
		out[k] = MicronutrientAmount{Value: v.Value, Unit: v.Unit}
	}
	return out
}

func convertOpenFoodFactsMicros(in openfoodfacts.Micronutrients) Micronutrients {
	out := Micronutrients{}
	for k, v := range in {
		out[k] = MicronutrientAmount{Value: v.Value, Unit: v.Unit}
	}
	return out
}

func convertUPCItemDBMicros(in upcitemdb.Micronutrients) Micronutrients {
	out := Micronutrients{}
	for k, v := range in {
		out[k] = MicronutrientAmount{Value: v.Value, Unit: v.Unit}
	}
	return out
}

func applyBarcodeConfidence(result *BarcodeLookupResult, minScore float64) {
	if result == nil {
		return
	}
	confidence := ScoreBarcodeConfidence(*result, minScore)
	result.ConfidenceScore = confidence.Score
	result.ProviderConfidence = confidence.Score
	result.IsVerified = confidence.IsVerified
	result.VerificationReasons = confidence.Reasons
}
