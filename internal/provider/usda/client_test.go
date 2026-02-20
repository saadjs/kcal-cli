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
