package kcal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var lookupCmd = &cobra.Command{
	Use:   "lookup",
	Short: "Lookup nutrition data from external providers",
}

const (
	usdaAPIGuideURL      = "https://fdc.nal.usda.gov/api-guide/"
	usdaSignupURL        = "https://api.data.gov/signup/"
	usdaRateLimitSummary = "USDA default rate limit is 1,000 requests per hour per IP."
	offAPIDocsURL        = "https://openfoodfacts.github.io/openfoodfacts-server/api/"
	offRateLimitSummary  = "Open Food Facts enforces fair-use limits and requires a descriptive User-Agent."
	upcDocsURL           = "https://devs.upcitemdb.com/"
	upcLimitTrial        = "Trial endpoint: up to 100 requests/day."
	upcLimitDev          = "DEV plan: up to 20,000 lookup/day and 2,000 search/day."
	upcLimitPro          = "PRO plan: up to 150,000 lookup/day and 20,000 search/day."
	defaultFallbackOrder = "usda,openfoodfacts,upcitemdb"
)

var (
	lookupProvider      string
	lookupAPIKey        string
	lookupAPIKeyType    string
	lookupFallback      bool
	lookupFallbackOrder string
	lookupJSON          bool
	lookupQuery         string
	lookupSearchLimit   int
	lookupVerifiedOnly  bool
	lookupVerifiedMin   float64
	overrideName        string
	overrideBrand       string
	overrideAmount      float64
	overrideUnit        string
	overrideKcal        float64
	overrideP           float64
	overrideC           float64
	overrideF           float64
	overrideFiber       float64
	overrideSugar       float64
	overrideSodium      float64
	overrideMicros      string
	overrideNotes       string
	overrideLimit       int
	cacheLimit          int
	cacheBarcode        string
	cachePurgeAll       bool
	cacheSearchQuery    string
	cacheSearchPurgeAll bool
)

var lookupBarcodeCmd = &cobra.Command{
	Use:   "barcode <code>",
	Short: "Lookup food by barcode using configured provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		barcode := strings.TrimSpace(args[0])
		return withDB(func(sqldb *sql.DB) error {
			result, err := performBarcodeLookup(sqldb, barcode, lookupProvider, lookupAPIKey, lookupAPIKeyType, lookupFallback, lookupFallbackOrder)
			if err != nil {
				return err
			}
			if lookupJSON {
				b, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal barcode lookup json: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			source := "live"
			if result.FromOverride {
				source = "override"
			} else if result.FromCache {
				source = "cache"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s (%s)\n", result.Provider, source)
			fmt.Fprintf(cmd.OutOrStdout(), "Barcode: %s\n", result.Barcode)
			fmt.Fprintf(cmd.OutOrStdout(), "Food: %s\n", result.Description)
			fmt.Fprintf(cmd.OutOrStdout(), "Brand: %s\n", result.Brand)
			fmt.Fprintf(cmd.OutOrStdout(), "Serving: %.2f %s\n", result.ServingAmount, result.ServingUnit)
			fmt.Fprintf(cmd.OutOrStdout(), "Calories: %.1f\nProtein: %.1fg\nCarbs: %.1fg\nFat: %.1fg\nFiber: %.1fg\nSugar: %.1fg\nSodium: %.1fmg\nMicronutrients: %s\n", result.Calories, result.ProteinG, result.CarbsG, result.FatG, result.FiberG, result.SugarG, result.SodiumMg, formatLookupMicronutrients(result.Micronutrients))
			fmt.Fprintf(cmd.OutOrStdout(), "Confidence: %.2f (%s)\n", result.ProviderConfidence, result.NutritionCompleteness)
			if len(result.LookupTrail) > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Lookup trail: %s\n", strings.Join(result.LookupTrail, " -> "))
			}
			return nil
		})
	},
}

