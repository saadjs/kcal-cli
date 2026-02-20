package service

import (
	"database/sql"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/model"
)

type CreateSavedMealInput struct {
	Name     string
	Category string
	Notes    string
}

type UpdateSavedMealInput struct {
	Name     string
	Category string
	Notes    string
}

type ListSavedMealsFilter struct {
	IncludeArchived bool
	Limit           int
	Query           string
}

type SavedMealComponentInput struct {
	SavedFoodIdentifier string
	Name                string
	Quantity            float64
	Unit                string
	Calories            int
	ProteinG            float64
	CarbsG              float64
	FatG                float64
	FiberG              float64
	SugarG              float64
	SodiumMg            float64
	Micros              string
	Position            int
}

type UpdateSavedMealComponentInput struct {
	Name     string
	Quantity float64
	Unit     string
	Calories int
	ProteinG float64
	CarbsG   float64
	FatG     float64
	FiberG   float64
	SugarG   float64
	SodiumMg float64
	Micros   string
	Position int
}

type LogSavedMealInput struct {
	Identifier string
	Servings   float64
	Category   string
	ConsumedAt time.Time
	Notes      string
}

func CreateSavedMeal(db *sql.DB, in CreateSavedMealInput) (int64, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return 0, fmt.Errorf("saved meal name is required")
	}
	catID, err := resolveCategoryIDWithDefault(db, in.Category)
	if err != nil {
		return 0, err
	}
	res, err := db.Exec(`
INSERT INTO saved_meals(name, name_norm, default_category_id, notes)
VALUES(?, ?, ?, ?)
`, name, normalizeName(name), catID, strings.TrimSpace(in.Notes))
	if err != nil {
		return 0, fmt.Errorf("create saved meal: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve saved meal id: %w", err)
	}
	return id, nil
}

func CreateSavedMealFromEntry(db *sql.DB, entryID int64, mealName, componentName string) (int64, error) {
	e, err := EntryByID(db, entryID)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(mealName) == "" {
		mealName = e.Name
	}
	if strings.TrimSpace(componentName) == "" {
		componentName = e.Name
	}
	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	catID, err := resolveCategoryIDWithDefault(db, e.Category)
	if err != nil {
		return 0, err
	}
	res, err := tx.Exec(`INSERT INTO saved_meals(name, name_norm, default_category_id, notes) VALUES(?, ?, ?, ?)`, strings.TrimSpace(mealName), normalizeName(mealName), catID, "")
	if err != nil {
		return 0, fmt.Errorf("create saved meal from entry: %w", err)
	}
	mealID, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve saved meal id: %w", err)
	}
	if _, err := tx.Exec(`
INSERT INTO saved_meal_components(saved_meal_id, position, name, quantity, unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, micronutrients_json)
VALUES(?, 1, ?, 1, 'serving', ?, ?, ?, ?, ?, ?, ?, ?)
`, mealID, strings.TrimSpace(componentName), e.Calories, e.ProteinG, e.CarbsG, e.FatG, e.FiberG, e.SugarG, e.SodiumMg, e.Micronutrients); err != nil {
		return 0, fmt.Errorf("create saved meal component from entry: %w", err)
	}
	if err := recalcSavedMealTotalsTx(tx, mealID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit saved meal from entry: %w", err)
	}
	return mealID, nil
}

func ResolveSavedMeal(db *sql.DB, idOrName string) (*model.SavedMeal, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, fmt.Errorf("saved meal identifier is required")
	}
	var row *sql.Row
	if id, err := parseIDLoose(idOrName); err == nil {
		row = db.QueryRow(savedMealSelectBase()+` WHERE sm.id = ?`, id)
	} else {
		row = db.QueryRow(savedMealSelectBase()+` WHERE sm.name_norm = ?`, normalizeName(idOrName))
	}
	item, err := scanSavedMeal(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("saved meal %q not found", idOrName)
	}
	if err != nil {
		return nil, fmt.Errorf("resolve saved meal %q: %w", idOrName, err)
	}
	return item, nil
}

func ListSavedMeals(db *sql.DB, f ListSavedMealsFilter) ([]model.SavedMeal, error) {
	query := savedMealSelectBase() + ` WHERE 1=1`
	args := make([]any, 0, 4)
	if !f.IncludeArchived {
		query += ` AND sm.archived_at IS NULL`
	}
	if strings.TrimSpace(f.Query) != "" {
		query += ` AND sm.name_norm LIKE ?`
		args = append(args, "%"+normalizeName(f.Query)+"%")
	}
	query += ` ORDER BY sm.usage_count DESC, sm.last_used_at DESC, sm.updated_at DESC, sm.name ASC`
	if f.Limit <= 0 {
		f.Limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, f.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list saved meals: %w", err)
	}
	defer rows.Close()
	out := make([]model.SavedMeal, 0)
	for rows.Next() {
		item, err := scanSavedMealRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate saved meals: %w", err)
	}
	return out, nil
}

