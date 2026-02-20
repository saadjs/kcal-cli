package openfoodfacts

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLookupBarcodeParsesOpenFoodFactsResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "status": 1,
  "product": {
    "product_name": "Yogurt Cup",
    "brands": "Brand Co",
    "serving_quantity": 170,
    "serving_quantity_unit": "g",
    "nutriments": {
      "energy-kcal_serving": 120,
      "proteins_serving": 10,
      "carbohydrates_serving": 15,
      "fat_serving": 2,
      "fiber_serving": 1.8,
      "sugars_serving": 12,
      "sodium_serving": 0.09,
      "vitamin-c_serving": 25
    }
  }
}`))
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, HTTPClient: ts.Client()}
	item, _, err := c.LookupBarcode(context.Background(), "12345678")
	if err != nil {
		t.Fatalf("lookup barcode: %v", err)
	}
	if item.Description != "Yogurt Cup" || item.Calories != 120 || item.ProteinG != 10 {
		t.Fatalf("unexpected parsed item: %+v", item)
	}
	if item.FiberG != 1.8 || item.SugarG != 12 || item.SodiumMg != 90 {
		t.Fatalf("unexpected extended nutrients: %+v", item)
	}
	if _, ok := item.Micronutrients["vitamin_c"]; !ok {
		t.Fatalf("expected micronutrient vitamin_c, got %+v", item.Micronutrients)
	}
}

func TestSearchFoodsParsesOpenFoodFactsResponse(t *testing.T) {
	t.Parallel()

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "products": [
    {
      "_id": "12345",
      "product_name": "Yogurt Vanilla",
      "brands": "Brand Co",
      "serving_quantity": 170,
      "serving_quantity_unit": "g",
      "nutriments": {
        "energy-kcal_serving": 130,
        "proteins_serving": 9,
        "carbohydrates_serving": 17,
        "fat_serving": 3
      }
    }
  ]
}`))
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, HTTPClient: ts.Client()}
	items, _, err := c.SearchFoods(context.Background(), "yogurt", 5)
	if err != nil {
		t.Fatalf("search foods: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 result, got %d", len(items))
	}
	if items[0].Description != "Yogurt Vanilla" || items[0].SourceID != 12345 {
		t.Fatalf("unexpected item: %+v", items[0])
	}
}