var lookupSearchCmd = &cobra.Command{
	Use:   "search",
	Short: "Search food text across providers",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			results, err := performFoodSearch(sqldb, lookupQuery, lookupProvider, lookupAPIKey, lookupAPIKeyType, lookupFallback, lookupFallbackOrder, lookupSearchLimit, lookupVerifiedOnly, lookupVerifiedMin)
			if err != nil {
				return err
			}
			if lookupJSON {
				b, err := json.MarshalIndent(results, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal search lookup json: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "RANK\tSCORE\tVERIFIED\tPROVIDER\tFOOD\tBRAND\tKCAL\tSERVING\tCOMPLETENESS")
			for i, r := range results {
				fmt.Fprintf(
					cmd.OutOrStdout(),
					"%d\t%.2f\t%t\t%s\t%s\t%s\t%.1f\t%.2f %s\t%s\n",
					i+1,
					r.ConfidenceScore,
					r.IsVerified,
					r.Provider,
					r.Description,
					r.Brand,
					r.Calories,
					r.ServingAmount,
					r.ServingUnit,
					r.NutritionCompleteness,
				)
			}
			return nil
		})
	},
}

var lookupOverrideCmd = &cobra.Command{
	Use:   "override",
	Short: "Manage local barcode nutrition overrides",
}

var lookupCacheCmd = &cobra.Command{
	Use:   "cache",
	Short: "Manage barcode lookup cache",
}

var lookupCacheSearchListCmd = &cobra.Command{
	Use:   "search-list",
	Short: "List provider text-search cache rows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListProviderSearchCache(sqldb, lookupProvider, cacheSearchQuery, cacheLimit)
			if err != nil {
				return err
			}
			if lookupJSON {
				b, err := json.MarshalIndent(items, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal search cache list json: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PROVIDER\tQUERY\tLIMIT\tFETCHED_AT\tEXPIRES_AT")
			for _, it := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%d\t%s\t%s\n", it.Provider, it.Query, it.LimitRequested, it.FetchedAt.Format(time.RFC3339), it.ExpiresAt.Format(time.RFC3339))
			}
			return nil
		})
	},
}

var lookupCacheSearchPurgeCmd = &cobra.Command{
	Use:   "search-purge",
	Short: "Purge provider text-search cache rows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			count, err := service.PurgeProviderSearchCache(sqldb, lookupProvider, cacheSearchQuery, cacheSearchPurgeAll)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Purged %d search cache row(s)\n", count)
			return nil
		})
	},
}

var lookupCacheListCmd = &cobra.Command{
	Use:   "list",
	Short: "List barcode cache rows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListBarcodeCache(sqldb, lookupProvider, cacheLimit)
			if err != nil {
				return err
			}
			if lookupJSON {
				b, err := json.MarshalIndent(items, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal cache list json: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PROVIDER\tBARCODE\tNAME\tBRAND\tEXPIRES_AT")
			for _, it := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%s\t%s\n", it.Provider, it.Barcode, it.Description, it.Brand, it.ExpiresAt.Format(time.RFC3339))
			}
			return nil
		})
	},
}

var lookupCachePurgeCmd = &cobra.Command{
	Use:   "purge",
	Short: "Purge barcode cache rows",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			count, err := service.PurgeBarcodeCache(sqldb, lookupProvider, cacheBarcode, cachePurgeAll)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Purged %d cache row(s)\n", count)
			return nil
		})
	},
}

var lookupCacheRefreshCmd = &cobra.Command{
	Use:   "refresh <barcode>",
	Short: "Refresh cache from live provider lookup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := resolveBarcodeProvider(lookupProvider)
		apiKey, err := resolveProviderAPIKey(provider, lookupAPIKey)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			result, err := service.RefreshBarcodeCache(sqldb, provider, args[0], service.BarcodeLookupOptions{
				APIKey:     apiKey,
				APIKeyType: resolveProviderAPIKeyType(provider, lookupAPIKeyType),
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Refreshed cache for %s (%s): %s\n", result.Barcode, result.Provider, result.Description)
			return nil
		})
	},
}

