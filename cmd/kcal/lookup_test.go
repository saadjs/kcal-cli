package kcal

import (
	"os"
	"strings"
	"testing"
)

func TestResolveUSDAAPIKeyPriority(t *testing.T) {
	t.Setenv("KCAL_USDA_API_KEY", "usda-env")
	t.Setenv("KCAL_BARCODE_API_KEY", "legacy-env")

	if got := resolveUSDAAPIKey("flag"); got != "flag" {
		t.Fatalf("expected flag value to win, got %q", got)
	}
	if got := resolveUSDAAPIKey(""); got != "usda-env" {
		t.Fatalf("expected KCAL_USDA_API_KEY fallback, got %q", got)
	}

	_ = os.Unsetenv("KCAL_USDA_API_KEY")
	if got := resolveUSDAAPIKey(""); got != "legacy-env" {
		t.Fatalf("expected KCAL_BARCODE_API_KEY fallback, got %q", got)
	}
}

func TestUSDAHelpTextIncludesSetupAndRateLimit(t *testing.T) {
	out := usdaHelpText()
	if out == "" {
		t.Fatalf("expected non-empty USDA help text")
	}
	if !containsAll(out, []string{
		"api.data.gov/signup",
		"fdc.nal.usda.gov/api-guide",
		"KCAL_USDA_API_KEY",
		"1,000 requests per hour per IP",
	}) {
		t.Fatalf("USDA help text missing expected guidance: %s", out)
	}
}

func containsAll(s string, parts []string) bool {
	for _, p := range parts {
		if !strings.Contains(s, p) {
			return false
		}
	}
	return true
}
