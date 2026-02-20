package kcal

import (
	"database/sql"
	"fmt"

	"github.com/saadjs/kcal-cli/internal/service"
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

var recipeRecalcCmd = &cobra.Command{
	Use:   "recalc <id|name>",
	Short: "Recalculate recipe totals from ingredients",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			if err := service.RecalculateRecipeTotals(sqldb, args[0]); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Recalculated recipe %q totals\n", args[0])
			return nil
		})
	},
}

var recipeIngredientCmd = &cobra.Command{
	Use:   "ingredient",
	Short: "Manage recipe ingredients",
}

var (
	ingredientName     string
	ingredientAmount   float64
	ingredientUnit     string
	ingredientCalories int
	ingredientProtein  float64
	ingredientCarbs    float64
	ingredientFat      float64
	refAmount          float64
	refUnit            string
	refCalories        int
	refProtein         float64
	refCarbs           float64
	refFat             float64
	densityGPerML      float64
)

var recipeIngredientAddCmd = &cobra.Command{
	Use:   "add <recipe-id|name>",
	Short: "Add ingredient to recipe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		in, err := buildRecipeIngredientInput(cmd)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			id, err := service.AddRecipeIngredient(sqldb, args[0], in)
			if err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Added ingredient %d\n", id)
			return nil
		})
	},
}

var recipeIngredientListCmd = &cobra.Command{
	Use:   "list <recipe-id|name>",
	Short: "List ingredients for a recipe",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return withDB(func(sqldb *sql.DB) error {
			items, err := service.ListRecipeIngredients(sqldb, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), "ID\tNAME\tAMOUNT\tUNIT\tKCAL\tP\tC\tF")
			for _, it := range items {
				fmt.Fprintf(cmd.OutOrStdout(), "%d\t%s\t%.2f\t%s\t%d\t%.1f\t%.1f\t%.1f\n", it.ID, it.Name, it.Amount, it.AmountUnit, it.Calories, it.ProteinG, it.CarbsG, it.FatG)
			}
			return nil
		})
	},
}

var recipeIngredientUpdateCmd = &cobra.Command{
	Use:   "update <ingredient-id>",
	Short: "Update recipe ingredient",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("ingredient id", args[0])
		if err != nil {
			return err
		}
		in, err := buildRecipeIngredientInput(cmd)
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.UpdateRecipeIngredient(sqldb, id, in); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Updated ingredient %d\n", id)
			return nil
		})
	},
}