var lookupOverrideSetCmd = &cobra.Command{
	Use:   "set <barcode>",
	Short: "Set or update local override for a barcode",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := resolveBarcodeProvider(lookupProvider)
		in := service.BarcodeOverrideInput{
			Description:    overrideName,
			Brand:          overrideBrand,
			ServingAmount:  overrideAmount,
			ServingUnit:    overrideUnit,
			Calories:       overrideKcal,
			ProteinG:       overrideP,
			CarbsG:         overrideC,
			FatG:           overrideF,
			FiberG:         overrideFiber,
			SugarG:         overrideSugar,
			SodiumMg:       overrideSodium,
			Micronutrients: overrideMicros,
			Notes:          overrideNotes,
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.SetBarcodeOverride(sqldb, provider, args[0], in); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Set override for %s (%s)\n", args[0], provider)
			return nil
		})
	},
}

var lookupOverrideShowCmd = &cobra.Command{
	Use:   "show <barcode>",
	Short: "Show local override for a barcode",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := resolveBarcodeProvider(lookupProvider)
		return withDB(func(sqldb *sql.DB) error {
			result, found, err := service.GetBarcodeOverride(sqldb, provider, args[0])
			if err != nil {
				return err
			}
			if !found {
				return fmt.Errorf("no override found for %s (%s)", args[0], provider)
			}
			if lookupJSON {
				b, err := json.MarshalIndent(result, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal override json: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s\nBarcode: %s\nFood: %s\nBrand: %s\nServing: %.2f %s\nCalories: %.1f\nProtein: %.1fg\nCarbs: %.1fg\nFat: %.1fg\nFiber: %.1fg\nSugar: %.1fg\nSodium: %.1fmg\nMicronutrients: %s\n", result.Provider, result.Barcode, result.Description, result.Brand, result.ServingAmount, result.ServingUnit, result.Calories, result.ProteinG, result.CarbsG, result.FatG, result.FiberG, result.SugarG, result.SodiumMg, formatLookupMicronutrients(result.Micronutrients))
			return nil
		})
	},
}

var lookupOverrideDeleteCmd = &cobra.Command{
	Use:   "delete <barcode>",
	Short: "Delete local override for a barcode",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := resolveBarcodeProvider(lookupProvider)
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteBarcodeOverride(sqldb, provider, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted override for %s (%s)\n", args[0], provider)
			return nil
		})
	},
}

var lookupOverrideListCmd = &cobra.Command{
	Use:   "list",
	Short: "List barcode overrides",
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := strings.ToLower(strings.TrimSpace(lookupProvider))
		if provider == "" {
			provider = ""
		}
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListBarcodeOverrides(sqldb, provider, overrideLimit)
			if err != nil {
				return err
			}
			if lookupJSON {
				b, err := json.MarshalIndent(items, "", "  ")
				if err != nil {
					return fmt.Errorf("marshal override list json: %w", err)
				}
				fmt.Fprintln(cmd.OutOrStdout(), string(b))
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "PROVIDER\tBARCODE\tNAME\tKCAL\tP\tC\tF\tFIBER_G\tSUGAR_G\tSODIUM_MG\tMICRONUTRIENTS")
			for _, it := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%.1f\t%s\n", it.Provider, it.Barcode, it.Description, it.Calories, it.ProteinG, it.CarbsG, it.FatG, it.FiberG, it.SugarG, it.SodiumMg, formatLookupMicronutrients(it.Micronutrients))
			}
			return nil
		})
	},
}

var lookupProvidersCmd = &cobra.Command{
	Use:   "providers",
	Short: "List available barcode providers and setup",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), providersHelpText())
		return nil
	},
}

var lookupUSDAHelpCmd = &cobra.Command{
	Use:   "usda-help",
	Short: "Show how to obtain and configure a USDA API key",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), usdaHelpText())
		return nil
	},
}

