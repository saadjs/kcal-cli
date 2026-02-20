package usda

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLookupBarcodeParsesUSDAResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "foods": [
    {
      "fdcId": 12345,
      "description": "Greek Yogurt",
      "brandOwner": "Test Brand",
      "gtinUpc": "012345678905",
      "servingSize": 170,
      "servingSizeUnit": "g",
      "foodNutrients": [
        {"nutrientName": "Energy", "unitName": "KCAL", "value": 100},
        {"nutrientName": "Protein", "unitName": "G", "value": 17},
        {"nutrientName": "Carbohydrate, by difference", "unitName": "G", "value": 6},
        {"nutrientName": "Total lipid (fat)", "unitName": "G", "value": 0},
        {"nutrientName": "Fiber, total dietary", "unitName": "G", "value": 1.2},
        {"nutrientName": "Sugars, total including NLEA", "unitName": "G", "value": 4.1},
        {"nutrientName": "Sodium, Na", "unitName": "MG", "value": 55},
        {"nutrientName": "Vitamin C, total ascorbic acid", "unitName": "MG", "value": 6}
      ]
    }
  ]
}`))
	}))
	defer ts.Close()

	c := &Client{
		APIKey:     "demo",
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	}

	item, _, err := c.LookupBarcode(context.Background(), "012345678905")
	if err != nil {
		t.Fatalf("lookup barcode: %v", err)
	}
	if item.FDCID != 12345 {
		t.Fatalf("expected fdc id 12345, got %d", item.FDCID)
	}
	if !item.ExactBarcodeMatch {
		t.Fatalf("expected exact barcode match")
	}
	if item.Calories != 100 || item.ProteinG != 17 || item.CarbsG != 6 || item.FatG != 0 {
		t.Fatalf("unexpected nutrients: %+v", item)
	}
	if item.FiberG != 1.2 || item.SugarG != 4.1 || item.SodiumMg != 55 {
		t.Fatalf("unexpected extended nutrients: %+v", item)
	}
	if _, ok := item.Micronutrients["vitamin_c_total_ascorbic_acid"]; !ok {
		t.Fatalf("expected vitamin c micronutrient in %+v", item.Micronutrients)
	}
}

func TestLookupBarcodeMarksNonExactFallbackMatch(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "foods": [
    {
      "fdcId": 111,
      "description": "Fallback item",
      "brandOwner": "Brand",
      "gtinUpc": "999999999999",
      "servingSize": 100,
      "servingSizeUnit": "g",
      "foodNutrients": [
        {"nutrientName": "Energy", "unitName": "KCAL", "value": 100}
      ]
    }
  ]
}`))
	}))
	defer ts.Close()

	c := &Client{APIKey: "demo", BaseURL: ts.URL, HTTPClient: ts.Client()}
	item, _, err := c.LookupBarcode(context.Background(), "012345678905")
	if err != nil {
		t.Fatalf("lookup barcode: %v", err)
	}
	if item.ExactBarcodeMatch {
		t.Fatalf("expected non-exact fallback match")
	}
}

func TestSearchFoodsParsesUSDAResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "foods": [
    {
      "fdcId": 54321,
      "description": "Greek Yogurt Plain",
      "brandOwner": "Brand A",
      "servingSize": 150,
      "servingSizeUnit": "g",
      "foodNutrients": [
        {"nutrientName": "Energy", "unitName": "KCAL", "value": 120},
        {"nutrientName": "Protein", "unitName": "G", "value": 15},
        {"nutrientName": "Carbohydrate, by difference", "unitName": "G", "value": 8},
        {"nutrientName": "Total lipid (fat)", "unitName": "G", "value": 3}
      ]
    }
  ]
}`))
	}))
	defer ts.Close()

	c := &Client{
		APIKey:     "demo",
		BaseURL:    ts.URL,
		HTTPClient: ts.Client(),
	}
	items, _, err := c.SearchFoods(context.Background(), "yogurt", 5)
	if err != nil {
		t.Fatalf("search foods: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 result, got %d", len(items))
	}
	if items[0].Description != "Greek Yogurt Plain" || items[0].FDCID != 54321 {
		t.Fatalf("unexpected item: %+v", items[0])
	}
}
