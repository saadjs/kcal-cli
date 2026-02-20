package service

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/saadjs/kcal-cli/internal/model"
)

type RecipeInput struct {
	Name          string
	CaloriesTotal int
	ProteinTotalG float64
	CarbsTotalG   float64
	FatTotalG     float64
	Servings      float64
	Notes         string
}

func CreateRecipe(db *sql.DB, in RecipeInput) (int64, error) {
	if err := validateRecipeInput(in); err != nil {
		return 0, err
	}
	res, err := db.Exec(`
INSERT INTO recipes(name, calories_total, protein_total_g, carbs_total_g, fat_total_g, servings, notes)
VALUES(?, ?, ?, ?, ?, ?, ?)
`, strings.TrimSpace(in.Name), in.CaloriesTotal, in.ProteinTotalG, in.CarbsTotalG, in.FatTotalG, in.Servings, strings.TrimSpace(in.Notes))
	if err != nil {
		return 0, fmt.Errorf("create recipe: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve recipe id: %w", err)
	}
	return id, nil
}

func ListRecipes(db *sql.DB) ([]model.Recipe, error) {
	rows, err := db.Query(`
SELECT id, name, calories_total, protein_total_g, carbs_total_g, fat_total_g, servings, IFNULL(notes,''), created_at, updated_at
FROM recipes
ORDER BY name
`)
	if err != nil {
		return nil, fmt.Errorf("list recipes: %w", err)
	}
	defer rows.Close()

	items := make([]model.Recipe, 0)
	for rows.Next() {
		var r model.Recipe
		if err := rows.Scan(&r.ID, &r.Name, &r.CaloriesTotal, &r.ProteinTotalG, &r.CarbsTotalG, &r.FatTotalG, &r.Servings, &r.Notes, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan recipe: %w", err)
		}
		items = append(items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate recipes: %w", err)
	}
	return items, nil
}

func ResolveRecipe(db *sql.DB, idOrName string) (*model.Recipe, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, fmt.Errorf("recipe identifier is required")
	}
	var row *sql.Row
	if id, err := parseIDLoose(idOrName); err == nil {
		row = db.QueryRow(`
SELECT id, name, calories_total, protein_total_g, carbs_total_g, fat_total_g, servings, IFNULL(notes,''), created_at, updated_at
FROM recipes WHERE id = ?
`, id)
	} else {
		row = db.QueryRow(`
SELECT id, name, calories_total, protein_total_g, carbs_total_g, fat_total_g, servings, IFNULL(notes,''), created_at, updated_at
FROM recipes WHERE LOWER(name) = ?
`, strings.ToLower(idOrName))
	}
	var r model.Recipe
	if err := row.Scan(&r.ID, &r.Name, &r.CaloriesTotal, &r.ProteinTotalG, &r.CarbsTotalG, &r.FatTotalG, &r.Servings, &r.Notes, &r.CreatedAt, &r.UpdatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("recipe %q not found", idOrName)
		}
		return nil, fmt.Errorf("resolve recipe %q: %w", idOrName, err)
	}
	return &r, nil
}

func UpdateRecipe(db *sql.DB, idOrName string, in RecipeInput) error {
	if err := validateRecipeInput(in); err != nil {
		return err
	}
	recipe, err := ResolveRecipe(db, idOrName)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
UPDATE recipes SET
  name = ?, calories_total = ?, protein_total_g = ?, carbs_total_g = ?, fat_total_g = ?, servings = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, strings.TrimSpace(in.Name), in.CaloriesTotal, in.ProteinTotalG, in.CarbsTotalG, in.FatTotalG, in.Servings, strings.TrimSpace(in.Notes), recipe.ID)
	if err != nil {
		return fmt.Errorf("update recipe %q: %w", idOrName, err)
	}
	return nil
}

func DeleteRecipe(db *sql.DB, idOrName string) error {
	recipe, err := ResolveRecipe(db, idOrName)
	if err != nil {
		return err
	}
	_, err = db.Exec(`DELETE FROM recipes WHERE id = ?`, recipe.ID)
	if err != nil {
		return fmt.Errorf("delete recipe %q: %w", idOrName, err)
	}
	return nil
}

type LogRecipeInput struct {
	RecipeIdentifier string
	Servings         float64
	Category         string
	ConsumedAt       time.Time
	Notes            string
}

func LogRecipe(db *sql.DB, in LogRecipeInput) (int64, error) {
	if in.Servings <= 0 {
		return 0, fmt.Errorf("servings must be > 0")
	}
	recipe, err := ResolveRecipe(db, in.RecipeIdentifier)
	if err != nil {
		return 0, err
	}
	if recipe.Servings <= 0 {
		return 0, fmt.Errorf("recipe %q has invalid servings", recipe.Name)
	}
	factor := in.Servings / recipe.Servings
	calories := int(float64(recipe.CaloriesTotal) * factor)
	protein := recipe.ProteinTotalG * factor
	carbs := recipe.CarbsTotalG * factor
	fat := recipe.FatTotalG * factor

	if in.ConsumedAt.IsZero() {
		in.ConsumedAt = time.Now()
	}
	sourceID := recipe.ID
	entry := CreateEntryInput{
		Name:       fmt.Sprintf("%s (%.2f servings)", recipe.Name, in.Servings),
		Calories:   calories,
		ProteinG:   protein,
		CarbsG:     carbs,
		FatG:       fat,
		Category:   in.Category,
		Consumed:   in.ConsumedAt,
		Notes:      strings.TrimSpace(in.Notes),
		SourceType: "recipe",
		SourceID:   &sourceID,
	}
	return CreateEntry(db, entry)
}

func validateRecipeInput(in RecipeInput) error {
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("recipe name is required")
	}
	if err := validateNonNegativeInt("calories", in.CaloriesTotal); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("protein", in.ProteinTotalG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("carbs", in.CarbsTotalG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("fat", in.FatTotalG); err != nil {
		return err
	}
	if in.Servings <= 0 {
		return fmt.Errorf("servings must be > 0")
	}
	return nil
}

func parseIDLoose(value string) (int64, error) {
	id, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("not numeric")
	}
	return id, nil
}
