package kcal

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var entryCmd = &cobra.Command{
	Use:   "entry",
	Short: "Manage calorie and macro entries",
}

var (
	entryName          string
	entryCalories      int
	entryProtein       float64
	entryCarbs         float64
	entryFat           float64
	entryCategory      string
	entryDate          string
	entryTime          string
	entryNotes         string
	entryBarcode       string
	entryProvider      string
	entryAPIKey        string
	entryKeyType       string
	entryFallback      bool
	entryFallbackOrder string
	entryServings      float64
	entryMetadata      string
)

var entryAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new entry",
	RunE: func(cmd *cobra.Command, args []string) error {
		consumed, err := parseDateTimeOrNow(entryDate, entryTime)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			in, err := buildEntryAddInput(sqldb, consumed)
			if err != nil {
				return err
			}
			id, err := service.CreateEntry(sqldb, in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added entry %d\n", id)
			return nil
		})
	},
}

var (
	listDate     string
	listFromDate string
	listToDate   string
	listCategory string
	listLimit    int
	listMetadata bool
)

var entryListCmd = &cobra.Command{
	Use:   "list",
	Short: "List entries",
	RunE: func(cmd *cobra.Command, args []string) error {
		filter := service.ListEntriesFilter{
			Date:     listDate,
			FromDate: listFromDate,
			ToDate:   listToDate,
			Category: listCategory,
			Limit:    listLimit,
		}
		return withDB(func(sqldb *sql.DB) error {
			entries, err := service.ListEntries(sqldb, filter)
			if err != nil {
				return err
			}
			header := "ID\tDATE\tCATEGORY\tNAME\tKCAL\tP\tC\tF\tSOURCE"
			if listMetadata {
				header += "\tMETADATA"
			}
			fmt.Fprintln(cmd.OutOrStdout(), header)
			for _, e := range entries {
				if listMetadata {
					fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%s\t%d\t%.1f\t%.1f\t%.1f\t%s\t%s\n", e.ID, e.ConsumedAt.Local().Format("2006-01-02 15:04"), e.Category, e.Name, e.Calories, e.ProteinG, e.CarbsG, e.FatG, e.SourceType, e.Metadata)
					continue
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%s\t%d\t%.1f\t%.1f\t%.1f\t%s\n", e.ID, e.ConsumedAt.Local().Format("2006-01-02 15:04"), e.Category, e.Name, e.Calories, e.ProteinG, e.CarbsG, e.FatG, e.SourceType)
			}
			return nil
		})
	},
}

var entryShowCmd = &cobra.Command{
	Use:   "show <id>",
	Short: "Show a single entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			e, err := service.EntryByID(sqldb, id)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\n", e.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Date: %s\n", e.ConsumedAt.Local().Format("2006-01-02 15:04"))
			fmt.Fprintf(cmd.OutOrStdout(), "Category: %s\n", e.Category)
			fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", e.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Calories: %d\n", e.Calories)
			fmt.Fprintf(cmd.OutOrStdout(), "Protein: %.1f\nCarbs: %.1f\nFat: %.1f\n", e.ProteinG, e.CarbsG, e.FatG)
			fmt.Fprintf(cmd.OutOrStdout(), "Source: %s\n", e.SourceType)
			if e.SourceID != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Source ID: %d\n", *e.SourceID)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Notes: %s\n", e.Notes)
			fmt.Fprintf(cmd.OutOrStdout(), "Metadata: %s\n", e.Metadata)
			return nil
		})
	},
}

var (
	updateName     string
	updateCalories int
	updateProtein  float64
	updateCarbs    float64
	updateFat      float64
	updateCategory string
	updateDate     string
	updateTime     string
	updateNotes    string
	updateMetadata string
)

var entryUpdateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		consumed, err := parseDateTime(updateDate, updateTime)
		if err != nil {
			return err
		}

		in := service.UpdateEntryInput{
			ID:          id,
			Name:        updateName,
			Calories:    updateCalories,
			ProteinG:    updateProtein,
			CarbsG:      updateCarbs,
			FatG:        updateFat,
			Category:    updateCategory,
			Consumed:    consumed,
			Notes:       updateNotes,
			Metadata:    updateMetadata,
			MetadataSet: cmd.Flags().Changed("metadata-json"),
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateEntry(sqldb, in); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated entry %d\n", id)
			return nil
		})
	},
}

var entryMetadataSet string

