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
	Description    string
	Brand          string
	ServingAmount  float64
	ServingUnit    string
	Calories       float64
	ProteinG       float64
	CarbsG         float64
	FatG           float64
	FiberG         float64
	SugarG         float64
	SodiumMg       float64
	Micronutrients Micronutrients
	SourceID       int64
}

type MicronutrientAmount struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Micronutrients map[string]MicronutrientAmount

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
	calories := nutrientValue(parsed.Product.Nutriments, "energy-kcal")
	protein := nutrientValue(parsed.Product.Nutriments, "proteins")
	carbs := nutrientValue(parsed.Product.Nutriments, "carbohydrates")
	fat := nutrientValue(parsed.Product.Nutriments, "fat")
	fiber := nutrientValue(parsed.Product.Nutriments, "fiber")
	sugar := nutrientValue(parsed.Product.Nutriments, "sugars")
	sodium := nutrientValue(parsed.Product.Nutriments, "sodium") * 1000
	micros := parseMicronutrients(parsed.Product.Nutriments)

	return FoodLookup{
		Description:    strings.TrimSpace(parsed.Product.ProductName),
		Brand:          strings.TrimSpace(parsed.Product.Brands),
		ServingAmount:  servingAmount,
		ServingUnit:    servingUnit,
		Calories:       calories,
		ProteinG:       protein,
		CarbsG:         carbs,
		FatG:           fat,
		FiberG:         fiber,
		SugarG:         sugar,
		SodiumMg:       sodium,
		Micronutrients: micros,
		SourceID:       0,
	}, body, nil
}

func nutrientValue(n map[string]any, base string) float64 {
	for _, key := range []string{base + "_serving", base + "_100g"} {
		if v, ok := parseFloatAny(n[key]); ok {
			return v
		}
	}
	return 0
}

func parseFloatAny(v any) (float64, bool) {
	switch t := v.(type) {
	case float64:
		return t, true
	case float32:
		return float64(t), true
	case int:
		return float64(t), true
	case int64:
		return float64(t), true
	case json.Number:
		f, err := t.Float64()
		return f, err == nil
	case string:
		f, err := strconv.ParseFloat(strings.TrimSpace(t), 64)
		return f, err == nil
	default:
		return 0, false
	}
}

func parseMicronutrients(n map[string]any) Micronutrients {
	out := Micronutrients{}
	for key, raw := range n {
		if !strings.HasSuffix(key, "_serving") && !strings.HasSuffix(key, "_100g") {
			continue
		}
		base := strings.TrimSuffix(strings.TrimSuffix(strings.ToLower(key), "_serving"), "_100g")
		if base == "energy-kcal" || base == "proteins" || base == "carbohydrates" || base == "fat" || base == "fiber" || base == "sugars" || base == "sodium" {
			continue
		}
		if !strings.Contains(base, "vitamin") && !strings.Contains(base, "iron") && !strings.Contains(base, "calcium") &&
			!strings.Contains(base, "potassium") && !strings.Contains(base, "magnesium") && !strings.Contains(base, "zinc") &&
			!strings.Contains(base, "phosphorus") && !strings.Contains(base, "selenium") && !strings.Contains(base, "copper") {
			continue
		}
		value, ok := parseFloatAny(raw)
		if !ok {
			continue
		}
		unit := micronutrientUnit(base)
		if unit == "" {
			continue
		}
		canonical := strings.ReplaceAll(base, "-", "_")
		out[canonical] = MicronutrientAmount{Value: value, Unit: unit}
	}
	return out
}

func micronutrientUnit(base string) string {
	switch {
	case strings.Contains(base, "vitamin-a"), strings.Contains(base, "vitamin-d"):
		return "ug"
	case strings.Contains(base, "vitamin-b12"), strings.Contains(base, "selenium"):
		return "ug"
	case strings.Contains(base, "sodium"), strings.Contains(base, "iron"), strings.Contains(base, "calcium"),
		strings.Contains(base, "potassium"), strings.Contains(base, "magnesium"), strings.Contains(base, "zinc"),
		strings.Contains(base, "phosphorus"), strings.Contains(base, "copper"), strings.Contains(base, "vitamin-c"):
		return "mg"
	default:
		return "mg"
	}
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
	ProductName         string         `json:"product_name"`
	Brands              string         `json:"brands"`
	ServingSize         string         `json:"serving_size"`
	ServingQuantity     float64        `json:"serving_quantity"`
	ServingQuantityUnit string         `json:"serving_quantity_unit"`
	Nutriments          map[string]any `json:"nutriments"`
}
