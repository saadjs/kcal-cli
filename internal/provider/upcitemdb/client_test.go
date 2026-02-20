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
