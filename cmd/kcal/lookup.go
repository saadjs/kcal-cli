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
)

var (
	lookupAPIKey string
	lookupJSON   bool
)

var lookupBarcodeCmd = &cobra.Command{
	Use:   "barcode <code>",
	Short: "Lookup food by barcode using USDA provider",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		apiKey := resolveUSDAAPIKey(lookupAPIKey)
		if strings.TrimSpace(apiKey) == "" {
			return fmt.Errorf("missing USDA API key; set --api-key or KCAL_USDA_API_KEY (see: kcal lookup usda-help)")
		}
		barcode := strings.TrimSpace(args[0])
		return withDB(func(sqldb *sql.DB) error {
			result, err := service.LookupBarcodeUSDA(sqldb, apiKey, barcode)
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
			if result.FromCache {
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

func init() {
	rootCmd.AddCommand(lookupCmd)
	lookupCmd.AddCommand(lookupBarcodeCmd, lookupUSDAHelpCmd)

	lookupBarcodeCmd.Flags().StringVar(&lookupAPIKey, "api-key", "", "USDA API key (fallback: KCAL_USDA_API_KEY)")
	lookupBarcodeCmd.Flags().BoolVar(&lookupJSON, "json", false, "Output as JSON")
}