func usdaHelpText() string {
	return fmt.Sprintf(`USDA Barcode Lookup Setup

1) Get an API key:
- Sign up at: %s
- USDA API docs: %s

2) Configure key for kcal:
- Preferred: export KCAL_USDA_API_KEY=your_key_here
- Legacy fallback: export KCAL_BARCODE_API_KEY=your_key_here
- One-off override: kcal lookup barcode <code> --api-key your_key_here

3) Try a lookup:
- kcal lookup barcode 786012004549
- kcal lookup barcode 786012004549 --json

Rate limits:
- %s
- Cache is enabled in kcal to reduce repeated provider requests.`, usdaSignupURL, usdaAPIGuideURL, usdaRateLimitSummary)
}

func openFoodFactsHelpText() string {
	return fmt.Sprintf(`Open Food Facts Barcode Setup

1) API/docs:
- Docs: %s
- API key: not required for basic usage

2) Configure provider in kcal:
- One-off: kcal lookup barcode <code> --provider openfoodfacts
- Default via env: export KCAL_BARCODE_PROVIDER=openfoodfacts

Rate limits / usage:
- %s
- Cache is enabled in kcal to reduce repeated provider requests.`, offAPIDocsURL, offRateLimitSummary)
}

func providersHelpText() string {
	return `Available providers:
- usda (default): requires API key
- openfoodfacts: no API key required for basic usage
- upcitemdb: trial mode without key, paid plans with API key

Useful commands:
- kcal lookup usda-help
- kcal lookup openfoodfacts-help
- kcal lookup upcitemdb-help
- kcal lookup barcode <code> --provider usda|openfoodfacts|upcitemdb
- kcal lookup search --query "<text>" --fallback --limit 10
- kcal lookup override set|show|list|delete ...`
}

func upcItemDBHelpText() string {
	return fmt.Sprintf(`UPCitemdb Barcode Setup

1) Docs and plans:
- Docs: %s
- %s
- %s
- %s

2) Configure in kcal:
- Trial (no key): kcal lookup barcode <code> --provider upcitemdb
- Paid: export KCAL_UPCITEMDB_API_KEY=your_key
- Optional key type: export KCAL_UPCITEMDB_KEY_TYPE=3scale
- One-off key: kcal lookup barcode <code> --provider upcitemdb --api-key your_key --api-key-type 3scale

3) Notes:
- UPCitemdb may return limited nutrition fields for some products.
- Cache is enabled in kcal to reduce repeated provider requests.`, upcDocsURL, upcLimitTrial, upcLimitDev, upcLimitPro)
}

func resolveUSDAAPIKey(flagValue string) string {
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	if v := strings.TrimSpace(os.Getenv("KCAL_USDA_API_KEY")); v != "" {
		return v
	}
	if v := strings.TrimSpace(os.Getenv("KCAL_BARCODE_API_KEY")); v != "" {
		return v
	}
	return ""
}

func resolveBarcodeProvider(flagValue string) string {
	if v := strings.TrimSpace(flagValue); v != "" {
		return strings.ToLower(v)
	}
	if v := strings.TrimSpace(os.Getenv("KCAL_BARCODE_PROVIDER")); v != "" {
		return strings.ToLower(v)
	}
	return service.BarcodeProviderUSDA
}

func resolveProviderAPIKey(provider string, flagValue string) (string, error) {
	switch provider {
	case service.BarcodeProviderUSDA:
		key := resolveUSDAAPIKey(flagValue)
		if strings.TrimSpace(key) == "" {
			return "", fmt.Errorf("missing USDA API key; set --api-key or KCAL_USDA_API_KEY (see: kcal lookup usda-help)")
		}
		return key, nil
	case service.BarcodeProviderOpenFoodFacts, "off":
		return "", nil
	case service.BarcodeProviderUPCItemDB, "upc":
		if strings.TrimSpace(flagValue) != "" {
			return strings.TrimSpace(flagValue), nil
		}
		if v := strings.TrimSpace(os.Getenv("KCAL_UPCITEMDB_API_KEY")); v != "" {
			return v, nil
		}
		return "", nil
	default:
		return "", fmt.Errorf("unsupported provider %q (use usda, openfoodfacts, or upcitemdb)", provider)
	}
}

