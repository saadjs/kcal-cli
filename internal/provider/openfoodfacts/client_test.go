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
      "fat_serving": 2
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
}
