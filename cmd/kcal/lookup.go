package kcal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/saad/kcal-cli/internal/service"
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
)

var (
	lookupProvider string
	lookupAPIKey   string
	lookupJSON     bool
	overrideName   string
	overrideBrand  string
	overrideAmount float64
	overrideUnit   string
	overrideKcal   float64
	overrideP      float64
	overrideC      float64
	overrideF      float64
	overrideNotes  string
	overrideLimit  int
)

var lookupBarcodeCmd = &cobra.Command{
	Use:   "barcode <code>",
	Short: "Lookup food by barcode using configured provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := resolveBarcodeProvider(lookupProvider)
		apiKey, err := resolveProviderAPIKey(provider, lookupAPIKey)
		if err != nil {
			return err
		}
		barcode := strings.TrimSpace(args[0])
		return withDB(func(sqldb *sql.DB) error {
			result, err := service.LookupBarcode(sqldb, provider, apiKey, barcode)
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
			fmt.Fprintf(cmd.OutOrStdout(), "Calories: %.1f\nProtein: %.1fg\nCarbs: %.1fg\nFat: %.1fg\n", result.Calories, result.ProteinG, result.CarbsG, result.FatG)
			return nil
		})
	},
}

var lookupOverrideCmd = &cobra.Command{
	Use:   "override",
	Short: "Manage local barcode nutrition overrides",
}

var lookupOverrideSetCmd = &cobra.Command{
	Use:   "set <barcode>",
	Short: "Set or update local override for a barcode",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		provider := resolveBarcodeProvider(lookupProvider)
		in := service.BarcodeOverrideInput{
			Description:   overrideName,
			Brand:         overrideBrand,
			ServingAmount: overrideAmount,
			ServingUnit:   overrideUnit,
			Calories:      overrideKcal,
			ProteinG:      overrideP,
			CarbsG:        overrideC,
			FatG:          overrideF,
			Notes:         overrideNotes,
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
			fmt.Fprintf(cmd.OutOrStdout(), "Provider: %s\nBarcode: %s\nFood: %s\nBrand: %s\nServing: %.2f %s\nCalories: %.1f\nProtein: %.1fg\nCarbs: %.1fg\nFat: %.1fg\n", result.Provider, result.Barcode, result.Description, result.Brand, result.ServingAmount, result.ServingUnit, result.Calories, result.ProteinG, result.CarbsG, result.FatG)
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
			fmt.Fprintln(cmd.OutOrStdout(), "PROVIDER\tBARCODE\tNAME\tKCAL\tP\tC\tF")
			for _, it := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%s\t%s\t%s\t%.1f\t%.1f\t%.1f\t%.1f\n", it.Provider, it.Barcode, it.Description, it.Calories, it.ProteinG, it.CarbsG, it.FatG)
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

Useful commands:
- kcal lookup usda-help
- kcal lookup openfoodfacts-help
- kcal lookup barcode <code> --provider usda|openfoodfacts
- kcal lookup override set|show|list|delete ...`
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
	default:
		return "", fmt.Errorf("unsupported provider %q (use usda or openfoodfacts)", provider)
	}
}

var lookupOpenFoodFactsHelpCmd = &cobra.Command{
	Use:   "openfoodfacts-help",
	Short: "Show setup and usage guidance for Open Food Facts provider",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), openFoodFactsHelpText())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(lookupCmd)
	lookupCmd.AddCommand(lookupBarcodeCmd, lookupProvidersCmd, lookupUSDAHelpCmd, lookupOpenFoodFactsHelpCmd, lookupOverrideCmd)
	lookupOverrideCmd.AddCommand(lookupOverrideSetCmd, lookupOverrideShowCmd, lookupOverrideListCmd, lookupOverrideDeleteCmd)

	lookupBarcodeCmd.Flags().StringVar(&lookupProvider, "provider", "", "Barcode provider: usda or openfoodfacts (or set KCAL_BARCODE_PROVIDER)")
	lookupBarcodeCmd.Flags().StringVar(&lookupAPIKey, "api-key", "", "USDA API key (fallback: KCAL_USDA_API_KEY)")
	lookupBarcodeCmd.Flags().BoolVar(&lookupJSON, "json", false, "Output as JSON")

	for _, c := range []*cobra.Command{lookupOverrideSetCmd, lookupOverrideShowCmd, lookupOverrideListCmd, lookupOverrideDeleteCmd} {
		c.Flags().StringVar(&lookupProvider, "provider", "", "Barcode provider: usda or openfoodfacts (default from KCAL_BARCODE_PROVIDER/usda)")
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
	lookupOverrideSetCmd.Flags().StringVar(&overrideNotes, "notes", "", "Override notes")
	_ = lookupOverrideSetCmd.MarkFlagRequired("name")
	_ = lookupOverrideSetCmd.MarkFlagRequired("serving-amount")
	_ = lookupOverrideSetCmd.MarkFlagRequired("serving-unit")
	_ = lookupOverrideSetCmd.MarkFlagRequired("calories")
	_ = lookupOverrideSetCmd.MarkFlagRequired("protein")
	_ = lookupOverrideSetCmd.MarkFlagRequired("carbs")
	_ = lookupOverrideSetCmd.MarkFlagRequired("fat")

	lookupOverrideListCmd.Flags().IntVar(&overrideLimit, "limit", 100, "Max overrides to return")
}
