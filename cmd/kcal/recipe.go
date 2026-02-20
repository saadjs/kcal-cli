package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saad/kcal-cli/internal/service"
	"github.com/spf13/cobra"
)

var recipeCmd = &cobra.Command{
	Use:   "recipe",
	Short: "Manage recipes",
}

var (
	recipeName     string
	recipeCalories int
	recipeProtein  float64
	recipeCarbs    float64
	recipeFat      float64
	recipeServings float64
	recipeNotes    string
)

var recipeAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Create a recipe",
	RunE: func(cmd *cobra.Command, args []string) error {
		in := service.RecipeInput{
			Name:          recipeName,
			CaloriesTotal: recipeCalories,
			ProteinTotalG: recipeProtein,
			CarbsTotalG:   recipeCarbs,
			FatTotalG:     recipeFat,
			Servings:      recipeServings,
			Notes:         recipeNotes,
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.CreateRecipe(sqldb, in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Created recipe %d\n", id)
			return nil
		})
	},
}

var recipeListCmd = &cobra.Command{
	Use:   "list",
	Short: "List recipes",
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			recipes, err := service.ListRecipes(sqldb)
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tNAME\tKCAL\tP\tC\tF\tSERVINGS")
			for _, r := range recipes {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%d\t%.1f\t%.1f\t%.1f\t%.2f\n", r.ID, r.Name, r.CaloriesTotal, r.ProteinTotalG, r.CarbsTotalG, r.FatTotalG, r.Servings)
			}
			return nil
		})
	},
}

var recipeShowCmd = &cobra.Command{
	Use:   "show <id|name>",
	Short: "Show recipe details",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			r, err := service.ResolveRecipe(sqldb, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "ID: %d\nName: %s\nCalories Total: %d\nProtein Total: %.1fg\nCarbs Total: %.1fg\nFat Total: %.1fg\nServings: %.2f\nNotes: %s\n", r.ID, r.Name, r.CaloriesTotal, r.ProteinTotalG, r.CarbsTotalG, r.FatTotalG, r.Servings, r.Notes)
			return nil
		})
	},
}

var recipeUpdateCmd = &cobra.Command{
	Use:   "update <id|name>",
	Short: "Update a recipe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		in := service.RecipeInput{
			Name:          recipeName,
			CaloriesTotal: recipeCalories,
			ProteinTotalG: recipeProtein,
			CarbsTotalG:   recipeCarbs,
			FatTotalG:     recipeFat,
			Servings:      recipeServings,
			Notes:         recipeNotes,
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateRecipe(sqldb, args[0], in); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated recipe %q\n", args[0])
			return nil
		})
	},
}

var recipeDeleteCmd = &cobra.Command{
	Use:   "delete <id|name>",
	Short: "Delete a recipe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteRecipe(sqldb, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted recipe %q\n", args[0])
			return nil
		})
	},
}

var (
	logRecipeServings float64
	logRecipeCategory string
	logRecipeDate     string
	logRecipeTime     string
	logRecipeNotes    string
)

var recipeLogCmd = &cobra.Command{
	Use:   "log <id|name>",
	Short: "Log a recipe as an entry by servings",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		consumed, err := parseDateTimeOrNow(logRecipeDate, logRecipeTime)
		if err != nil {
			return err
		}
		in := service.LogRecipeInput{
			RecipeIdentifier: args[0],
			Servings:         logRecipeServings,
			Category:         logRecipeCategory,
			ConsumedAt:       consumed,
			Notes:            logRecipeNotes,
		}
		return withDB(func(sqldb *sql.DB) error {
			entryID, err := service.LogRecipe(sqldb, in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Logged recipe %q as entry %d\n", args[0], entryID)
			return nil
		})
	},
}

func bindRecipeFields(cmd *cobra.Command) {
	cmd.Flags().StringVar(&recipeName, "name", "", "Recipe name")
	cmd.Flags().IntVar(&recipeCalories, "calories", 0, "Total calories")
	cmd.Flags().Float64Var(&recipeProtein, "protein", 0, "Total protein grams")
	cmd.Flags().Float64Var(&recipeCarbs, "carbs", 0, "Total carbs grams")
	cmd.Flags().Float64Var(&recipeFat, "fat", 0, "Total fat grams")
	cmd.Flags().Float64Var(&recipeServings, "servings", 0, "Total recipe servings")
	cmd.Flags().StringVar(&recipeNotes, "notes", "", "Recipe notes")
}

func init() {
	rootCmd.AddCommand(recipeCmd)
	recipeCmd.AddCommand(recipeAddCmd, recipeListCmd, recipeShowCmd, recipeUpdateCmd, recipeDeleteCmd, recipeLogCmd)

	bindRecipeFields(recipeAddCmd)
	_ = recipeAddCmd.MarkFlagRequired("name")
	_ = recipeAddCmd.MarkFlagRequired("calories")
	_ = recipeAddCmd.MarkFlagRequired("protein")
	_ = recipeAddCmd.MarkFlagRequired("carbs")
	_ = recipeAddCmd.MarkFlagRequired("fat")
	_ = recipeAddCmd.MarkFlagRequired("servings")

	bindRecipeFields(recipeUpdateCmd)
	_ = recipeUpdateCmd.MarkFlagRequired("name")
	_ = recipeUpdateCmd.MarkFlagRequired("calories")
	_ = recipeUpdateCmd.MarkFlagRequired("protein")
	_ = recipeUpdateCmd.MarkFlagRequired("carbs")
	_ = recipeUpdateCmd.MarkFlagRequired("fat")
	_ = recipeUpdateCmd.MarkFlagRequired("servings")

	recipeLogCmd.Flags().Float64Var(&logRecipeServings, "servings", 0, "Servings to log")
	recipeLogCmd.Flags().StringVar(&logRecipeCategory, "category", "", "Category name")
	recipeLogCmd.Flags().StringVar(&logRecipeDate, "date", "", "Date in YYYY-MM-DD")
	recipeLogCmd.Flags().StringVar(&logRecipeTime, "time", "", "Time in HH:MM")
	recipeLogCmd.Flags().StringVar(&logRecipeNotes, "notes", "", "Optional notes")
	_ = recipeLogCmd.MarkFlagRequired("servings")
	_ = recipeLogCmd.MarkFlagRequired("category")
}
