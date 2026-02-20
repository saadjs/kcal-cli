package usda

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.nal.usda.gov"

type FoodLookup struct {
	Barcode       string  `json:"barcode"`
	Description   string  `json:"description"`
	Brand         string  `json:"brand"`
	ServingAmount float64 `json:"serving_amount"`
	ServingUnit   string  `json:"serving_unit"`
	Calories      float64 `json:"calories"`
	ProteinG      float64 `json:"protein_g"`
	CarbsG        float64 `json:"carbs_g"`
	FatG          float64 `json:"fat_g"`
	FDCID         int64   `json:"fdc_id"`
}

type Client struct {
	APIKey     string
	BaseURL    string
	HTTPClient *http.Client
}

func (c *Client) LookupBarcode(ctx context.Context, barcode string) (FoodLookup, []byte, error) {
	if strings.TrimSpace(c.APIKey) == "" {
		return FoodLookup{}, nil, fmt.Errorf("missing USDA API key")
	}
	baseURL := strings.TrimRight(strings.TrimSpace(c.BaseURL), "/")
	if baseURL == "" {
		baseURL = defaultBaseURL
	}
	httpClient := c.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 12 * time.Second}
	}

	reqBody := map[string]any{
		"query":    barcode,
		"dataType": []string{"Branded"},
		"pageSize": 20,
	}
	payload, err := json.Marshal(reqBody)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("marshal USDA search payload: %w", err)
	}

	url := fmt.Sprintf("%s/fdc/v1/foods/search?api_key=%s", baseURL, c.APIKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("create USDA request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := httpClient.Do(req)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("execute USDA request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return FoodLookup{}, nil, fmt.Errorf("read USDA response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return FoodLookup{}, body, fmt.Errorf("USDA request failed with status %d", resp.StatusCode)
	}

	var parsed searchResponse
	if err := json.Unmarshal(body, &parsed); err != nil {
		return FoodLookup{}, body, fmt.Errorf("decode USDA response: %w", err)
	}

	food, ok := selectBarcodeMatch(parsed.Foods, barcode)
	if !ok {
		return FoodLookup{}, body, fmt.Errorf("no USDA branded food found for barcode %q", barcode)
	}

	out := FoodLookup{
		Barcode:       barcode,
		Description:   strings.TrimSpace(food.Description),
		Brand:         strings.TrimSpace(food.BrandOwner),
		ServingAmount: food.ServingSize,
		ServingUnit:   strings.TrimSpace(food.ServingSizeUnit),
		FDCID:         food.FDCID,
	}
	for _, n := range food.FoodNutrients {
		switch strings.ToLower(strings.TrimSpace(n.NutrientName)) {
		case "energy":
			out.Calories = n.Value
		case "protein":
			out.ProteinG = n.Value
		case "carbohydrate, by difference":
			out.CarbsG = n.Value
		case "total lipid (fat)":
			out.FatG = n.Value
		}
	}

	return out, body, nil
}

func selectBarcodeMatch(foods []usdaFood, barcode string) (usdaFood, bool) {
	for _, f := range foods {
		if strings.TrimSpace(f.GTINUPC) == barcode {
			return f, true
		}
	}
	if len(foods) > 0 {
		return foods[0], true
	}
	return usdaFood{}, false
}

type searchResponse struct {
	Foods []usdaFood `json:"foods"`
}

type usdaFood struct {
	FDCID           int64          `json:"fdcId"`
	Description     string         `json:"description"`
	BrandOwner      string         `json:"brandOwner"`
	GTINUPC         string         `json:"gtinUpc"`
	ServingSize     float64        `json:"servingSize"`
	ServingSizeUnit string         `json:"servingSizeUnit"`
	FoodNutrients   []usdaNutrient `json:"foodNutrients"`
}

type usdaNutrient struct {
	NutrientName string  `json:"nutrientName"`
	Value        float64 `json:"value"`
}
