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
	calories := parseNutrient(item.NutritionFacts, "calories")
	protein := parseNutrient(item.NutritionFacts, "protein")
	carbs := parseNutrient(item.NutritionFacts, "carbohydrate")
	fat := parseNutrient(item.NutritionFacts, "fat")

	return FoodLookup{
		Description:   strings.TrimSpace(item.Title),
		Brand:         strings.TrimSpace(item.Brand),
		ServingAmount: amount,
		ServingUnit:   unit,
		Calories:      calories,
		ProteinG:      protein,
		CarbsG:        carbs,
		FatG:          fat,
		SourceID:      0,
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

func parseNutrient(n map[string]any, keyContains string) float64 {
	for k, v := range n {
		if strings.Contains(strings.ToLower(k), keyContains) {
			s := fmt.Sprintf("%v", v)
			var filtered strings.Builder
			for _, r := range s {
				if (r >= '0' && r <= '9') || r == '.' {
					filtered.WriteRune(r)
				}
			}
			if f, err := strconv.ParseFloat(filtered.String(), 64); err == nil {
				return f
			}
		}
	}
	return 0
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
