package service

import (
	"testing"
	"time"
)

func TestDedupeAndRankFoodSearchKeepsBestAndAlternatives(t *testing.T) {
	items := []FoodSearchResult{
		{Provider: BarcodeProviderOpenFoodFacts, Description: "Greek Yogurt", Brand: "Fage", ConfidenceScore: 0.82, NutritionCompleteness: "complete"},
		{Provider: BarcodeProviderUSDA, Description: "Greek Yogurt", Brand: "Fage", ConfidenceScore: 0.91, NutritionCompleteness: "complete"},
		{Provider: BarcodeProviderUPCItemDB, Description: "Skyr Yogurt", Brand: "Siggi", ConfidenceScore: 0.70, NutritionCompleteness: "partial"},
	}
	out := dedupeAndRankFoodSearch(items, []string{BarcodeProviderUSDA, BarcodeProviderOpenFoodFacts, BarcodeProviderUPCItemDB})
	if len(out) != 2 {
		t.Fatalf("expected 2 deduped results, got %d", len(out))
	}
	if out[0].Provider != BarcodeProviderUSDA {
		t.Fatalf("expected USDA result ranked first, got %+v", out[0])
	}
	if len(out[0].Alternatives) != 1 {
		t.Fatalf("expected one alternative in merged group, got %d", len(out[0].Alternatives))
	}
}

func TestProviderSearchCacheRoundTrip(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	items := []BarcodeLookupResult{{Provider: BarcodeProviderUSDA, Description: "Greek Yogurt", Brand: "Fage", Calories: 120}}
	err := upsertProviderSearchCache(sqldb, BarcodeProviderUSDA, "greek yogurt", 10, items, nil, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("upsert provider search cache: %v", err)
	}
	got, found, err := lookupProviderSearchCache(sqldb, BarcodeProviderUSDA, "greek yogurt", 10)
	if err != nil {
		t.Fatalf("lookup provider search cache: %v", err)
	}
	if !found {
		t.Fatalf("expected cache hit")
	}
	if len(got) != 1 || got[0].Description != "Greek Yogurt" {
		t.Fatalf("unexpected cache payload: %+v", got)
	}
}

func TestProviderSearchCacheExpires(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	items := []BarcodeLookupResult{{Provider: BarcodeProviderUSDA, Description: "Greek Yogurt"}}
	err := upsertProviderSearchCache(sqldb, BarcodeProviderUSDA, "greek yogurt", 10, items, nil, time.Now().Add(-time.Minute))
	if err != nil {
		t.Fatalf("upsert provider search cache: %v", err)
	}
	_, found, err := lookupProviderSearchCache(sqldb, BarcodeProviderUSDA, "greek yogurt", 10)
	if err != nil {
		t.Fatalf("lookup provider search cache: %v", err)
	}
	if found {
		t.Fatalf("expected expired cache miss")
	}
}

func TestListAndPurgeProviderSearchCache(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	items := []BarcodeLookupResult{{Provider: BarcodeProviderOpenFoodFacts, Description: "Greek Yogurt"}}
	if err := upsertProviderSearchCache(sqldb, BarcodeProviderOpenFoodFacts, "greek yogurt", 10, items, nil, time.Now().Add(time.Hour)); err != nil {
		t.Fatalf("upsert provider search cache: %v", err)
	}
	list, err := ListProviderSearchCache(sqldb, BarcodeProviderOpenFoodFacts, "greek yogurt", 10)
	if err != nil {
		t.Fatalf("list provider search cache: %v", err)
	}
	if len(list) != 1 {
		t.Fatalf("expected one provider search cache row, got %d", len(list))
	}
	removed, err := PurgeProviderSearchCache(sqldb, BarcodeProviderOpenFoodFacts, "greek yogurt", false)
	if err != nil {
		t.Fatalf("purge provider search cache: %v", err)
	}
	if removed != 1 {
		t.Fatalf("expected one deleted row, got %d", removed)
	}
}