func UpdateSavedMeal(db *sql.DB, idOrName string, in UpdateSavedMealInput) error {
	item, err := ResolveSavedMeal(db, idOrName)
	if err != nil {
		return err
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("saved meal name is required")
	}
	catID, err := resolveCategoryIDWithDefault(db, in.Category)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
UPDATE saved_meals
SET name = ?, name_norm = ?, default_category_id = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, strings.TrimSpace(in.Name), normalizeName(in.Name), catID, strings.TrimSpace(in.Notes), item.ID)
	if err != nil {
		return fmt.Errorf("update saved meal %q: %w", idOrName, err)
	}
	return nil
}

func ArchiveSavedMeal(db *sql.DB, idOrName string) error {
	item, err := ResolveSavedMeal(db, idOrName)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE saved_meals SET archived_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, item.ID)
	if err != nil {
		return fmt.Errorf("archive saved meal %q: %w", idOrName, err)
	}
	return nil
}

func RestoreSavedMeal(db *sql.DB, idOrName string) error {
	item, err := ResolveSavedMeal(db, idOrName)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE saved_meals SET archived_at = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, item.ID)
	if err != nil {
		return fmt.Errorf("restore saved meal %q: %w", idOrName, err)
	}
	return nil
}

func AddSavedMealComponent(db *sql.DB, mealIdentifier string, in SavedMealComponentInput) (int64, error) {
	meal, err := ResolveSavedMeal(db, mealIdentifier)
	if err != nil {
		return 0, err
	}
	if meal.ArchivedAt != nil {
		return 0, fmt.Errorf("saved meal %q is archived", meal.Name)
	}
	if in.Quantity <= 0 {
		in.Quantity = 1
	}
	if strings.TrimSpace(in.Unit) == "" {
		in.Unit = "serving"
	}

	var savedFoodID any
	if strings.TrimSpace(in.SavedFoodIdentifier) != "" {
		food, err := ResolveSavedFood(db, in.SavedFoodIdentifier)
		if err != nil {
			return 0, err
		}
		if food.ArchivedAt != nil {
			return 0, fmt.Errorf("saved food %q is archived", food.Name)
		}
		savedFoodID = food.ID
		if strings.TrimSpace(in.Name) == "" {
			in.Name = food.Name
		}
		if strings.TrimSpace(in.Unit) == "serving" && strings.TrimSpace(food.ServingUnit) != "" {
			in.Unit = food.ServingUnit
		}
		if in.Quantity == 1 && food.ServingAmount > 0 {
			in.Quantity = food.ServingAmount
		}
		in.Calories = food.Calories
		in.ProteinG = food.ProteinG
		in.CarbsG = food.CarbsG
		in.FatG = food.FatG
		in.FiberG = food.FiberG
		in.SugarG = food.SugarG
		in.SodiumMg = food.SodiumMg
		in.Micros = food.Micronutrients
	} else {
		savedFoodID = nil
		if strings.TrimSpace(in.Name) == "" {
			return 0, fmt.Errorf("component name is required")
		}
		if err := validateNonNegativeInt("calories", in.Calories); err != nil {
			return 0, err
		}
		if err := validateNonNegativeFloat("protein", in.ProteinG); err != nil {
			return 0, err
		}
		if err := validateNonNegativeFloat("carbs", in.CarbsG); err != nil {
			return 0, err
		}
		if err := validateNonNegativeFloat("fat", in.FatG); err != nil {
			return 0, err
		}
		if err := validateNonNegativeFloat("fiber", in.FiberG); err != nil {
			return 0, err
		}
		if err := validateNonNegativeFloat("sugar", in.SugarG); err != nil {
			return 0, err
		}
		if err := validateNonNegativeFloat("sodium", in.SodiumMg); err != nil {
			return 0, err
		}
	}
	micros, err := normalizeMicronutrientsJSON(in.Micros)
	if err != nil {
		return 0, err
	}
	if in.Position <= 0 {
		_ = db.QueryRow(`SELECT COALESCE(MAX(position), 0) + 1 FROM saved_meal_components WHERE saved_meal_id = ?`, meal.ID).Scan(&in.Position)
	}

	tx, err := db.Begin()
	if err != nil {
		return 0, fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()
	res, err := tx.Exec(`
INSERT INTO saved_meal_components(saved_meal_id, saved_food_id, position, name, quantity, unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, micronutrients_json)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, meal.ID, savedFoodID, in.Position, strings.TrimSpace(in.Name), in.Quantity, strings.TrimSpace(in.Unit), in.Calories, in.ProteinG, in.CarbsG, in.FatG, in.FiberG, in.SugarG, in.SodiumMg, micros)
	if err != nil {
		return 0, fmt.Errorf("add saved meal component: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve saved meal component id: %w", err)
	}
	if err := recalcSavedMealTotalsTx(tx, meal.ID); err != nil {
		return 0, err
	}
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("commit add component: %w", err)
	}
	return id, nil
}

func ListSavedMealComponents(db *sql.DB, mealIdentifier string) ([]model.SavedMealComponent, error) {
	meal, err := ResolveSavedMeal(db, mealIdentifier)
	if err != nil {
		return nil, err
	}
	return listSavedMealComponentsByID(db, meal.ID)
}

func UpdateSavedMealComponent(db *sql.DB, componentID int64, in UpdateSavedMealComponentInput) error {
	if componentID <= 0 {
		return fmt.Errorf("component id must be > 0")
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("component name is required")
	}
	if in.Quantity <= 0 {
		return fmt.Errorf("component quantity must be > 0")
	}
	if strings.TrimSpace(in.Unit) == "" {
		return fmt.Errorf("component unit is required")
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
	if err := validateNonNegativeFloat("fiber", in.FiberG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("sugar", in.SugarG); err != nil {
		return err
	}
	if err := validateNonNegativeFloat("sodium", in.SodiumMg); err != nil {
		return err
	}
	micros, err := normalizeMicronutrientsJSON(in.Micros)
	if err != nil {
		return err
	}

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var mealID int64
	if err := tx.QueryRow(`SELECT saved_meal_id FROM saved_meal_components WHERE id = ?`, componentID).Scan(&mealID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("saved meal component %d not found", componentID)
		}
		return fmt.Errorf("resolve meal for component %d: %w", componentID, err)
	}

	res, err := tx.Exec(`
UPDATE saved_meal_components
SET name = ?, quantity = ?, unit = ?, calories = ?, protein_g = ?, carbs_g = ?, fat_g = ?, fiber_g = ?, sugar_g = ?, sodium_mg = ?, micronutrients_json = ?, position = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, strings.TrimSpace(in.Name), in.Quantity, strings.TrimSpace(in.Unit), in.Calories, in.ProteinG, in.CarbsG, in.FatG, in.FiberG, in.SugarG, in.SodiumMg, micros, in.Position, componentID)
	if err != nil {
		return fmt.Errorf("update component %d: %w", componentID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for component %d: %w", componentID, err)
	}
	if affected == 0 {
		return fmt.Errorf("saved meal component %d not found", componentID)
	}
	if err := recalcSavedMealTotalsTx(tx, mealID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update component: %w", err)
	}
	return nil
}

func DeleteSavedMealComponent(db *sql.DB, componentID int64) error {
	if componentID <= 0 {
		return fmt.Errorf("component id must be > 0")
	}
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var mealID int64
	if err := tx.QueryRow(`SELECT saved_meal_id FROM saved_meal_components WHERE id = ?`, componentID).Scan(&mealID); err != nil {
		if err == sql.ErrNoRows {
			return fmt.Errorf("saved meal component %d not found", componentID)
		}
		return fmt.Errorf("resolve meal for component %d: %w", componentID, err)
	}
	res, err := tx.Exec(`DELETE FROM saved_meal_components WHERE id = ?`, componentID)
	if err != nil {
		return fmt.Errorf("delete component %d: %w", componentID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected for component %d: %w", componentID, err)
	}
	if affected == 0 {
		return fmt.Errorf("saved meal component %d not found", componentID)
	}
	if err := recalcSavedMealTotalsTx(tx, mealID); err != nil {
		return err
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete component: %w", err)
	}
	return nil
}

func LogSavedMeal(db *sql.DB, in LogSavedMealInput) (int64, error) {
	if in.Servings <= 0 {
		in.Servings = 1
	}
	meal, err := ResolveSavedMeal(db, in.Identifier)
	if err != nil {
		return 0, err
	}
	if meal.ArchivedAt != nil {
		return 0, fmt.Errorf("saved meal %q is archived", meal.Name)
	}
	if in.ConsumedAt.IsZero() {
		in.ConsumedAt = time.Now()
	}
	category := strings.TrimSpace(in.Category)
	if category == "" {
		category = meal.DefaultCategory
	}
	components, err := listSavedMealComponentsByID(db, meal.ID)
	if err != nil {
		return 0, err
	}
	if len(components) == 0 {
		return 0, fmt.Errorf("saved meal %q has no components", meal.Name)
	}
	calories := 0.0
	protein := 0.0
	carbs := 0.0
	fat := 0.0
	fiber := 0.0
	sugar := 0.0
	sodium := 0.0
	mergedMicros := Micronutrients{}
	for _, c := range components {
		calories += float64(c.Calories)
		protein += c.ProteinG
		carbs += c.CarbsG
		fat += c.FatG
		fiber += c.FiberG
		sugar += c.SugarG
		sodium += c.SodiumMg
		m, err := ParseMicronutrientsJSON(c.Micronutrients)
		if err != nil {
			return 0, err
		}
		for k, v := range m {
			if existing, ok := mergedMicros[k]; ok && existing.Unit == v.Unit {
				existing.Value += v.Value
				mergedMicros[k] = existing
				continue
			}
			mergedMicros[k] = v
		}
	}
	calories *= in.Servings
	protein *= in.Servings
	carbs *= in.Servings
	fat *= in.Servings
	fiber *= in.Servings
	sugar *= in.Servings
	sodium *= in.Servings
	microsJSON, err := EncodeMicronutrientsJSON(ScaleMicronutrients(mergedMicros, in.Servings))
	if err != nil {
		return 0, err
	}
	sourceID := meal.ID
	entryID, err := CreateEntry(db, CreateEntryInput{
		Name:           fmt.Sprintf("%s (saved meal x%.2f)", meal.Name, in.Servings),
		Calories:       int(math.Round(calories)),
		ProteinG:       protein,
		CarbsG:         carbs,
		FatG:           fat,
		FiberG:         fiber,
		SugarG:         sugar,
		SodiumMg:       sodium,
		Micronutrients: microsJSON,
		Category:       category,
		Consumed:       in.ConsumedAt,
		Notes:          strings.TrimSpace(in.Notes),
		SourceType:     "saved_meal",
		SourceID:       &sourceID,
	})
	if err != nil {
		return 0, err
	}
	if _, err := db.Exec(`UPDATE saved_meals SET usage_count = usage_count + 1, last_used_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, meal.ID); err != nil {
		return 0, fmt.Errorf("update saved meal usage: %w", err)
	}
	return entryID, nil
}

func RecalcSavedMealTotals(db *sql.DB, mealIdentifier string) error {
	meal, err := ResolveSavedMeal(db, mealIdentifier)
	if err != nil {
		return err
	}
	return recalcSavedMealTotalsTx(db, meal.ID)
}

func recalcSavedMealTotalsTx(exec sqlExecutor, mealID int64) error {
	rows, err := exec.Query(`SELECT calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, IFNULL(micronutrients_json,'') FROM saved_meal_components WHERE saved_meal_id = ?`, mealID)
	if err != nil {
		return fmt.Errorf("list saved meal components for recalc: %w", err)
	}
	defer rows.Close()

	totalCalories := 0
	totalProtein := 0.0
	totalCarbs := 0.0
	totalFat := 0.0
	totalFiber := 0.0
	totalSugar := 0.0
	totalSodium := 0.0
	microsTotals := Micronutrients{}
	for rows.Next() {
		var calories int
		var protein, carbs, fat, fiber, sugar, sodium float64
		var microsRaw string
		if err := rows.Scan(&calories, &protein, &carbs, &fat, &fiber, &sugar, &sodium, &microsRaw); err != nil {
			return fmt.Errorf("scan saved meal component for recalc: %w", err)
		}
		totalCalories += calories
		totalProtein += protein
		totalCarbs += carbs
		totalFat += fat
		totalFiber += fiber
		totalSugar += sugar
		totalSodium += sodium
		micros, err := ParseMicronutrientsJSON(microsRaw)
		if err != nil {
			return err
		}
		for k, v := range micros {
			if existing, ok := microsTotals[k]; ok && existing.Unit == v.Unit {
				existing.Value += v.Value
				microsTotals[k] = existing
				continue
			}
			microsTotals[k] = v
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("iterate saved meal components for recalc: %w", err)
	}
	microsJSON, err := EncodeMicronutrientsJSON(microsTotals)
	if err != nil {
		return err
	}
	if _, err := exec.Exec(`
UPDATE saved_meals
SET calories_total = ?, protein_total_g = ?, carbs_total_g = ?, fat_total_g = ?, fiber_total_g = ?, sugar_total_g = ?, sodium_total_mg = ?, micronutrients_json = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, totalCalories, totalProtein, totalCarbs, totalFat, totalFiber, totalSugar, totalSodium, microsJSON, mealID); err != nil {
		return fmt.Errorf("update saved meal totals: %w", err)
	}
	return nil
}

type sqlExecutor interface {
	Exec(query string, args ...any) (sql.Result, error)
	Query(query string, args ...any) (*sql.Rows, error)
}

func listSavedMealComponentsByID(db *sql.DB, mealID int64) ([]model.SavedMealComponent, error) {
	rows, err := db.Query(`
SELECT id, saved_meal_id, saved_food_id, position, name, quantity, unit, calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, IFNULL(micronutrients_json,''), created_at, updated_at
FROM saved_meal_components
WHERE saved_meal_id = ?
ORDER BY position ASC, id ASC
`, mealID)
	if err != nil {
		return nil, fmt.Errorf("list saved meal components: %w", err)
	}
	defer rows.Close()
	items := make([]model.SavedMealComponent, 0)
	for rows.Next() {
		var item model.SavedMealComponent
		var savedFoodID sql.NullInt64
		if err := rows.Scan(&item.ID, &item.SavedMealID, &savedFoodID, &item.Position, &item.Name, &item.Quantity, &item.Unit, &item.Calories, &item.ProteinG, &item.CarbsG, &item.FatG, &item.FiberG, &item.SugarG, &item.SodiumMg, &item.Micronutrients, &item.CreatedAt, &item.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan saved meal component: %w", err)
		}
		if savedFoodID.Valid {
			v := savedFoodID.Int64
			item.SavedFoodID = &v
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate saved meal components: %w", err)
	}
	return items, nil
}

func savedMealSelectBase() string {
	return `
SELECT sm.id, sm.name, sm.name_norm, sm.default_category_id, c.name, IFNULL(sm.notes,''),
       sm.calories_total, sm.protein_total_g, sm.carbs_total_g, sm.fat_total_g, sm.fiber_total_g, sm.sugar_total_g, sm.sodium_total_mg, IFNULL(sm.micronutrients_json,''),
       sm.usage_count, sm.last_used_at, sm.archived_at, sm.created_at, sm.updated_at
FROM saved_meals sm
JOIN categories c ON c.id = sm.default_category_id`
}

func scanSavedMeal(row *sql.Row) (*model.SavedMeal, error) {
	var item model.SavedMeal
	var lastUsed sql.NullString
	var archived sql.NullString
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.NameNorm,
		&item.DefaultCategoryID,
		&item.DefaultCategory,
		&item.Notes,
		&item.CaloriesTotal,
		&item.ProteinTotalG,
		&item.CarbsTotalG,
		&item.FatTotalG,
		&item.FiberTotalG,
		&item.SugarTotalG,
		&item.SodiumTotalMg,
		&item.Micronutrients,
		&item.UsageCount,
		&lastUsed,
		&archived,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		t, err := time.Parse(time.RFC3339, lastUsed.String)
		if err == nil {
			item.LastUsedAt = &t
		}
	}
	if archived.Valid {
		t, err := time.Parse(time.RFC3339, archived.String)
		if err == nil {
			item.ArchivedAt = &t
		}
	}
	return &item, nil
}

func scanSavedMealRows(rows *sql.Rows) (*model.SavedMeal, error) {
	var item model.SavedMeal
	var lastUsed sql.NullString
	var archived sql.NullString
	if err := rows.Scan(
		&item.ID,
		&item.Name,
		&item.NameNorm,
		&item.DefaultCategoryID,
		&item.DefaultCategory,
		&item.Notes,
		&item.CaloriesTotal,
		&item.ProteinTotalG,
		&item.CarbsTotalG,
		&item.FatTotalG,
		&item.FiberTotalG,
		&item.SugarTotalG,
		&item.SodiumTotalMg,
		&item.Micronutrients,
		&item.UsageCount,
		&lastUsed,
		&archived,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan saved meal: %w", err)
	}
	if lastUsed.Valid {
		t, err := time.Parse(time.RFC3339, lastUsed.String)
		if err == nil {
			item.LastUsedAt = &t
		}
	}
	if archived.Valid {
		t, err := time.Parse(time.RFC3339, archived.String)
		if err == nil {
			item.ArchivedAt = &t
		}
	}
	return &item, nil
}
