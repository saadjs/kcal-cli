package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

var micronutrientKeyPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

type MicronutrientAmount struct {
	Value float64 `json:"value"`
	Unit  string  `json:"unit"`
}

type Micronutrients map[string]MicronutrientAmount

func ParseMicronutrientsJSON(value string) (Micronutrients, error) {
	return decodeMicronutrientsJSON(value)
}

func NormalizeMicronutrientsJSON(value string) (string, error) {
	return normalizeMicronutrientsJSON(value)
}

func EncodeMicronutrientsJSON(m Micronutrients) (string, error) {
	if len(m) == 0 {
		return "", nil
	}
	normalized, err := json.Marshal(m)
	if err != nil {
		return "", fmt.Errorf("marshal micronutrients: %w", err)
	}
	return string(normalized), nil
}

func MergeMicronutrients(base, override Micronutrients) Micronutrients {
	return mergeMicronutrients(base, override)
}

func ScaleMicronutrients(src Micronutrients, factor float64) Micronutrients {
	return scaleMicronutrients(src, factor)
}

func normalizeMicronutrientsJSON(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	parsed, err := parseMicronutrients(value)
	if err != nil {
		return "", err
	}
	if len(parsed) == 0 {
		return "", nil
	}
	normalized, err := json.Marshal(parsed)
	if err != nil {
		return "", fmt.Errorf("marshal micronutrients: %w", err)
	}
	return string(normalized), nil
}

func parseMicronutrients(value string) (Micronutrients, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Micronutrients{}, nil
	}
	var decoded map[string]MicronutrientAmount
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return nil, fmt.Errorf("micronutrients must be a valid JSON object: %w", err)
	}
	normalized := Micronutrients{}
	for rawKey, amount := range decoded {
		key := normalizeMicronutrientKey(rawKey)
		if key == "" || !micronutrientKeyPattern.MatchString(key) {
			return nil, fmt.Errorf("invalid micronutrient key %q (expected lowercase snake_case)", rawKey)
		}
		if amount.Value < 0 {
			return nil, fmt.Errorf("micronutrient %q value must be >= 0", rawKey)
		}
		amount.Unit = strings.TrimSpace(amount.Unit)
		if amount.Unit == "" {
			return nil, fmt.Errorf("micronutrient %q unit is required", rawKey)
		}
		normalized[key] = amount
	}
	return normalized, nil
}

func decodeMicronutrientsJSON(value string) (Micronutrients, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return Micronutrients{}, nil
	}
	return parseMicronutrients(value)
}

func mergeMicronutrients(base, override Micronutrients) Micronutrients {
	out := Micronutrients{}
	for k, v := range base {
		out[k] = v
	}
	for k, v := range override {
		out[k] = v
	}
	return out
}

func scaleMicronutrients(src Micronutrients, factor float64) Micronutrients {
	out := Micronutrients{}
	for k, v := range src {
		out[k] = MicronutrientAmount{
			Value: v.Value * factor,
			Unit:  v.Unit,
		}
	}
	return out
}

func normalizeMicronutrientKey(raw string) string {
	k := strings.TrimSpace(strings.ToLower(raw))
	k = strings.ReplaceAll(k, "-", "_")
	k = strings.ReplaceAll(k, " ", "_")
	k = strings.Trim(k, "_")
	for strings.Contains(k, "__") {
		k = strings.ReplaceAll(k, "__", "_")
	}
	return k
}
