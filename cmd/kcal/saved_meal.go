package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saadjs/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var savedMealCmd = &cobra.Command{
	Use:   "saved-meal",
	Short: "Manage saved meal templates",
}

var (
	savedMealName        string
	savedMealCategory    string
	savedMealNotes       string
	savedMealIncludeArch bool
	savedMealLimit       int
	savedMealQuery       string
	savedMealServings    float64
	savedMealDate        string
	savedMealTime        string

	savedMealComponentName     string
	savedMealComponentQty      float64
	savedMealComponentUnit     string
	savedMealComponentCalories int
	savedMealComponentProtein  float64
	savedMealComponentCarbs    float64
	savedMealComponentFat      float64
	savedMealComponentFiber    float64
	savedMealComponentSugar    float64
	savedMealComponentSodium   float64
	savedMealComponentMicros   string
	savedMealComponentPosition int
	savedMealComponentFood     string
	savedMealFromEntryCompName string
)

var savedMealAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add saved meal template",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.CreateSavedMeal(sqldb, service.CreateSavedMealInput{
				Name:     savedMealName,
				Category: savedMealCategory,
				Notes:    savedMealNotes,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added saved meal %d\n", id)
			return nil
		})
	},
}

var savedMealAddFromEntryCmd = &cobra.Command{
	Use:   "add-from-entry <entry-id>",
	Short: "Create saved meal with one component from an existing entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		entryID, err := parseInt64Arg("entry id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.CreateSavedMealFromEntry(sqldb, entryID, savedMealName, savedMealFromEntryCompName)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added saved meal %d from entry %d\n", id, entryID)
			return nil
		})
	},
}

var savedMealListCmd = &cobra.Command{
	Use:   "list",
	Short: "List saved meals",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListSavedMeals(sqldb, service.ListSavedMealsFilter{
				IncludeArchived: savedMealIncludeArch,
				Limit:           savedMealLimit,
				Query:           savedMealQuery,
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
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%s\t%d\t%.1f\t%.1f\t%.1f\t%d\t%s\n", it.ID, it.Name, it.DefaultCategory, it.CaloriesTotal, it.ProteinTotalG, it.CarbsTotalG, it.FatTotalG, it.UsageCount, archived)
			}
			return nil
		})
	},
}

var savedMealShowCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show saved meal details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			meal, err := service.ResolveSavedMeal(sqldb, args[0])
			if err != nil {
				return err
			}
			comps, err := service.ListSavedMealComponents(sqldb, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\n", meal.ID)
			fmt.Fprintf(cmd.OutOrStdout(), "Name: %s\n", meal.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "Category: %s\n", meal.DefaultCategory)
			fmt.Fprintf(cmd.OutOrStdout(), "Calories: %d\nProtein: %.1f\nCarbs: %.1f\nFat: %.1f\n", meal.CaloriesTotal, meal.ProteinTotalG, meal.CarbsTotalG, meal.FatTotalG)
			fmt.Fprintf(cmd.OutOrStdout(), "Components: %d\n", len(comps))
			for _, c := range comps {
				fmt.Fprintf(cmd.OutOrStdout(), "  - %d: %s (%.2f %s) kcal=%d\n", c.ID, c.Name, c.Quantity, c.Unit, c.Calories)
			}
			return nil
		})
	},
}

var savedMealUpdateCmd = &cobra.Command{
	Use:   "update <id|name>",
	Short: "Update saved meal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateSavedMeal(sqldb, args[0], service.UpdateSavedMealInput{
				Name:     savedMealName,
				Category: savedMealCategory,
				Notes:    savedMealNotes,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated saved meal %s\n", args[0])
			return nil
		})
	},
}

var savedMealArchiveCmd = &cobra.Command{
	Use:   "archive <id|name>",
	Short: "Archive saved meal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.ArchiveSavedMeal(sqldb, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Archived saved meal %s\n", args[0])
			return nil
		})
	},
}