var recipeIngredientDeleteCmd = &cobra.Command{
	Use:   "delete <ingredient-id>",
	Short: "Delete recipe ingredient",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := parseInt64Arg("ingredient id", args[0])
		if err != nil {
			return err
		}
		return withDB(func(sqldb *sql.DB) error {
			if err := service.DeleteRecipeIngredient(sqldb, id); err != nil {
				return err
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Deleted ingredient %d\n", id)
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

func buildRecipeIngredientInput(cmd *cobra.Command) (service.RecipeIngredientInput, error) {
	manualInput := service.RecipeIngredientInput{
		Name:       ingredientName,
		Amount:     ingredientAmount,
		AmountUnit: ingredientUnit,
		Calories:   ingredientCalories,
		ProteinG:   ingredientProtein,
		CarbsG:     ingredientCarbs,
		FatG:       ingredientFat,
	}

	hasRefMode := cmd.Flags().Changed("ref-amount") ||
		cmd.Flags().Changed("ref-unit") ||
		cmd.Flags().Changed("ref-calories") ||
		cmd.Flags().Changed("ref-protein") ||
		cmd.Flags().Changed("ref-carbs") ||
		cmd.Flags().Changed("ref-fat")

	if !hasRefMode {
		if !cmd.Flags().Changed("calories") ||
			!cmd.Flags().Changed("protein") ||
			!cmd.Flags().Changed("carbs") ||
			!cmd.Flags().Changed("fat") {
			return service.RecipeIngredientInput{}, fmt.Errorf("manual mode requires --calories --protein --carbs --fat")
		}
		return manualInput, nil
	}

	if cmd.Flags().Changed("calories") ||
		cmd.Flags().Changed("protein") ||
		cmd.Flags().Changed("carbs") ||
		cmd.Flags().Changed("fat") {
		return service.RecipeIngredientInput{}, fmt.Errorf("cannot combine manual macro flags with reference scaling flags")
	}

	if !cmd.Flags().Changed("ref-amount") ||
		!cmd.Flags().Changed("ref-unit") ||
		!cmd.Flags().Changed("ref-calories") ||
		!cmd.Flags().Changed("ref-protein") ||
		!cmd.Flags().Changed("ref-carbs") ||
		!cmd.Flags().Changed("ref-fat") {
		return service.RecipeIngredientInput{}, fmt.Errorf("reference mode requires --ref-amount --ref-unit --ref-calories --ref-protein --ref-carbs --ref-fat")
	}

	scaled, err := service.ScaleIngredientMacros(service.ScaleIngredientMacrosInput{
		Amount:      ingredientAmount,
		Unit:        ingredientUnit,
		RefAmount:   refAmount,
		RefUnit:     refUnit,
		RefCalories: refCalories,
		RefProteinG: refProtein,
		RefCarbsG:   refCarbs,
		RefFatG:     refFat,
		DensityGML:  densityGPerML,
	})
	if err != nil {
		return service.RecipeIngredientInput{}, err
	}

	manualInput.Calories = scaled.Calories
	manualInput.ProteinG = scaled.ProteinG
	manualInput.CarbsG = scaled.CarbsG
	manualInput.FatG = scaled.FatG
	return manualInput, nil
}

func init() {
	rootCmd.AddCommand(recipeCmd)
	recipeCmd.AddCommand(recipeAddCmd, recipeListCmd, recipeShowCmd, recipeUpdateCmd, recipeDeleteCmd, recipeRecalcCmd, recipeLogCmd, recipeIngredientCmd)
	recipeIngredientCmd.AddCommand(recipeIngredientAddCmd, recipeIngredientListCmd, recipeIngredientUpdateCmd, recipeIngredientDeleteCmd)

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

	for _, c := range []*cobra.Command{recipeIngredientAddCmd, recipeIngredientUpdateCmd} {
		c.Flags().StringVar(&ingredientName, "name", "", "Ingredient name")
		c.Flags().Float64Var(&ingredientAmount, "amount", 0, "Ingredient amount")
		c.Flags().StringVar(&ingredientUnit, "unit", "", "Ingredient unit")
		c.Flags().IntVar(&ingredientCalories, "calories", 0, "Ingredient calories")
		c.Flags().Float64Var(&ingredientProtein, "protein", 0, "Ingredient protein grams")
		c.Flags().Float64Var(&ingredientCarbs, "carbs", 0, "Ingredient carbs grams")
		c.Flags().Float64Var(&ingredientFat, "fat", 0, "Ingredient fat grams")
		c.Flags().Float64Var(&refAmount, "ref-amount", 0, "Reference amount used for scaling")
		c.Flags().StringVar(&refUnit, "ref-unit", "", "Reference unit used for scaling")
		c.Flags().IntVar(&refCalories, "ref-calories", 0, "Reference calories for ref amount")
		c.Flags().Float64Var(&refProtein, "ref-protein", 0, "Reference protein grams for ref amount")
		c.Flags().Float64Var(&refCarbs, "ref-carbs", 0, "Reference carbs grams for ref amount")
		c.Flags().Float64Var(&refFat, "ref-fat", 0, "Reference fat grams for ref amount")
		c.Flags().Float64Var(&densityGPerML, "density-g-per-ml", 0, "Density for mass/volume conversion when scaling")
		_ = c.MarkFlagRequired("name")
		_ = c.MarkFlagRequired("amount")
		_ = c.MarkFlagRequired("unit")
	}
}