func resolveProviderAPIKeyType(provider string, flagValue string) string {
	if provider != service.BarcodeProviderUPCItemDB && provider != "upc" {
		return ""
	}
	if strings.TrimSpace(flagValue) != "" {
		return strings.TrimSpace(flagValue)
	}
	if v := strings.TrimSpace(os.Getenv("KCAL_UPCITEMDB_KEY_TYPE")); v != "" {
		return v
	}
	return "3scale"
}

func performBarcodeLookup(sqldb *sql.DB, barcode, providerFlag, apiKeyFlag, apiKeyTypeFlag string, fallback bool, fallbackOrder string) (service.BarcodeLookupResult, error) {
	providerFlag = strings.TrimSpace(providerFlag)
	if providerFlag == "" && strings.TrimSpace(os.Getenv("KCAL_BARCODE_PROVIDER")) == "" {
		if v := lookupConfigValue(sqldb, service.ConfigBarcodeProvider); v != "" {
			providerFlag = v
		}
	}
	fallbackOrder = strings.TrimSpace(fallbackOrder)
	if fallbackOrder == "" && strings.TrimSpace(os.Getenv("KCAL_BARCODE_FALLBACK_ORDER")) == "" {
		if v := lookupConfigValue(sqldb, service.ConfigBarcodeFallbackOrder); v != "" {
			fallbackOrder = v
		}
	}

	if !fallback {
		provider := resolveBarcodeProvider(providerFlag)
		apiKey, err := resolveProviderAPIKey(provider, apiKeyFlag)
		if err != nil {
			return service.BarcodeLookupResult{}, err
		}
		return service.LookupBarcode(sqldb, provider, barcode, service.BarcodeLookupOptions{
			APIKey:     apiKey,
			APIKeyType: resolveProviderAPIKeyType(provider, apiKeyTypeFlag),
		})
	}

	providers := resolveFallbackProviders(providerFlag, fallbackOrder)
	candidates := make([]service.BarcodeLookupCandidate, 0, len(providers))
	for _, p := range providers {
		apiKey, err := resolveProviderAPIKey(p, apiKeyFlag)
		if err != nil {
			// In fallback mode, skip providers that require unavailable credentials.
			continue
		}
		candidates = append(candidates, service.BarcodeLookupCandidate{
			Provider: p,
			Options: service.BarcodeLookupOptions{
				APIKey:     apiKey,
				APIKeyType: resolveProviderAPIKeyType(p, apiKeyTypeFlag),
			},
		})
	}
	if len(candidates) == 0 {
		return service.BarcodeLookupResult{}, fmt.Errorf("no usable lookup providers configured; set provider API key or disable fallback")
	}
	return service.LookupBarcodeWithFallback(sqldb, barcode, candidates)
}

