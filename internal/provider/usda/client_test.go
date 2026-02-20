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
        {"nutrientName": "Energy", "value": 100},
        {"nutrientName": "Protein", "value": 17},
        {"nutrientName": "Carbohydrate, by difference", "value": 6},
        {"nutrientName": "Total lipid (fat)", "value": 0}
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
	if item.Calories != 100 || item.ProteinG != 17 || item.CarbsG != 6 || item.FatG != 0 {
		t.Fatalf("unexpected nutrients: %+v", item)
	}
}
