package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/provider/openfoodfacts"
	"github.com/saad/kcal-cli/internal/provider/usda"
)

const (
	BarcodeProviderUSDA          = "usda"
	BarcodeProviderOpenFoodFacts = "openfoodfacts"
	defaultBarcodeTTL            = 30 * 24 * time.Hour
)

type BarcodeLookupResult struct {
	Provider      string  `json:"provider"`
	Barcode       string  `json:"barcode"`
	Description   string  `json:"description"`
	Brand         string  `json:"brand"`
	ServingAmount float64 `json:"serving_amount"`
	ServingUnit   string  `json:"serving_unit"`
	Calories      float64 `json:"calories"`
	ProteinG      float64 `json:"protein_g"`
	CarbsG        float64 `json:"carbs_g"`
	FatG          float64 `json:"fat_g"`
	SourceID      int64   `json:"source_id"`
	FromCache     bool    `json:"from_cache"`
}

type barcodeClient interface {
	LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error)
}

func LookupBarcodeUSDA(db *sql.DB, apiKey, barcode string) (BarcodeLookupResult, error) {
	return LookupBarcode(db, BarcodeProviderUSDA, apiKey, barcode)
}

func LookupBarcodeOpenFoodFacts(db *sql.DB, barcode string) (BarcodeLookupResult, error) {
	return LookupBarcode(db, BarcodeProviderOpenFoodFacts, "", barcode)
}

func LookupBarcode(db *sql.DB, provider, apiKey, barcode string) (BarcodeLookupResult, error) {
	p := strings.ToLower(strings.TrimSpace(provider))
	switch p {
	case "", BarcodeProviderUSDA:
		return lookupBarcodeWithClient(db, BarcodeProviderUSDA, &usdaClientAdapter{client: &usda.Client{APIKey: apiKey}}, barcode)
	case BarcodeProviderOpenFoodFacts, "off":
		return lookupBarcodeWithClient(db, BarcodeProviderOpenFoodFacts, &openFoodFactsClientAdapter{client: &openfoodfacts.Client{}}, barcode)
	default:
		return BarcodeLookupResult{}, fmt.Errorf("unsupported barcode provider %q", provider)
	}
}

func lookupBarcodeWithClient(db *sql.DB, provider string, client barcodeClient, barcode string) (BarcodeLookupResult, error) {
	barcode = strings.TrimSpace(barcode)
	if !isValidBarcode(barcode) {
		return BarcodeLookupResult{}, fmt.Errorf("invalid barcode %q (expected 8-14 digits)", barcode)
	}

	cached, found, err := lookupBarcodeCache(db, provider, barcode)
	if err != nil {
		return BarcodeLookupResult{}, err
	}
	if found {
		cached.FromCache = true
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
		Description:   food.Description,
		Brand:         food.Brand,
		ServingAmount: food.ServingAmount,
		ServingUnit:   food.ServingUnit,
		Calories:      food.Calories,
		ProteinG:      food.ProteinG,
		CarbsG:        food.CarbsG,
		FatG:          food.FatG,
		SourceID:      food.FDCID,
	}, raw, nil
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
		Description:   food.Description,
		Brand:         food.Brand,
		ServingAmount: food.ServingAmount,
		ServingUnit:   food.ServingUnit,
		Calories:      food.Calories,
		ProteinG:      food.ProteinG,
		CarbsG:        food.CarbsG,
		FatG:          food.FatG,
		SourceID:      food.SourceID,
	}, raw, nil
}

func isValidBarcode(code string) bool {
	return regexp.MustCompile(`^\d{8,14}$`).MatchString(code)
}

func lookupBarcodeCache(db *sql.DB, provider, barcode string) (BarcodeLookupResult, bool, error) {
	var row BarcodeLookupResult
	var expiresAtRaw string
	err := db.QueryRow(`
SELECT provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, source_id, expires_at
FROM barcode_cache
WHERE provider = ? AND barcode = ?
`, provider, barcode).Scan(
		&row.Provider, &row.Barcode, &row.Description, &row.Brand,
		&row.ServingAmount, &row.ServingUnit,
		&row.Calories, &row.ProteinG, &row.CarbsG, &row.FatG,
		&row.SourceID, &expiresAtRaw,
	)
	if err == sql.ErrNoRows {
		return BarcodeLookupResult{}, false, nil
	}
	if err != nil {
		return BarcodeLookupResult{}, false, fmt.Errorf("lookup barcode cache: %w", err)
	}
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
	_, err := db.Exec(`
INSERT INTO barcode_cache(provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, source_id, raw_json, fetched_at, expires_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, barcode) DO UPDATE SET
  description=excluded.description,
  brand=excluded.brand,
  serving_amount=excluded.serving_amount,
  serving_unit=excluded.serving_unit,
  calories=excluded.calories,
  protein_g=excluded.protein_g,
  carbs_g=excluded.carbs_g,
  fat_g=excluded.fat_g,
  source_id=excluded.source_id,
  raw_json=excluded.raw_json,
  fetched_at=excluded.fetched_at,
  expires_at=excluded.expires_at
`, result.Provider, result.Barcode, result.Description, result.Brand, result.ServingAmount, result.ServingUnit, result.Calories, result.ProteinG, result.CarbsG, result.FatG, result.SourceID, rawStr, time.Now().Format(time.RFC3339), expiresAt.Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("upsert barcode cache: %w", err)
	}
	return nil
}
