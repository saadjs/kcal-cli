package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/saadjs/kcal-cli/internal/db"
)

type fakeBarcodeClient struct {
	calls int
	item  BarcodeLookupResult
}

func (f *fakeBarcodeClient) LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error) {
	_ = ctx
	_ = barcode
	f.calls++
	return f.item, []byte(`{"ok":true}`), nil
}

func TestLookupBarcodeUsesCache(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	client := &fakeBarcodeClient{item: BarcodeLookupResult{
		Description:   "Protein Bar",
		Brand:         "Brand",
		ServingAmount: 1,
		ServingUnit:   "bar",
		Calories:      200,
		ProteinG:      20,
		CarbsG:        20,
		FatG:          7,
		SourceID:      111,
	}}

	_, err := lookupBarcodeWithClient(sqldb, BarcodeProviderUSDA, client, "012345678905")
	if err != nil {
		t.Fatalf("first lookup: %v", err)
	}
	_, err = lookupBarcodeWithClient(sqldb, BarcodeProviderUSDA, client, "012345678905")
	if err != nil {
		t.Fatalf("second lookup: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("expected 1 provider call due to cache hit, got %d", client.calls)
	}
}

func TestLookupBarcodeValidation(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	client := &fakeBarcodeClient{}
	_, err := lookupBarcodeWithClient(sqldb, BarcodeProviderUSDA, client, "abc")
	if err == nil {
		t.Fatalf("expected invalid barcode to fail")
	}
}

func TestLookupBarcodePrefersOverrideOverProvider(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	if err := SetBarcodeOverride(sqldb, BarcodeProviderUSDA, "012345678905", BarcodeOverrideInput{
		Description:   "Override Food",
		Brand:         "Local",
		ServingAmount: 1,
		ServingUnit:   "bar",
		Calories:      123,
		ProteinG:      9,
		CarbsG:        10,
		FatG:          2,
	}); err != nil {
		t.Fatalf("set override: %v", err)
	}

	client := &fakeBarcodeClient{item: BarcodeLookupResult{
		Description:   "Provider Food",
		Brand:         "Remote",
		ServingAmount: 1,
		ServingUnit:   "bar",
		Calories:      999,
	}}
	got, err := lookupBarcodeWithClient(sqldb, BarcodeProviderUSDA, client, "012345678905")
	if err != nil {
		t.Fatalf("lookup with override: %v", err)
	}
	if client.calls != 0 {
		t.Fatalf("expected provider to not be called when override exists")
	}
	if !got.FromOverride || got.Calories != 123 {
		t.Fatalf("expected override result, got %+v", got)
	}
}

func TestBarcodeOverrideLifecycle(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	if err := SetBarcodeOverride(sqldb, "off", "3017620422003", BarcodeOverrideInput{
		Description:   "Nutella Custom",
		Brand:         "Ferrero",
		ServingAmount: 15,
		ServingUnit:   "g",
		Calories:      81,
		ProteinG:      1,
		CarbsG:        9,
		FatG:          5,
		Notes:         "custom",
	}); err != nil {
		t.Fatalf("set override: %v", err)
	}

	item, found, err := GetBarcodeOverride(sqldb, BarcodeProviderOpenFoodFacts, "3017620422003")
	if err != nil {
		t.Fatalf("get override: %v", err)
	}
	if !found || item.Description != "Nutella Custom" {
		t.Fatalf("expected override item, got found=%t item=%+v", found, item)
	}

	list, err := ListBarcodeOverrides(sqldb, BarcodeProviderOpenFoodFacts, 10)
	if err != nil {
		t.Fatalf("list overrides: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected 1 override, got %d", len(list))
	}

	if err := DeleteBarcodeOverride(sqldb, BarcodeProviderOpenFoodFacts, "3017620422003"); err != nil {
		t.Fatalf("delete override: %v", err)
	}
	_, found, err = GetBarcodeOverride(sqldb, BarcodeProviderOpenFoodFacts, "3017620422003")
	if err != nil {
		t.Fatalf("get after delete: %v", err)
	}
	if found {
		t.Fatalf("expected override to be deleted")
	}
}

func TestLookupBarcodeWithFallbackUsesNextProvider(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	const code = "3017620422003"
	if err := SetBarcodeOverride(sqldb, BarcodeProviderOpenFoodFacts, code, BarcodeOverrideInput{
		Description:   "Fallback Food",
		Brand:         "Local",
		ServingAmount: 15,
		ServingUnit:   "g",
		Calories:      100,
		ProteinG:      1,
		CarbsG:        10,
		FatG:          6,
	}); err != nil {
		t.Fatalf("set override: %v", err)
	}

	got, err := LookupBarcodeWithFallback(sqldb, code, []BarcodeLookupCandidate{
		{Provider: BarcodeProviderUSDA, Options: BarcodeLookupOptions{}},
		{Provider: BarcodeProviderOpenFoodFacts, Options: BarcodeLookupOptions{}},
	})
	if err != nil {
		t.Fatalf("lookup fallback: %v", err)
	}
	if got.Provider != BarcodeProviderOpenFoodFacts {
		t.Fatalf("expected openfoodfacts result, got %q", got.Provider)
	}
	if !got.FromOverride {
		t.Fatalf("expected override-backed result, got %+v", got)
	}
	if got.ProviderConfidence != 1.0 {
		t.Fatalf("expected confidence 1.0 for override, got %.2f", got.ProviderConfidence)
	}
	if got.NutritionCompleteness != "complete" {
		t.Fatalf("expected completeness complete, got %q", got.NutritionCompleteness)
	}
	if len(got.LookupTrail) != 2 || got.LookupTrail[0] != BarcodeProviderUSDA || got.LookupTrail[1] != BarcodeProviderOpenFoodFacts {
		t.Fatalf("unexpected lookup trail: %+v", got.LookupTrail)
	}
}

func TestDeriveNutritionCompleteness(t *testing.T) {
	if got := deriveNutritionCompleteness(BarcodeLookupResult{}); got != "unknown" {
		t.Fatalf("expected unknown completeness, got %q", got)
	}
	if got := deriveNutritionCompleteness(BarcodeLookupResult{Description: "Food"}); got != "partial" {
		t.Fatalf("expected partial completeness, got %q", got)
	}
	if got := deriveNutritionCompleteness(BarcodeLookupResult{Description: "Food", ServingAmount: 10, ServingUnit: "g", ProteinG: 1}); got != "complete" {
		t.Fatalf("expected complete completeness, got %q", got)
	}
}

func newServiceDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kcal.db")
	sqldb, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.ApplyMigrations(sqldb); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return sqldb
}
