package upcitemdb

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

const defaultBaseURL = "https://api.upcitemdb.com"

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
	APIKey     string
	APIKeyType string
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
	path := "/prod/trial/lookup"
	if strings.TrimSpace(c.APIKey) != "" {
		path = "/prod/v1/lookup"
	}
	url := fmt.Sprintf("%s%s?upc=%s", base, path, barcode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("create upcitemdb request: %w", err)
	}
	if strings.TrimSpace(c.APIKey) != "" {
		keyType := strings.TrimSpace(c.APIKeyType)
		if keyType == "" {
			keyType = "3scale"
		}
		req.Header.Set("key_type", keyType)
		req.Header.Set("user_key", strings.TrimSpace(c.APIKey))
	}
	req.Header.Set("Accept", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("execute upcitemdb request: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("read upcitemdb response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FoodLookup{}, body, fmt.Errorf("upcitemdb request failed with status %d", resp.StatusCode)
	}

	var parsed response
	if err := json.Unmarshal(body, &parsed); err != nil {
		return FoodLookup{}, body, fmt.Errorf("decode upcitemdb response: %w", err)
	}
	if strings.ToUpper(parsed.Code) != "OK" || len(parsed.Items) == 0 {
		return FoodLookup{}, body, fmt.Errorf("no upcitemdb product found for barcode %q", barcode)
	}
	item := parsed.Items[0]
	amount, unit := parseServing(item.Size)
	calories, _ := parseNutrient(item.NutritionFacts, "calories")
	protein, _ := parseNutrient(item.NutritionFacts, "protein")
	carbs, _ := parseNutrient(item.NutritionFacts, "carbohydrate")
	fat, _ := parseNutrient(item.NutritionFacts, "fat")
	fiber, _ := parseNutrient(item.NutritionFacts, "fiber")
	sugar, _ := parseNutrient(item.NutritionFacts, "sugar")
	sodium, sodiumUnit := parseNutrient(item.NutritionFacts, "sodium")
	if strings.ToLower(sodiumUnit) == "g" {
		sodium *= 1000
	}
	micros := parseMicronutrients(item.NutritionFacts)

	return FoodLookup{
		Description:    strings.TrimSpace(item.Title),
		Brand:          strings.TrimSpace(item.Brand),
		ServingAmount:  amount,
		ServingUnit:    unit,
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

func parseServing(size string) (float64, string) {
	size = strings.TrimSpace(size)
	if size == "" {
		return 100, "g"
	}
	parts := strings.Fields(size)
	if len(parts) >= 2 {
		if f, err := strconv.ParseFloat(strings.Trim(parts[0], ","), 64); err == nil && f > 0 {
			return f, parts[1]
		}
	}
	return 100, "g"
}

func parseNutrient(n map[string]any, keyContains string) (float64, string) {
	for k, v := range n {
		if strings.Contains(strings.ToLower(k), keyContains) {
			if amount, ok := parseNutrientAmount(fmt.Sprintf("%v", v)); ok {
				return amount.Value, amount.Unit
			}
		}
	}
	return 0, ""
}

func parseNutrientAmount(raw string) (MicronutrientAmount, bool) {
	raw = strings.ToLower(strings.TrimSpace(raw))
	if raw == "" {
		return MicronutrientAmount{}, false
	}
	var filtered strings.Builder
	for _, r := range raw {
		if (r >= '0' && r <= '9') || r == '.' {
			filtered.WriteRune(r)
		}
	}
	value, err := strconv.ParseFloat(filtered.String(), 64)
	if err != nil {
		return MicronutrientAmount{}, false
	}
	unit := "g"
	switch {
	case strings.Contains(raw, "mg"):
		unit = "mg"
	case strings.Contains(raw, "ug"), strings.Contains(raw, "mcg"):
		unit = "ug"
	case strings.Contains(raw, "iu"):
		unit = "iu"
	case strings.Contains(raw, "g"):
		unit = "g"
	}
	return MicronutrientAmount{Value: value, Unit: unit}, true
}

func parseMicronutrients(n map[string]any) Micronutrients {
	out := Micronutrients{}
	for k, v := range n {
		key := strings.ToLower(strings.TrimSpace(k))
		if strings.Contains(key, "calorie") || strings.Contains(key, "protein") || strings.Contains(key, "carbohydrate") ||
			strings.Contains(key, "fat") || strings.Contains(key, "fiber") || strings.Contains(key, "sugar") || strings.Contains(key, "sodium") {
			continue
		}
		if !strings.Contains(key, "vitamin") && !strings.Contains(key, "iron") && !strings.Contains(key, "calcium") &&
			!strings.Contains(key, "potassium") && !strings.Contains(key, "zinc") && !strings.Contains(key, "magnesium") {
			continue
		}
		amount, ok := parseNutrientAmount(fmt.Sprintf("%v", v))
		if !ok {
			continue
		}
		canonical := strings.ReplaceAll(strings.ReplaceAll(key, " ", "_"), "-", "_")
		out[canonical] = amount
	}
	return out
}

type response struct {
	Code  string `json:"code"`
	Items []item `json:"items"`
}

type item struct {
	Title          string         `json:"title"`
	Brand          string         `json:"brand"`
	Size           string         `json:"size"`
	NutritionFacts map[string]any `json:"nutrition_facts"`
}
