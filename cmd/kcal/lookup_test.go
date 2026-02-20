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

func TestResolveBarcodeProvider(t *testing.T) {
	t.Setenv("KCAL_BARCODE_PROVIDER", "")
	if got := resolveBarcodeProvider(""); got != "usda" {
		t.Fatalf("expected default provider usda, got %q", got)
	}
	t.Setenv("KCAL_BARCODE_PROVIDER", "openfoodfacts")
	if got := resolveBarcodeProvider(""); got != "openfoodfacts" {
		t.Fatalf("expected env provider openfoodfacts, got %q", got)
	}
	if got := resolveBarcodeProvider("usda"); got != "usda" {
		t.Fatalf("expected flag provider usda, got %q", got)
	}
}

func TestOpenFoodFactsHelpTextIncludesDocsAndRateLimit(t *testing.T) {
	out := openFoodFactsHelpText()
	if !containsAll(out, []string{
		"openfoodfacts.github.io",
		"not required",
		"KCAL_BARCODE_PROVIDER=openfoodfacts",
		"fair-use limits",
	}) {
		t.Fatalf("openfoodfacts help text missing expected guidance: %s", out)
	}
}

func TestUPCItemDBHelpTextIncludesPlanLimits(t *testing.T) {
	out := upcItemDBHelpText()
	if !containsAll(out, []string{
		"devs.upcitemdb.com",
		"100 requests/day",
		"20,000 lookup/day",
		"150,000 lookup/day",
	}) {
		t.Fatalf("upcitemdb help text missing expected plan limit guidance: %s", out)
	}
}

func TestResolveUPCItemDBKeyType(t *testing.T) {
	t.Setenv("KCAL_UPCITEMDB_KEY_TYPE", "")
	if got := resolveProviderAPIKeyType("upcitemdb", ""); got != "3scale" {
		t.Fatalf("expected default key type 3scale, got %q", got)
	}
	t.Setenv("KCAL_UPCITEMDB_KEY_TYPE", "apikey")
	if got := resolveProviderAPIKeyType("upcitemdb", ""); got != "apikey" {
		t.Fatalf("expected env key type apikey, got %q", got)
	}
	if got := resolveProviderAPIKeyType("upcitemdb", "custom"); got != "custom" {
		t.Fatalf("expected flag key type custom, got %q", got)
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
