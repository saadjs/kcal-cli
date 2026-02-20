package kcal

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var savedFoodCmd = &cobra.Command{
	Use:   "saved-food",
	Short: "Manage saved food templates",
}

var (
	savedFoodName        string
	savedFoodBrand       string
	savedFoodCategory    string
	savedFoodCalories    int
	savedFoodProtein     float64
	savedFoodCarbs       float64
	savedFoodFat         float64
	savedFoodFiber       float64
	savedFoodSugar       float64
	savedFoodSodium      float64
	savedFoodMicros      string
	savedFoodServingAmt  float64
	savedFoodServingUnit string
	savedFoodSourceType  string
	savedFoodSourceProv  string
	savedFoodSourceRef   string
	savedFoodNotes       string
	savedFoodMetadata    string
	savedFoodLimit       int
	savedFoodIncludeArch bool
	savedFoodQuery       string
	savedFoodDate        string
	savedFoodTime        string
	savedFoodServings    float64
	savedFoodEntryID     int64

	savedFoodProvider      string
	savedFoodAPIKey        string
	savedFoodAPIKeyType    string
	savedFoodFallback      bool
	savedFoodFallbackOrder string
)

var savedFoodAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add saved food template",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.CreateSavedFood(sqldb, service.CreateSavedFoodInput{
				Name:        savedFoodName,
				Brand:       savedFoodBrand,
				Category:    savedFoodCategory,
				Calories:    savedFoodCalories,
				ProteinG:    savedFoodProtein,
				CarbsG:      savedFoodCarbs,
				FatG:        savedFoodFat,
				FiberG:      savedFoodFiber,
				SugarG:      savedFoodSugar,
				SodiumMg:    savedFoodSodium,
				Micros:      savedFoodMicros,
				ServingAmt:  savedFoodServingAmt,
				ServingUnit: savedFoodServingUnit,
				SourceType:  savedFoodSourceType,
				SourceProv:  savedFoodSourceProv,
				SourceRef:   savedFoodSourceRef,
				Notes:       savedFoodNotes,
				Metadata:    savedFoodMetadata,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added saved food %d\n", id)
			return nil
		})
	},
}

var savedFoodAddFromEntryCmd = &cobra.Command{
	Use:   "add-from-entry <entry-id>",
	Short: "Create saved food from an existing entry snapshot",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entryID, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.CreateSavedFoodFromEntry(sqldb, entryID, savedFoodName, savedFoodNotes)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added saved food %d from entry %d\n", id, entryID)
			return nil
		})
	},
}

var savedFoodAddFromBarcodeCmd = &cobra.Command{
	Use:   "add-from-barcode <barcode>",
	Short: "Create saved food from barcode lookup",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		barcode := strings.TrimSpace(args[0])
		return withDB(func(sqldb *sql.DB) error {
			result, err := performBarcodeLookup(sqldb, barcode, savedFoodProvider, savedFoodAPIKey, savedFoodAPIKeyType, savedFoodFallback, savedFoodFallbackOrder)
			if err != nil {
				return err
			}
			name := savedFoodName
			if strings.TrimSpace(name) == "" {
				name = result.Description
			}
			id, err := service.CreateSavedFood(sqldb, service.CreateSavedFoodInput{
				Name:        name,
				Brand:       result.Brand,
				Category:    savedFoodCategory,
				Calories:    int(result.Calories),
				ProteinG:    result.ProteinG,
				CarbsG:      result.CarbsG,
				FatG:        result.FatG,
				FiberG:      result.FiberG,
				SugarG:      result.SugarG,
				SodiumMg:    result.SodiumMg,
				Micros:      mustEncodeMicros(result.Micronutrients),
				ServingAmt:  result.ServingAmount,
				ServingUnit: result.ServingUnit,
				SourceType:  "barcode",
				SourceProv:  result.Provider,
				SourceRef:   result.Barcode,
				Notes:       savedFoodNotes,
				Metadata:    savedFoodMetadata,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added saved food %d from barcode %s\n", id, barcode)
			return nil
		})
	},
}

var savedFoodListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved foods",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListSavedFoods(sqldb, service.ListSavedFoodsFilter{
				IncludeArchived: savedFoodIncludeArch,
				Limit:           savedFoodLimit,
				Query:           savedFoodQuery,
			})
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tNAME\tCATEGORY\tKCAL\tP\tC\tF\tUSAGE\tARCHIVED")
			for _, it := range items {
				archived := "no"
				if it.ArchivedAt != nil {
					archived = "yes"
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%d\t%.1f\t%.1f\t%.1f\t%d\t%s\n", it.ID, it.Name, it.DefaultCategory, it.Calories, it.ProteinG, it.CarbsG, it.FatG, it.UsageCount, archived)
			}
			return nil
		})
	},
}

var savedFoodShowCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show saved food details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			it, err := service.ResolveSavedFood(sqldb, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\n", it.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", it.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Brand: %s\n", it.Brand)
			fmt.Fprintf(cmd.OutOrStdout(), "Category: %s\n", it.DefaultCategory)
			fmt.Fprintf(cmd.OutOrStdout(), "Calories: %d\nProtein: %.1f\nCarbs: %.1f\nFat: %.1f\nFiber: %.1f\nSugar: %.1f\nSodium: %.1f\n", it.Calories, it.ProteinG, it.CarbsG, it.FatG, it.FiberG, it.SugarG, it.SodiumMg)
			fmt.Fprintf(cmd.OutOrStdout(), "Serving: %.2f %s\n", it.ServingAmount, it.ServingUnit)
			fmt.Fprintf(cmd.OutOrStdout(), "Source: %s (%s:%s)\n", it.SourceType, it.SourceProvider, it.SourceRef)
			fmt.Fprintf(cmd.OutOrStdout(), "Usage: %d\n", it.UsageCount)
			fmt.Fprintf(cmd.OutOrStdout(), "Notes: %s\n", it.Notes)
			return nil
		})
	},
}

var savedFoodUpdateCmd = &cobra.Command{
	Use:   "update <id|name>",
	Short: "Update saved food",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			err := service.UpdateSavedFood(sqldb, args[0], service.UpdateSavedFoodInput{
				Name:        savedFoodName,
				Brand:       savedFoodBrand,
				Category:    savedFoodCategory,
				Calories:    savedFoodCalories,
				ProteinG:    savedFoodProtein,
				CarbsG:      savedFoodCarbs,
				FatG:        savedFoodFat,
				FiberG:      savedFoodFiber,
				SugarG:      savedFoodSugar,
				SodiumMg:    savedFoodSodium,
				Micros:      savedFoodMicros,
				ServingAmt:  savedFoodServingAmt,
				ServingUnit: savedFoodServingUnit,
				SourceType:  savedFoodSourceType,
				SourceProv:  savedFoodSourceProv,
				SourceRef:   savedFoodSourceRef,
				Notes:       savedFoodNotes,
				Metadata:    savedFoodMetadata,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated saved food %s\n", args[0])
			return nil
		})
	},
}

var savedFoodArchiveCmd = &cobra.Command{
	Use:   "archive <id|name>",
	Short: "Archive saved food",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.ArchiveSavedFood(sqldb, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Archived saved food %s\n", args[0])
			return nil
		})
	},
}

var savedFoodRestoreCmd = &cobra.Command{
	Use:   "restore <id|name>",
	Short: "Restore saved food",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.RestoreSavedFood(sqldb, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Restored saved food %s\n", args[0])
			return nil
		})
	},
}

var savedFoodLogCmd = &cobra.Command{
	Use:   "log <id|name>",
	Short: "Log a saved food as an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		consumed, err := parseDateTimeOrNow(savedFoodDate, savedFoodTime)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.LogSavedFood(sqldb, service.LogSavedFoodInput{
				Identifier: args[0],
				Servings:   savedFoodServings,
				Category:   savedFoodCategory,
				ConsumedAt: consumed,
				Notes:      savedFoodNotes,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged saved food as entry %d\n", id)
			return nil
		})
	},
}

func mustEncodeMicros(m service.Micronutrients) string {
	out, err := service.EncodeMicronutrientsJSON(m)
	if err != nil {
		return ""
	}
	return out
}

func addSavedFoodTemplateFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&savedFoodName, "name", "", "Saved food name")
	cmd.Flags().StringVar(&savedFoodBrand, "brand", "", "Brand")
	cmd.Flags().StringVar(&savedFoodCategory, "category", "", "Default category (defaults to snacks)")
	cmd.Flags().IntVar(&savedFoodCalories, "calories", 0, "Calories")
	cmd.Flags().Float64Var(&savedFoodProtein, "protein", 0, "Protein grams")
	cmd.Flags().Float64Var(&savedFoodCarbs, "carbs", 0, "Carbs grams")
	cmd.Flags().Float64Var(&savedFoodFat, "fat", 0, "Fat grams")
	cmd.Flags().Float64Var(&savedFoodFiber, "fiber", 0, "Fiber grams")
	cmd.Flags().Float64Var(&savedFoodSugar, "sugar", 0, "Sugar grams")
	cmd.Flags().Float64Var(&savedFoodSodium, "sodium", 0, "Sodium milligrams")
	cmd.Flags().StringVar(&savedFoodMicros, "micros-json", "", "Micronutrients JSON object")
	cmd.Flags().Float64Var(&savedFoodServingAmt, "serving-amount", 1, "Serving amount")
	cmd.Flags().StringVar(&savedFoodServingUnit, "serving-unit", "serving", "Serving unit")
	cmd.Flags().StringVar(&savedFoodSourceType, "source-type", "manual", "Source type: manual|entry|barcode")
	cmd.Flags().StringVar(&savedFoodSourceProv, "source-provider", "", "Source provider")
	cmd.Flags().StringVar(&savedFoodSourceRef, "source-ref", "", "Source reference")
	cmd.Flags().StringVar(&savedFoodNotes, "notes", "", "Notes")
	cmd.Flags().StringVar(&savedFoodMetadata, "metadata-json", "", "Metadata JSON object")
}

func init() {
	rootCmd.AddCommand(savedFoodCmd)
	savedFoodCmd.AddCommand(savedFoodAddCmd, savedFoodAddFromEntryCmd, savedFoodAddFromBarcodeCmd, savedFoodListCmd, savedFoodShowCmd, savedFoodUpdateCmd, savedFoodArchiveCmd, savedFoodRestoreCmd, savedFoodLogCmd)

	addSavedFoodTemplateFlags(savedFoodAddCmd)
	_ = savedFoodAddCmd.MarkFlagRequired("name")

	savedFoodAddFromEntryCmd.Flags().StringVar(&savedFoodName, "name", "", "Override saved food name")
	savedFoodAddFromEntryCmd.Flags().StringVar(&savedFoodNotes, "notes", "", "Notes")

	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodName, "name", "", "Override saved food name")
	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodCategory, "category", "", "Default category (defaults to snacks)")
	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodNotes, "notes", "", "Notes")
	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodMetadata, "metadata-json", "", "Metadata JSON object")
	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodProvider, "provider", "", "Barcode provider")
	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodAPIKey, "api-key", "", "Provider API key")
	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodAPIKeyType, "api-key-type", "", "Provider API key type (UPCitemdb)")
	savedFoodAddFromBarcodeCmd.Flags().BoolVar(&savedFoodFallback, "fallback", true, "Try providers in fallback order")
	savedFoodAddFromBarcodeCmd.Flags().StringVar(&savedFoodFallbackOrder, "fallback-order", "", "Comma-separated fallback provider order")

	savedFoodListCmd.Flags().IntVar(&savedFoodLimit, "limit", 100, "Result limit")
	savedFoodListCmd.Flags().BoolVar(&savedFoodIncludeArch, "include-archived", false, "Include archived saved foods")
	savedFoodListCmd.Flags().StringVar(&savedFoodQuery, "query", "", "Filter by name")

	addSavedFoodTemplateFlags(savedFoodUpdateCmd)
	_ = savedFoodUpdateCmd.MarkFlagRequired("name")

	savedFoodLogCmd.Flags().Float64Var(&savedFoodServings, "servings", 1, "Serving multiplier")
	savedFoodLogCmd.Flags().StringVar(&savedFoodCategory, "category", "", "Optional category override")
	savedFoodLogCmd.Flags().StringVar(&savedFoodDate, "date", "", "Date in YYYY-MM-DD")
	savedFoodLogCmd.Flags().StringVar(&savedFoodTime, "time", "", "Time in HH:MM")
	savedFoodLogCmd.Flags().StringVar(&savedFoodNotes, "notes", "", "Optional notes")

	_ = savedFoodEntryID
}