var entryMetadataCmd = &cobra.Command{
	Use:   "metadata <id>",
	Short: "Update entry metadata JSON",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateEntryMetadata(sqldb, id, entryMetadataSet); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated metadata for entry %d\n", id)
			return nil
		})
	},
}

var entryDeleteCmd = &cobra.Command{
	Use:   "delete <id>",
	Short: "Delete an entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteEntry(sqldb, id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted entry %d\n", id)
			return nil
		})
	},
}

func addEntryFields(cmd *cobra.Command, prefix string) {
	cmd.Flags().StringVar(&entryName, "name", "", "Entry name")
	cmd.Flags().IntVar(&entryCalories, "calories", 0, "Calories")
	cmd.Flags().Float64Var(&entryProtein, "protein", 0, "Protein grams")
	cmd.Flags().Float64Var(&entryCarbs, "carbs", 0, "Carbs grams")
	cmd.Flags().Float64Var(&entryFat, "fat", 0, "Fat grams")
	cmd.Flags().StringVar(&entryCategory, "category", "", "Category name")
	cmd.Flags().StringVar(&entryDate, "date", "", "Date in YYYY-MM-DD")
	cmd.Flags().StringVar(&entryTime, "time", "", "Time in HH:MM")
	cmd.Flags().StringVar(&entryNotes, "notes", "", "Optional notes")
	cmd.Flags().StringVar(&entryBarcode, "barcode", "", "Barcode to lookup and log in one step")
	cmd.Flags().StringVar(&entryProvider, "provider", "", "Barcode provider: usda, openfoodfacts, or upcitemdb")
	cmd.Flags().StringVar(&entryAPIKey, "api-key", "", "Provider API key (USDA/UPCitemdb)")
	cmd.Flags().StringVar(&entryKeyType, "api-key-type", "", "Provider API key type (UPCitemdb)")
	cmd.Flags().BoolVar(&entryFallback, "fallback", true, "Try providers in fallback order when using --barcode")
	cmd.Flags().StringVar(&entryFallbackOrder, "fallback-order", "", "Comma-separated fallback provider order")
	cmd.Flags().Float64Var(&entryServings, "servings", 1, "Serving multiplier when logging by barcode")
	cmd.Flags().StringVar(&entryMetadata, "metadata-json", "", "Optional metadata JSON object to attach to the entry")
	_ = prefix
}

func buildEntryAddInput(sqldb *sql.DB, consumed time.Time) (service.CreateEntryInput, error) {
	if strings.TrimSpace(entryCategory) == "" {
		return service.CreateEntryInput{}, fmt.Errorf("--category is required")
	}
	if strings.TrimSpace(entryBarcode) == "" {
		if strings.TrimSpace(entryName) == "" {
			return service.CreateEntryInput{}, fmt.Errorf("--name is required when --barcode is not used")
		}
		return service.CreateEntryInput{
			Name:       entryName,
			Calories:   entryCalories,
			ProteinG:   entryProtein,
			CarbsG:     entryCarbs,
			FatG:       entryFat,
			Category:   entryCategory,
			Consumed:   consumed,
			Notes:      entryNotes,
			SourceType: "manual",
			Metadata:   entryMetadata,
		}, nil
	}

	if strings.TrimSpace(entryName) != "" ||
		entryCalories != 0 ||
		entryProtein != 0 ||
		entryCarbs != 0 ||
		entryFat != 0 {
		return service.CreateEntryInput{}, fmt.Errorf("cannot combine --barcode with manual nutrition flags (--name/--calories/--protein/--carbs/--fat)")
	}
	if entryServings <= 0 {
		return service.CreateEntryInput{}, fmt.Errorf("--servings must be > 0")
	}

	result, err := performBarcodeLookup(sqldb, entryBarcode, entryProvider, entryAPIKey, entryKeyType, entryFallback, entryFallbackOrder)
	if err != nil {
		return service.CreateEntryInput{}, err
	}

	var sourceID *int64
	if result.SourceID > 0 {
		v := result.SourceID
		sourceID = &v
	}
	metadata, err := buildBarcodeEntryMetadata(result, entryServings, entryMetadata)
	if err != nil {
		return service.CreateEntryInput{}, err
	}
	return service.CreateEntryInput{
		Name:       fmt.Sprintf("%s (barcode %s x%.2f)", result.Description, result.Barcode, entryServings),
		Calories:   int(math.Round(result.Calories * entryServings)),
		ProteinG:   result.ProteinG * entryServings,
		CarbsG:     result.CarbsG * entryServings,
		FatG:       result.FatG * entryServings,
		Category:   entryCategory,
		Consumed:   consumed,
		Notes:      entryNotes,
		SourceType: "barcode",
		SourceID:   sourceID,
		Metadata:   metadata,
	}, nil
}

