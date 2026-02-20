package upcitemdb

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestLookupBarcodeParsesUPCItemDBResponse(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "code": "OK",
  "items": [
    {
      "title": "Test Cereal",
      "brand": "Test Brand",
      "size": "40 g",
      "nutrition_facts": {
        "Calories": "150",
        "Protein": "3g",
        "Total Carbohydrate": "30g",
        "Total Fat": "2g",
        "Dietary Fiber": "5g",
        "Total Sugars": "8g",
        "Sodium": "120mg",
        "Vitamin C": "12mg"
      }
    }
  ]
}`))
	}))
	defer ts.Close()

	c := &Client{BaseURL: ts.URL, HTTPClient: ts.Client()}
	item, _, err := c.LookupBarcode(context.Background(), "123456789012")
	if err != nil {
		t.Fatalf("lookup barcode: %v", err)
	}
	if item.Description != "Test Cereal" || item.Calories != 150 || item.ProteinG != 3 || item.CarbsG != 30 || item.FatG != 2 {
		t.Fatalf("unexpected parsed item: %+v", item)
	}
	if item.FiberG != 5 || item.SugarG != 8 || item.SodiumMg != 120 {
		t.Fatalf("unexpected extended nutrients: %+v", item)
	}
	if _, ok := item.Micronutrients["vitamin_c"]; !ok {
		t.Fatalf("expected micronutrient vitamin_c, got %+v", item.Micronutrients)
	}
}

func TestSearchFoodsParsesUPCItemDBResponse(t *testing.T) {
	t.Parallel()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
  "code": "OK",
  "items": [
    {
      "title": "Greek Yogurt Strawberry",
      "brand": "Test Brand",
      "upc": "123456789012",
      "size": "150 g",
      "nutrition_facts": {
        "Calories": "140",
        "Protein": "12g",
        "Total Carbohydrate": "15g",
        "Total Fat": "3g"
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
	if items[0].Description != "Greek Yogurt Strawberry" || items[0].SourceID != 123456789012 {
		t.Fatalf("unexpected item: %+v", items[0])
	}
}