var savedMealRestoreCmd = &cobra.Command{
	Use:   "restore <id|name>",
	Short: "Restore saved meal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.RestoreSavedMeal(sqldb, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Restored saved meal %s\n", args[0])
			return nil
		})
	},
}

var savedMealLogCmd = &cobra.Command{
	Use:   "log <id|name>",
	Short: "Log saved meal as one entry",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		consumed, err := parseDateTimeOrNow(savedMealDate, savedMealTime)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.LogSavedMeal(sqldb, service.LogSavedMealInput{
				Identifier: args[0],
				Servings:   savedMealServings,
				Category:   savedMealCategory,
				ConsumedAt: consumed,
				Notes:      savedMealNotes,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged saved meal as entry %d\n", id)
			return nil
		})
	},
}

var savedMealComponentCmd = &cobra.Command{
	Use:   "component",
	Short: "Manage saved meal components",
}

var savedMealComponentAddCmd = &cobra.Command{
	Use:   "add <meal-id|name>",
	Short: "Add component to saved meal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.AddSavedMealComponent(sqldb, args[0], service.SavedMealComponentInput{
				SavedFoodIdentifier: savedMealComponentFood,
				Name:                savedMealComponentName,
				Quantity:            savedMealComponentQty,
				Unit:                savedMealComponentUnit,
				Calories:            savedMealComponentCalories,
				ProteinG:            savedMealComponentProtein,
				CarbsG:              savedMealComponentCarbs,
				FatG:                savedMealComponentFat,
				FiberG:              savedMealComponentFiber,
				SugarG:              savedMealComponentSugar,
				SodiumMg:            savedMealComponentSodium,
				Micros:              savedMealComponentMicros,
				Position:            savedMealComponentPosition,
			})
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added component %d\n", id)
			return nil
		})
	},
}

var savedMealComponentListCmd = &cobra.Command{
	Use:   "list <meal-id|name>",
	Short: "List components for saved meal",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListSavedMealComponents(sqldb, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tPOS\tNAME\tQTY\tUNIT\tKCAL\tP\tC\tF\tSAVED_FOOD_ID")
			for _, it := range items {
				sf := ""
				if it.SavedFoodID != nil {
					sf = fmt.Sprintf("%d", *it.SavedFoodID)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%d\t%s\t%.2f\t%s\t%d\t%.1f\t%.1f\t%.1f\t%s\n", it.ID, it.Position, it.Name, it.Quantity, it.Unit, it.Calories, it.ProteinG, it.CarbsG, it.FatG, sf)
			}
			return nil
		})
	},
}

var savedMealComponentUpdateCmd = &cobra.Command{
	Use:   "update <component-id>",
	Short: "Update saved meal component",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		componentID, err := parseInt64Arg("component id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateSavedMealComponent(sqldb, componentID, service.UpdateSavedMealComponentInput{
				Name:     savedMealComponentName,
				Quantity: savedMealComponentQty,
				Unit:     savedMealComponentUnit,
				Calories: savedMealComponentCalories,
				ProteinG: savedMealComponentProtein,
				CarbsG:   savedMealComponentCarbs,
				FatG:     savedMealComponentFat,
				FiberG:   savedMealComponentFiber,
				SugarG:   savedMealComponentSugar,
				SodiumMg: savedMealComponentSodium,
				Micros:   savedMealComponentMicros,
				Position: savedMealComponentPosition,
			}); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated component %d\n", componentID)
			return nil
		})
	},
}

var savedMealComponentDeleteCmd = &cobra.Command{
	Use:   "delete <component-id>",
	Short: "Delete saved meal component",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		componentID, err := parseInt64Arg("component id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteSavedMealComponent(sqldb, componentID); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted component %d\n", componentID)
			return nil
		})
	},
}

func addSavedMealFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&savedMealName, "name", "", "Saved meal name")
	cmd.Flags().StringVar(&savedMealCategory, "category", "", "Default category (defaults to snacks)")
	cmd.Flags().StringVar(&savedMealNotes, "notes", "", "Notes")
}

func addSavedMealComponentFlags(cmd *cobra.Command) {
	cmd.Flags().StringVar(&savedMealComponentFood, "saved-food", "", "Optional saved food id|name to snapshot from")
	cmd.Flags().StringVar(&savedMealComponentName, "name", "", "Component name")
	cmd.Flags().Float64Var(&savedMealComponentQty, "quantity", 1, "Component quantity")
	cmd.Flags().StringVar(&savedMealComponentUnit, "unit", "serving", "Component unit")
	cmd.Flags().IntVar(&savedMealComponentCalories, "calories", 0, "Calories")
	cmd.Flags().Float64Var(&savedMealComponentProtein, "protein", 0, "Protein grams")
	cmd.Flags().Float64Var(&savedMealComponentCarbs, "carbs", 0, "Carbs grams")
	cmd.Flags().Float64Var(&savedMealComponentFat, "fat", 0, "Fat grams")
	cmd.Flags().Float64Var(&savedMealComponentFiber, "fiber", 0, "Fiber grams")
	cmd.Flags().Float64Var(&savedMealComponentSugar, "sugar", 0, "Sugar grams")
	cmd.Flags().Float64Var(&savedMealComponentSodium, "sodium", 0, "Sodium milligrams")
	cmd.Flags().StringVar(&savedMealComponentMicros, "micros-json", "", "Micronutrients JSON object")
	cmd.Flags().IntVar(&savedMealComponentPosition, "position", 0, "Optional component position")
}

func init() {
	rootCmd.AddCommand(savedMealCmd)
	savedMealCmd.AddCommand(savedMealAddCmd, savedMealAddFromEntryCmd, savedMealListCmd, savedMealShowCmd, savedMealUpdateCmd, savedMealArchiveCmd, savedMealRestoreCmd, savedMealLogCmd, savedMealComponentCmd)
	savedMealComponentCmd.AddCommand(savedMealComponentAddCmd, savedMealComponentListCmd, savedMealComponentUpdateCmd, savedMealComponentDeleteCmd)

	addSavedMealFlags(savedMealAddCmd)
	_ = savedMealAddCmd.MarkFlagRequired("name")

	savedMealAddFromEntryCmd.Flags().StringVar(&savedMealName, "name", "", "Saved meal name override")
	savedMealAddFromEntryCmd.Flags().StringVar(&savedMealFromEntryCompName, "component-name", "", "Initial component name override")

	savedMealListCmd.Flags().BoolVar(&savedMealIncludeArch, "include-archived", false, "Include archived saved meals")
	savedMealListCmd.Flags().IntVar(&savedMealLimit, "limit", 100, "Result limit")
	savedMealListCmd.Flags().StringVar(&savedMealQuery, "query", "", "Filter by name")

	addSavedMealFlags(savedMealUpdateCmd)
	_ = savedMealUpdateCmd.MarkFlagRequired("name")

	savedMealLogCmd.Flags().Float64Var(&savedMealServings, "servings", 1, "Serving multiplier")
	savedMealLogCmd.Flags().StringVar(&savedMealCategory, "category", "", "Optional category override")
	savedMealLogCmd.Flags().StringVar(&savedMealDate, "date", "", "Date in YYYY-MM-DD")
	savedMealLogCmd.Flags().StringVar(&savedMealTime, "time", "", "Time in HH:MM")
	savedMealLogCmd.Flags().StringVar(&savedMealNotes, "notes", "", "Optional notes")

	addSavedMealComponentFlags(savedMealComponentAddCmd)
	addSavedMealComponentFlags(savedMealComponentUpdateCmd)
	_ = savedMealComponentUpdateCmd.MarkFlagRequired("name")
}