func performFoodSearch(sqldb *sql.DB, query, providerFlag, apiKeyFlag, apiKeyTypeFlag string, fallback bool, fallbackOrder string, limit int, verifiedOnly bool, verifiedMin float64) ([]service.FoodSearchResult, error) {
	providerFlag = strings.TrimSpace(providerFlag)
	if providerFlag == "" && strings.TrimSpace(os.Getenv("KCAL_BARCODE_PROVIDER")) == "" {
		if v := lookupConfigValue(sqldb, service.ConfigBarcodeProvider); v != "" {
			providerFlag = v
		}
	}
	fallbackOrder = strings.TrimSpace(fallbackOrder)
	if fallbackOrder == "" && strings.TrimSpace(os.Getenv("KCAL_BARCODE_FALLBACK_ORDER")) == "" {
		if v := lookupConfigValue(sqldb, service.ConfigBarcodeFallbackOrder); v != "" {
			fallbackOrder = v
		}
	}
	opts := service.FoodSearchOptions{
		Provider:         providerFlag,
		Limit:            limit,
		VerifiedOnly:     verifiedOnly,
		VerifiedMinScore: verifiedMin,
	}
	if !fallback {
		provider := resolveBarcodeProvider(providerFlag)
		apiKey, err := resolveProviderAPIKey(provider, apiKeyFlag)
		if err != nil {
			return nil, err
		}
		opts.Provider = provider
		opts.APIKey = apiKey
		opts.APIKeyType = resolveProviderAPIKeyType(provider, apiKeyTypeFlag)
		return service.SearchFoods(sqldb, query, opts)
	}

	providers := resolveFallbackProviders(providerFlag, fallbackOrder)
	candidates := make([]service.FoodSearchCandidate, 0, len(providers))
	for _, p := range providers {
		apiKey, err := resolveProviderAPIKey(p, apiKeyFlag)
		if err != nil {
			continue
		}
		candidates = append(candidates, service.FoodSearchCandidate{
			Provider: p,
			Options: service.BarcodeLookupOptions{
				APIKey:     apiKey,
				APIKeyType: resolveProviderAPIKeyType(p, apiKeyTypeFlag),
			},
		})
	}
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no usable search providers configured; set provider API key or disable fallback")
	}
	return service.SearchFoodsWithFallback(sqldb, query, candidates, opts)
}

func lookupConfigValue(sqldb *sql.DB, key string) string {
	if sqldb == nil {
		return ""
	}
	v, ok, err := service.GetConfig(sqldb, key)
	if err != nil || !ok {
		return ""
	}
	return strings.TrimSpace(v)
}

func resolveFallbackProviders(providerFlag, fallbackOrder string) []string {
	order := strings.TrimSpace(fallbackOrder)
	if order == "" {
		order = strings.TrimSpace(os.Getenv("KCAL_BARCODE_FALLBACK_ORDER"))
	}
	if order == "" {
		order = defaultFallbackOrder
	}
	parsed := parseProviderOrder(order)
	if len(parsed) == 0 {
		parsed = parseProviderOrder(defaultFallbackOrder)
	}
	primary := strings.TrimSpace(providerFlag)
	if primary == "" {
		primary = strings.TrimSpace(os.Getenv("KCAL_BARCODE_PROVIDER"))
	}
	primary = strings.ToLower(primary)
	if primary == "" {
		return parsed
	}
	return prependUniqueProvider(primary, parsed)
}

func parseProviderOrder(value string) []string {
	parts := strings.Split(value, ",")
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, p := range parts {
		n := normalizeProviderToken(p)
		if n == "" || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	return out
}

func prependUniqueProvider(provider string, order []string) []string {
	n := normalizeProviderToken(provider)
	if n == "" {
		return order
	}
	out := []string{n}
	for _, p := range order {
		if p != n {
			out = append(out, p)
		}
	}
	return out
}

func normalizeProviderToken(p string) string {
	p = strings.ToLower(strings.TrimSpace(p))
	switch p {
	case "usda":
		return service.BarcodeProviderUSDA
	case "openfoodfacts", "off":
		return service.BarcodeProviderOpenFoodFacts
	case "upcitemdb", "upc":
		return service.BarcodeProviderUPCItemDB
	case "override", "cache":
		// Overrides/cache are handled internally per provider lookup.
		return ""
	default:
		return ""
	}
}

func formatLookupMicronutrients(m service.Micronutrients) string {
	if len(m) == 0 {
		return "-"
	}
	b, err := json.Marshal(m)
	if err != nil {
		return "-"
	}
	return string(b)
}

var lookupOpenFoodFactsHelpCmd = &cobra.Command{
	Use:   "openfoodfacts-help",
	Short: "Show setup and usage guidance for Open Food Facts provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), openFoodFactsHelpText())
		return nil
	},
}

