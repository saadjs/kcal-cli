package openfoodfacts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

const defaultBaseURL = "https://world.openfoodfacts.org"

type FoodLookup struct {
	Description   string
	Brand         string
	ServingAmount float64
	ServingUnit   string
	Calories      float64
	ProteinG      float64
	CarbsG        float64
	FatG          float64
	SourceID      int64
}

type Client struct {
	BaseURL    string
	HTTPClient *http.Client
}

func (c *Client) LookupBarcode(ctx context.Context, barcode string) (FoodLookup, []byte, error) {
	base := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if base == "" {
		base = defaultBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 12 * time.Second}
	}
	url := fmt.Sprintf("%s/api/v2/product/%s.json", base, barcode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("create openfoodfacts request: %w", err)
	}
	req.Header.Set("User-Agent", "kcal-cli/1.0 (+https://github.com/saad/kcal-cli)")

	resp, err := httpClient.Do(req)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("execute openfoodfacts request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("read openfoodfacts response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FoodLookup{}, body, fmt.Errorf("openfoodfacts request failed with status %d", resp.StatusCode)
	}

	var parsed offResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return FoodLookup{}, body, fmt.Errorf("decode openfoodfacts response: %w", err)
	}
	if parsed.Status != 1 || parsed.Product.ProductName == "" {
		return FoodLookup{}, body, fmt.Errorf("no openfoodfacts product found for barcode %q", barcode)
	}

	servingAmount, servingUnit := parseServing(parsed.Product)
	calories := firstNonZero(parsed.Product.Nutriments.EnergyKcalServing, parsed.Product.Nutriments.EnergyKcal100g)
	protein := firstNonZero(parsed.Product.Nutriments.ProteinServing, parsed.Product.Nutriments.Protein100g)
	carbs := firstNonZero(parsed.Product.Nutriments.CarbsServing, parsed.Product.Nutriments.Carbs100g)
	fat := firstNonZero(parsed.Product.Nutriments.FatServing, parsed.Product.Nutriments.Fat100g)

	return FoodLookup{
		Description:   strings.TrimSpace(parsed.Product.ProductName),
		Brand:         strings.TrimSpace(parsed.Product.Brands),
		ServingAmount: servingAmount,
		ServingUnit:   servingUnit,
		Calories:      calories,
		ProteinG:      protein,
		CarbsG:        carbs,
		FatG:          fat,
		SourceID:      0,
	}, body, nil
}

func firstNonZero(a, b float64) float64 {
	if a != 0 {
		return a
	}
	return b
}

func parseServing(p offProduct) (float64, string) {
	if p.ServingQuantity > 0 {
		unit := strings.TrimSpace(p.ServingQuantityUnit)
		if unit == "" {
			unit = "g"
		}
		return p.ServingQuantity, unit
	}
	if strings.TrimSpace(p.ServingSize) != "" {
		parts := strings.Fields(strings.TrimSpace(p.ServingSize))
		if len(parts) >= 2 {
			if val, err := strconv.ParseFloat(strings.ReplaceAll(parts[0], ",", ""), 64); err == nil && val > 0 {
				return val, parts[1]
			}
		}
	}
	return 100, "g"
}

type offResponse struct {
	Status  int        `json:"status"`
	Product offProduct `json:"product"`
}

type offProduct struct {
	ProductName         string        `json:"product_name"`
	Brands              string        `json:"brands"`
	ServingSize         string        `json:"serving_size"`
	ServingQuantity     float64       `json:"serving_quantity"`
	ServingQuantityUnit string        `json:"serving_quantity_unit"`
	Nutriments          offNutriments `json:"nutriments"`
}

type offNutriments struct {
	EnergyKcalServing float64 `json:"energy-kcal_serving"`
	EnergyKcal100g    float64 `json:"energy-kcal_100g"`
	ProteinServing    float64 `json:"proteins_serving"`
	Protein100g       float64 `json:"proteins_100g"`
	CarbsServing      float64 `json:"carbohydrates_serving"`
	Carbs100g         float64 `json:"carbohydrates_100g"`
	FatServing        float64 `json:"fat_serving"`
	Fat100g           float64 `json:"fat_100g"`
}
