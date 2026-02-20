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

func TestParseProviderOrder(t *testing.T) {
	got := parseProviderOrder("usda,off,upcitemdb,openfoodfacts,unknown,usda")
	want := []string{"usda", "openfoodfacts", "upcitemdb"}
	if len(got) != len(want) {
		t.Fatalf("unexpected parsed length: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected provider at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestResolveFallbackProvidersPrependsPrimaryProvider(t *testing.T) {
	got := resolveFallbackProviders("openfoodfacts", "usda,upcitemdb")
	want := []string{"openfoodfacts", "usda", "upcitemdb"}
	if len(got) != len(want) {
		t.Fatalf("unexpected providers: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected provider at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestResolveFallbackProvidersUsesEnvOrder(t *testing.T) {
	t.Setenv("KCAL_BARCODE_FALLBACK_ORDER", "upc,off")
	got := resolveFallbackProviders("", "")
	want := []string{"upcitemdb", "openfoodfacts"}
	if len(got) != len(want) {
		t.Fatalf("unexpected providers: got=%v want=%v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected provider at %d: got=%q want=%q", i, got[i], want[i])
		}
	}
}

func TestLookupSearchCommandFlags(t *testing.T) {
	cmd, _, err := lookupCmd.Find([]string{"search"})
	if err != nil {
		t.Fatalf("find search command: %v", err)
	}
	if cmd == nil || cmd.Use != "search" {
		t.Fatalf("expected lookup search command to be registered")
	}
	if f := cmd.Flags().Lookup("verified-min-score"); f == nil {
		t.Fatalf("expected --verified-min-score flag")
	}
	if f := cmd.Flags().Lookup("query"); f == nil {
		t.Fatalf("expected --query flag")
	}
}

func TestLookupCacheSearchSubcommandsRegistered(t *testing.T) {
	if cmd, _, err := lookupCmd.Find([]string{"cache", "search-list"}); err != nil || cmd == nil || cmd.Use != "search-list" {
		t.Fatalf("expected lookup cache search-list command to be registered, err=%v", err)
	}
	if cmd, _, err := lookupCmd.Find([]string{"cache", "search-purge"}); err != nil || cmd == nil || cmd.Use != "search-purge" {
		t.Fatalf("expected lookup cache search-purge command to be registered, err=%v", err)
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