var lookupUPCItemDBHelpCmd = &cobra.Command{
	Use:   "upcitemdb-help",
	Short: "Show setup and plan-limit guidance for UPCitemdb provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), upcItemDBHelpText())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lookupCmd)
	lookupCmd.AddCommand(lookupBarcodeCmd, lookupSearchCmd, lookupProvidersCmd, lookupUSDAHelpCmd, lookupOpenFoodFactsHelpCmd, lookupUPCItemDBHelpCmd, lookupOverrideCmd, lookupCacheCmd)
	lookupOverrideCmd.AddCommand(lookupOverrideSetCmd, lookupOverrideShowCmd, lookupOverrideListCmd, lookupOverrideDeleteCmd)
	lookupCacheCmd.AddCommand(lookupCacheListCmd, lookupCachePurgeCmd, lookupCacheRefreshCmd, lookupCacheSearchListCmd, lookupCacheSearchPurgeCmd)

	lookupBarcodeCmd.Flags().StringVar(&lookupProvider, "provider", "", "Barcode provider: usda, openfoodfacts, or upcitemdb (or set KCAL_BARCODE_PROVIDER)")
	lookupBarcodeCmd.Flags().StringVar(&lookupAPIKey, "api-key", "", "Provider API key (USDA/UPCitemdb)")
	lookupBarcodeCmd.Flags().StringVar(&lookupAPIKeyType, "api-key-type", "", "Provider API key type (UPCitemdb, default: 3scale)")
	lookupBarcodeCmd.Flags().BoolVar(&lookupFallback, "fallback", true, "Try providers in fallback order until one succeeds")
	lookupBarcodeCmd.Flags().StringVar(&lookupFallbackOrder, "fallback-order", "", "Comma-separated provider order (default: usda,openfoodfacts,upcitemdb)")
	lookupBarcodeCmd.Flags().BoolVar(&lookupJSON, "json", false, "Output as JSON")
	lookupSearchCmd.Flags().StringVar(&lookupQuery, "query", "", "Text query")
	lookupSearchCmd.Flags().StringVar(&lookupProvider, "provider", "", "Provider: usda, openfoodfacts, or upcitemdb")
	lookupSearchCmd.Flags().StringVar(&lookupAPIKey, "api-key", "", "Provider API key (USDA/UPCitemdb)")
	lookupSearchCmd.Flags().StringVar(&lookupAPIKeyType, "api-key-type", "", "Provider API key type (UPCitemdb, default: 3scale)")
	lookupSearchCmd.Flags().BoolVar(&lookupFallback, "fallback", true, "Aggregate results across fallback provider order")
	lookupSearchCmd.Flags().StringVar(&lookupFallbackOrder, "fallback-order", "", "Comma-separated provider order (default: usda,openfoodfacts,upcitemdb)")
	lookupSearchCmd.Flags().IntVar(&lookupSearchLimit, "limit", 10, "Maximum results")
	lookupSearchCmd.Flags().BoolVar(&lookupVerifiedOnly, "verified-only", false, "Return only verified foods")
	lookupSearchCmd.Flags().Float64Var(&lookupVerifiedMin, "verified-min-score", service.DefaultVerifiedMinScore, "Minimum confidence score to mark as verified")
	lookupSearchCmd.Flags().BoolVar(&lookupJSON, "json", false, "Output as JSON")
	_ = lookupSearchCmd.MarkFlagRequired("query")

	for _, c := range []*cobra.Command{lookupOverrideSetCmd, lookupOverrideShowCmd, lookupOverrideListCmd, lookupOverrideDeleteCmd} {
		c.Flags().StringVar(&lookupProvider, "provider", "", "Barcode provider: usda, openfoodfacts, or upcitemdb (default from KCAL_BARCODE_PROVIDER/usda)")
		c.Flags().BoolVar(&lookupJSON, "json", false, "Output as JSON")
	}
	lookupOverrideSetCmd.Flags().StringVar(&overrideName, "name", "", "Food name")
	lookupOverrideSetCmd.Flags().StringVar(&overrideBrand, "brand", "", "Brand")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideAmount, "serving-amount", 0, "Serving amount")
	lookupOverrideSetCmd.Flags().StringVar(&overrideUnit, "serving-unit", "", "Serving unit")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideKcal, "calories", 0, "Calories")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideP, "protein", 0, "Protein grams")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideC, "carbs", 0, "Carbs grams")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideF, "fat", 0, "Fat grams")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideFiber, "fiber", 0, "Fiber grams")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideSugar, "sugar", 0, "Sugar grams")
	lookupOverrideSetCmd.Flags().Float64Var(&overrideSodium, "sodium", 0, "Sodium milligrams")
	lookupOverrideSetCmd.Flags().StringVar(&overrideMicros, "micros-json", "", "Micronutrients JSON object")
	lookupOverrideSetCmd.Flags().StringVar(&overrideNotes, "notes", "", "Override notes")
	_ = lookupOverrideSetCmd.MarkFlagRequired("name")
	_ = lookupOverrideSetCmd.MarkFlagRequired("serving-amount")
	_ = lookupOverrideSetCmd.MarkFlagRequired("serving-unit")
	_ = lookupOverrideSetCmd.MarkFlagRequired("calories")
	_ = lookupOverrideSetCmd.MarkFlagRequired("protein")
	_ = lookupOverrideSetCmd.MarkFlagRequired("carbs")
	_ = lookupOverrideSetCmd.MarkFlagRequired("fat")
	lookupOverrideListCmd.Flags().IntVar(&overrideLimit, "limit", 100, "Max overrides to return")
	lookupCacheListCmd.Flags().StringVar(&lookupProvider, "provider", "", "Filter by provider")
	lookupCacheListCmd.Flags().IntVar(&cacheLimit, "limit", 100, "Max cached rows to return")
	lookupCacheListCmd.Flags().BoolVar(&lookupJSON, "json", false, "Output as JSON")
	lookupCachePurgeCmd.Flags().StringVar(&lookupProvider, "provider", "", "Purge cache rows for provider")
	lookupCachePurgeCmd.Flags().StringVar(&cacheBarcode, "barcode", "", "Purge cache rows for barcode")
	lookupCachePurgeCmd.Flags().BoolVar(&cachePurgeAll, "all", false, "Purge all cache rows")
	lookupCacheRefreshCmd.Flags().StringVar(&lookupProvider, "provider", "", "Provider for refresh")
	lookupCacheRefreshCmd.Flags().StringVar(&lookupAPIKey, "api-key", "", "Provider API key (USDA/UPCitemdb)")
	lookupCacheRefreshCmd.Flags().StringVar(&lookupAPIKeyType, "api-key-type", "", "Provider API key type (UPCitemdb)")
	lookupCacheSearchListCmd.Flags().StringVar(&lookupProvider, "provider", "", "Filter by provider")
	lookupCacheSearchListCmd.Flags().StringVar(&cacheSearchQuery, "query", "", "Filter by normalized query")
	lookupCacheSearchListCmd.Flags().IntVar(&cacheLimit, "limit", 100, "Max search cache rows to return")
	lookupCacheSearchListCmd.Flags().BoolVar(&lookupJSON, "json", false, "Output as JSON")
	lookupCacheSearchPurgeCmd.Flags().StringVar(&lookupProvider, "provider", "", "Purge search cache rows for provider")
	lookupCacheSearchPurgeCmd.Flags().StringVar(&cacheSearchQuery, "query", "", "Purge search cache rows for query")
	lookupCacheSearchPurgeCmd.Flags().BoolVar(&cacheSearchPurgeAll, "all", false, "Purge all search cache rows")
}
