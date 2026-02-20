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
	"github.com/saad/kcal-cli/internal/provider/upcitemdb"
	"github.com/saad/kcal-cli/internal/provider/usda"
)

const (
	BarcodeProviderUSDA          = "usda"
	BarcodeProviderOpenFoodFacts = "openfoodfacts"
	BarcodeProviderUPCItemDB     = "upcitemdb"
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
	FromOverride  bool    `json:"from_override"`
	FromCache     bool    `json:"from_cache"`
}

type BarcodeOverrideInput struct {
	Description   string
	Brand         string
	ServingAmount float64
	ServingUnit   string
	Calories      float64
	ProteinG      float64
	CarbsG        float64
	FatG          float64
	Notes         string
}

type BarcodeLookupOptions struct {
	APIKey     string
	APIKeyType string
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
		return overridden, nil
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

type upcItemDBClientAdapter struct {
	client *upcitemdb.Client
}

func (a *upcItemDBClientAdapter) LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error) {
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
	if in.Calories < 0 || in.ProteinG < 0 || in.CarbsG < 0 || in.FatG < 0 {
		return fmt.Errorf("calories and macros must be >= 0")
	}

	_, err := db.Exec(`
INSERT INTO barcode_overrides(provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, notes, updated_at)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(provider, barcode) DO UPDATE SET
  description=excluded.description,
  brand=excluded.brand,
  serving_amount=excluded.serving_amount,
  serving_unit=excluded.serving_unit,
  calories=excluded.calories,
  protein_g=excluded.protein_g,
  carbs_g=excluded.carbs_g,
  fat_g=excluded.fat_g,
  notes=excluded.notes,
  updated_at=excluded.updated_at
`, provider, barcode, strings.TrimSpace(in.Description), strings.TrimSpace(in.Brand), in.ServingAmount, strings.TrimSpace(in.ServingUnit), in.Calories, in.ProteinG, in.CarbsG, in.FatG, strings.TrimSpace(in.Notes), time.Now().Format(time.RFC3339))
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
SELECT provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, source_id
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
		if err := rows.Scan(&r.Provider, &r.Barcode, &r.Description, &r.Brand, &r.ServingAmount, &r.ServingUnit, &r.Calories, &r.ProteinG, &r.CarbsG, &r.FatG, &r.SourceID); err != nil {
			return nil, fmt.Errorf("scan barcode override: %w", err)
		}
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
	err := db.QueryRow(`
SELECT provider, barcode, description, brand, serving_amount, serving_unit, calories, protein_g, carbs_g, fat_g, source_id
FROM barcode_overrides
WHERE provider = ? AND barcode = ?
`, provider, barcode).Scan(
		&row.Provider, &row.Barcode, &row.Description, &row.Brand,
		&row.ServingAmount, &row.ServingUnit,
		&row.Calories, &row.ProteinG, &row.CarbsG, &row.FatG,
		&row.SourceID,
	)
	if err == sql.ErrNoRows {
		return BarcodeLookupResult{}, false, nil
	}
	if err != nil {
		return BarcodeLookupResult{}, false, fmt.Errorf("lookup barcode override: %w", err)
	}
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
