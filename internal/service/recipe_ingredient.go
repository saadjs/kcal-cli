package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/saad/kcal-cli/internal/model"
)

type RecipeIngredientInput struct {
	Name       string
	Amount     float64
	AmountUnit string
	Calories   int
	ProteinG   float64
	CarbsG     float64
	FatG       float64
}

func AddRecipeIngredient(db *sql.DB, recipeIdentifier string, in RecipeIngredientInput) (int64, error) {
	recipe, err := ResolveRecipe(db, recipeIdentifier)
	if err != nil {
		return 0, err
	}
	if err := validateRecipeIngredientInput(in); err != nil {
		return 0, err
	}
	res, err := db.Exec(`
INSERT INTO recipe_ingredients(recipe_id, name, amount, amount_unit, calories, protein_g, carbs_g, fat_g)
VALUES(?, ?, ?, ?, ?, ?, ?, ?)
`, recipe.ID, strings.TrimSpace(in.Name), in.Amount, strings.TrimSpace(in.AmountUnit), in.Calories, in.ProteinG, in.CarbsG, in.FatG)
	if err != nil {
		return 0, fmt.Errorf("add recipe ingredient: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve recipe ingredient id: %w", err)
	}
	return id, nil
}

func ListRecipeIngredients(db *sql.DB, recipeIdentifier string) ([]model.RecipeIngredient, error) {
	recipe, err := ResolveRecipe(db, recipeIdentifier)
	if err != nil {
		return nil, err
	}
	rows, err := db.Query(`
SELECT id, recipe_id, name, amount, amount_unit, calories, protein_g, carbs_g, fat_g, created_at, updated_at
FROM recipe_ingredients
WHERE recipe_id = ?
ORDER BY id ASC
`, recipe.ID)
	if err != nil {
		return nil, fmt.Errorf("list recipe ingredients: %w", err)
	}
	defer rows.Close()
	items := make([]model.RecipeIngredient, 0)
	for rows.Next() {
		var it model.RecipeIngredient
		if err := rows.Scan(&it.ID, &it.RecipeID, &it.Name, &it.Amount, &it.AmountUnit, &it.Calories, &it.ProteinG, &it.CarbsG, &it.FatG, &it.CreatedAt, &it.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan recipe ingredient: %w", err)
		}
		items = append(items, it)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipe ingredients: %w", err)
	}
	return items, nil
}

func UpdateRecipeIngredient(db *sql.DB, ingredientID int64, in RecipeIngredientInput) error {
	if ingredientID <= 0 {
		return fmt.Errorf("ingredient id must be > 0")
	}
	if err := validateRecipeIngredientInput(in); err != nil {
		return err
	}
	res, err := db.Exec(`
UPDATE recipe_ingredients
SET name = ?, amount = ?, amount_unit = ?, calories = ?, protein_g = ?, carbs_g = ?, fat_g = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, strings.TrimSpace(in.Name), in.Amount, strings.TrimSpace(in.AmountUnit), in.Calories, in.ProteinG, in.CarbsG, in.FatG, ingredientID)
	if err != nil {
		return fmt.Errorf("update recipe ingredient %d: %w", ingredientID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("recipe ingredient %d not found", ingredientID)
	}
	return nil
}

func DeleteRecipeIngredient(db *sql.DB, ingredientID int64) error {
	if ingredientID <= 0 {
		return fmt.Errorf("ingredient id must be > 0")
	}
	res, err := db.Exec(`DELETE FROM recipe_ingredients WHERE id = ?`, ingredientID)
	if err != nil {
		return fmt.Errorf("delete recipe ingredient %d: %w", ingredientID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("recipe ingredient %d not found", ingredientID)
	}
	return nil
}

func RecalculateRecipeTotals(db *sql.DB, recipeIdentifier string) error {
	recipe, err := ResolveRecipe(db, recipeIdentifier)
	if err != nil {
		return err
	}
	var calories int
	var protein, carbs, fat float64
	if err := db.QueryRow(`
SELECT COALESCE(SUM(calories), 0), COALESCE(SUM(protein_g), 0), COALESCE(SUM(carbs_g), 0), COALESCE(SUM(fat_g), 0)
FROM recipe_ingredients WHERE recipe_id = ?
`, recipe.ID).Scan(&calories, &protein, &carbs, &fat); err != nil {
		return fmt.Errorf("aggregate recipe ingredients: %w", err)
	}
	_, err = db.Exec(`
UPDATE recipes
SET calories_total = ?, protein_total_g = ?, carbs_total_g = ?, fat_total_g = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, calories, protein, carbs, fat, recipe.ID)
	if err != nil {
		return fmt.Errorf("update recipe totals: %w", err)
	}
	return nil
}

func validateRecipeIngredientInput(in RecipeIngredientInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("ingredient name is required")
	}
	if in.Amount <= 0 {
		return fmt.Errorf("ingredient amount must be > 0")
	}
	if strings.TrimSpace(in.AmountUnit) == "" {
		return fmt.Errorf("ingredient unit is required")
	}
	if err := validateNonNegativeInt("calories", in.Calories); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("protein", in.ProteinG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("carbs", in.CarbsG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("fat", in.FatG); err != nil {
		return err
	}
	return nil
}