func buildBarcodeEntryMetadata(result service.BarcodeLookupResult, servings float64, userMetadata string) (string, error) {
	metadata := map[string]any{
		"provider":               result.Provider,
		"barcode":                result.Barcode,
		"source_tier":            result.SourceTier,
		"provider_confidence":    result.ProviderConfidence,
		"nutrition_completeness": result.NutritionCompleteness,
		"lookup_trail":           result.LookupTrail,
		"from_override":          result.FromOverride,
		"from_cache":             result.FromCache,
		"servings":               servings,
	}
	if strings.TrimSpace(userMetadata) != "" {
		user, err := parseMetadataObject(userMetadata)
		if err != nil {
			return "", err
		}
		for k, v := range user {
			metadata[k] = v
		}
	}
	b, err := json.Marshal(metadata)
	if err != nil {
		return "", fmt.Errorf("marshal barcode entry metadata: %w", err)
	}
	return string(b), nil
}

func parseMetadataObject(value string) (map[string]any, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return map[string]any{}, nil
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(value), &out); err != nil {
		return nil, fmt.Errorf("--metadata-json must be a valid JSON object: %w", err)
	}
	return out, nil
}

func init() {
	rootCmd.AddCommand(entryCmd)
	entryCmd.AddCommand(entryAddCmd, entryListCmd, entryShowCmd, entryMetadataCmd, entryUpdateCmd, entryDeleteCmd)

	addEntryFields(entryAddCmd, "add")
	_ = entryAddCmd.MarkFlagRequired("category")

	entryListCmd.Flags().StringVar(&listDate, "date", "", "Filter by date YYYY-MM-DD")
	entryListCmd.Flags().StringVar(&listFromDate, "from", "", "Filter from date YYYY-MM-DD")
	entryListCmd.Flags().StringVar(&listToDate, "to", "", "Filter to date YYYY-MM-DD")
	entryListCmd.Flags().StringVar(&listCategory, "category", "", "Filter by category")
	entryListCmd.Flags().IntVar(&listLimit, "limit", 50, "Result limit")
	entryListCmd.Flags().BoolVar(&listMetadata, "with-metadata", false, "Include metadata JSON column")

	entryUpdateCmd.Flags().StringVar(&updateName, "name", "", "Entry name")
	entryUpdateCmd.Flags().IntVar(&updateCalories, "calories", 0, "Calories")
	entryUpdateCmd.Flags().Float64Var(&updateProtein, "protein", 0, "Protein grams")
	entryUpdateCmd.Flags().Float64Var(&updateCarbs, "carbs", 0, "Carbs grams")
	entryUpdateCmd.Flags().Float64Var(&updateFat, "fat", 0, "Fat grams")
	entryUpdateCmd.Flags().StringVar(&updateCategory, "category", "", "Category name")
	entryUpdateCmd.Flags().StringVar(&updateDate, "date", "", "Date in YYYY-MM-DD")
	entryUpdateCmd.Flags().StringVar(&updateTime, "time", "", "Time in HH:MM")
	entryUpdateCmd.Flags().StringVar(&updateNotes, "notes", "", "Optional notes")
	entryUpdateCmd.Flags().StringVar(&updateMetadata, "metadata-json", "", "Metadata JSON object")
	_ = entryUpdateCmd.MarkFlagRequired("name")
	_ = entryUpdateCmd.MarkFlagRequired("calories")
	_ = entryUpdateCmd.MarkFlagRequired("protein")
	_ = entryUpdateCmd.MarkFlagRequired("carbs")
	_ = entryUpdateCmd.MarkFlagRequired("fat")
	_ = entryUpdateCmd.MarkFlagRequired("category")
	_ = entryUpdateCmd.MarkFlagRequired("date")
	_ = entryUpdateCmd.MarkFlagRequired("time")
	entryMetadataCmd.Flags().StringVar(&entryMetadataSet, "metadata-json", "", "Metadata JSON object")
	_ = entryMetadataCmd.MarkFlagRequired("metadata-json")
}
