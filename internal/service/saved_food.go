package service

import (
	"database/sql"
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/model"
)

type CreateSavedFoodInput struct {
	Name        string
	Brand       string
	Category    string
	Calories    int
	ProteinG    float64
	CarbsG      float64
	FatG        float64
	FiberG      float64
	SugarG      float64
	SodiumMg    float64
	Micros      string
	ServingAmt  float64
	ServingUnit string
	SourceType  string
	SourceProv  string
	SourceRef   string
	Notes       string
	Metadata    string
}

type UpdateSavedFoodInput struct {
	Name        string
	Brand       string
	Category    string
	Calories    int
	ProteinG    float64
	CarbsG      float64
	FatG        float64
	FiberG      float64
	SugarG      float64
	SodiumMg    float64
	Micros      string
	ServingAmt  float64
	ServingUnit string
	SourceType  string
	SourceProv  string
	SourceRef   string
	Notes       string
	Metadata    string
}

type ListSavedFoodsFilter struct {
	IncludeArchived bool
	Limit           int
	Query           string
}

type LogSavedFoodInput struct {
	Identifier string
	Servings   float64
	Category   string
	ConsumedAt time.Time
	Notes      string
}

func CreateSavedFood(db *sql.DB, in CreateSavedFoodInput) (int64, error) {
	name := strings.TrimSpace(in.Name)
	if name == "" {
		return 0, fmt.Errorf("saved food name is required")
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
	if in.ServingAmt <= 0 {
		in.ServingAmt = 1
	}
	if strings.TrimSpace(in.ServingUnit) == "" {
		in.ServingUnit = "serving"
	}
	if strings.TrimSpace(in.SourceType) == "" {
		in.SourceType = "manual"
	}
	catID, err := resolveCategoryIDWithDefault(db, in.Category)
	if err != nil {
		return 0, err
	}
	micros, err := normalizeMicronutrientsJSON(in.Micros)
	if err != nil {
		return 0, err
	}
	metadata, err := normalizeEntryMetadata(in.Metadata)
	if err != nil {
		return 0, err
	}

	res, err := db.Exec(`
INSERT INTO saved_foods(
  name, name_norm, brand, default_category_id,
  calories, protein_g, carbs_g, fat_g, fiber_g, sugar_g, sodium_mg, micronutrients_json,
  serving_amount, serving_unit,
  source_type, source_provider, source_ref,
  notes, metadata_json
) VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`,
		name,
		normalizeName(name),
		strings.TrimSpace(in.Brand),
		catID,
		in.Calories,
		in.ProteinG,
		in.CarbsG,
		in.FatG,
		in.FiberG,
		in.SugarG,
		in.SodiumMg,
		micros,
		in.ServingAmt,
		strings.TrimSpace(in.ServingUnit),
		strings.TrimSpace(in.SourceType),
		strings.TrimSpace(in.SourceProv),
		strings.TrimSpace(in.SourceRef),
		strings.TrimSpace(in.Notes),
		metadata,
	)
	if err != nil {
		return 0, fmt.Errorf("create saved food: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve saved food id: %w", err)
	}
	return id, nil
}

func CreateSavedFoodFromEntry(db *sql.DB, entryID int64, name, notes string) (int64, error) {
	e, err := EntryByID(db, entryID)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(name) == "" {
		name = e.Name
	}
	return CreateSavedFood(db, CreateSavedFoodInput{
		Name:       name,
		Brand:      "",
		Category:   e.Category,
		Calories:   e.Calories,
		ProteinG:   e.ProteinG,
		CarbsG:     e.CarbsG,
		FatG:       e.FatG,
		FiberG:     e.FiberG,
		SugarG:     e.SugarG,
		SodiumMg:   e.SodiumMg,
		Micros:     e.Micronutrients,
		ServingAmt: 1,
		SourceType: "entry",
		SourceRef:  strconv.FormatInt(entryID, 10),
		Notes:      strings.TrimSpace(notes),
		Metadata:   e.Metadata,
	})
}

func ResolveSavedFood(db *sql.DB, idOrName string) (*model.SavedFood, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return nil, fmt.Errorf("saved food identifier is required")
	}
	var row *sql.Row
	if id, err := parseIDLoose(idOrName); err == nil {
		row = db.QueryRow(savedFoodSelectBase()+` WHERE sf.id = ?`, id)
	} else {
		row = db.QueryRow(savedFoodSelectBase()+` WHERE sf.name_norm = ?`, normalizeName(idOrName))
	}
	item, err := scanSavedFood(row)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("saved food %q not found", idOrName)
	}
	if err != nil {
		return nil, fmt.Errorf("resolve saved food %q: %w", idOrName, err)
	}
	return item, nil
}

func ListSavedFoods(db *sql.DB, f ListSavedFoodsFilter) ([]model.SavedFood, error) {
	query := savedFoodSelectBase() + ` WHERE 1=1`
	args := make([]any, 0, 4)
	if !f.IncludeArchived {
		query += ` AND sf.archived_at IS NULL`
	}
	if strings.TrimSpace(f.Query) != "" {
		query += ` AND sf.name_norm LIKE ?`
		args = append(args, "%"+normalizeName(f.Query)+"%")
	}
	query += ` ORDER BY sf.usage_count DESC, sf.last_used_at DESC, sf.updated_at DESC, sf.name ASC`
	if f.Limit <= 0 {
		f.Limit = 100
	}
	query += ` LIMIT ?`
	args = append(args, f.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list saved foods: %w", err)
	}
	defer rows.Close()
	out := make([]model.SavedFood, 0)
	for rows.Next() {
		item, err := scanSavedFoodRows(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate saved foods: %w", err)
	}
	return out, nil
}

func UpdateSavedFood(db *sql.DB, idOrName string, in UpdateSavedFoodInput) error {
	item, err := ResolveSavedFood(db, idOrName)
	if err != nil {
		return err
	}
	if strings.TrimSpace(in.Name) == "" {
		return fmt.Errorf("saved food name is required")
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
	if in.ServingAmt <= 0 {
		in.ServingAmt = 1
	}
	if strings.TrimSpace(in.ServingUnit) == "" {
		in.ServingUnit = "serving"
	}
	if strings.TrimSpace(in.SourceType) == "" {
		in.SourceType = "manual"
	}
	catID, err := resolveCategoryIDWithDefault(db, in.Category)
	if err != nil {
		return err
	}
	micros, err := normalizeMicronutrientsJSON(in.Micros)
	if err != nil {
		return err
	}
	metadata, err := normalizeEntryMetadata(in.Metadata)
	if err != nil {
		return err
	}
	_, err = db.Exec(`
UPDATE saved_foods
SET name = ?, name_norm = ?, brand = ?, default_category_id = ?,
    calories = ?, protein_g = ?, carbs_g = ?, fat_g = ?, fiber_g = ?, sugar_g = ?, sodium_mg = ?, micronutrients_json = ?,
    serving_amount = ?, serving_unit = ?, source_type = ?, source_provider = ?, source_ref = ?, notes = ?, metadata_json = ?,
    updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`,
		strings.TrimSpace(in.Name),
		normalizeName(in.Name),
		strings.TrimSpace(in.Brand),
		catID,
		in.Calories,
		in.ProteinG,
		in.CarbsG,
		in.FatG,
		in.FiberG,
		in.SugarG,
		in.SodiumMg,
		micros,
		in.ServingAmt,
		strings.TrimSpace(in.ServingUnit),
		strings.TrimSpace(in.SourceType),
		strings.TrimSpace(in.SourceProv),
		strings.TrimSpace(in.SourceRef),
		strings.TrimSpace(in.Notes),
		metadata,
		item.ID,
	)
	if err != nil {
		return fmt.Errorf("update saved food %q: %w", idOrName, err)
	}
	return nil
}

func ArchiveSavedFood(db *sql.DB, idOrName string) error {
	item, err := ResolveSavedFood(db, idOrName)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE saved_foods SET archived_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, item.ID)
	if err != nil {
		return fmt.Errorf("archive saved food %q: %w", idOrName, err)
	}
	return nil
}

func RestoreSavedFood(db *sql.DB, idOrName string) error {
	item, err := ResolveSavedFood(db, idOrName)
	if err != nil {
		return err
	}
	_, err = db.Exec(`UPDATE saved_foods SET archived_at = NULL, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, item.ID)
	if err != nil {
		return fmt.Errorf("restore saved food %q: %w", idOrName, err)
	}
	return nil
}

func LogSavedFood(db *sql.DB, in LogSavedFoodInput) (int64, error) {
	if in.Servings <= 0 {
		in.Servings = 1
	}
	item, err := ResolveSavedFood(db, in.Identifier)
	if err != nil {
		return 0, err
	}
	if item.ArchivedAt != nil {
		return 0, fmt.Errorf("saved food %q is archived", item.Name)
	}
	if in.ConsumedAt.IsZero() {
		in.ConsumedAt = time.Now()
	}
	category := strings.TrimSpace(in.Category)
	if category == "" {
		category = item.DefaultCategory
	}
	micros, err := ParseMicronutrientsJSON(item.Micronutrients)
	if err != nil {
		return 0, err
	}
	microsJSON, err := EncodeMicronutrientsJSON(ScaleMicronutrients(micros, in.Servings))
	if err != nil {
		return 0, err
	}
	sourceID := item.ID
	entryID, err := CreateEntry(db, CreateEntryInput{
		Name:           fmt.Sprintf("%s (saved food x%.2f)", item.Name, in.Servings),
		Calories:       int(math.Round(float64(item.Calories) * in.Servings)),
		ProteinG:       item.ProteinG * in.Servings,
		CarbsG:         item.CarbsG * in.Servings,
		FatG:           item.FatG * in.Servings,
		FiberG:         item.FiberG * in.Servings,
		SugarG:         item.SugarG * in.Servings,
		SodiumMg:       item.SodiumMg * in.Servings,
		Micronutrients: microsJSON,
		Category:       category,
		Consumed:       in.ConsumedAt,
		Notes:          strings.TrimSpace(in.Notes),
		SourceType:     "saved_food",
		SourceID:       &sourceID,
	})
	if err != nil {
		return 0, err
	}
	if _, err := db.Exec(`UPDATE saved_foods SET usage_count = usage_count + 1, last_used_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, item.ID); err != nil {
		return 0, fmt.Errorf("update saved food usage: %w", err)
	}
	return entryID, nil
}

func resolveCategoryIDWithDefault(db *sql.DB, category string) (int64, error) {
	name := strings.TrimSpace(category)
	if name == "" {
		name = "snacks"
	}
	return categoryIDByName(db, name)
}

func savedFoodSelectBase() string {
	return `
SELECT sf.id, sf.name, sf.name_norm, sf.brand, sf.default_category_id, c.name,
       sf.calories, sf.protein_g, sf.carbs_g, sf.fat_g, sf.fiber_g, sf.sugar_g, sf.sodium_mg, IFNULL(sf.micronutrients_json,''),
       sf.serving_amount, sf.serving_unit, sf.source_type, sf.source_provider, sf.source_ref,
       IFNULL(sf.notes,''), IFNULL(sf.metadata_json,''), sf.usage_count, sf.last_used_at, sf.archived_at, sf.created_at, sf.updated_at
FROM saved_foods sf
JOIN categories c ON c.id = sf.default_category_id`
}

func scanSavedFood(row *sql.Row) (*model.SavedFood, error) {
	var item model.SavedFood
	var lastUsed sql.NullString
	var archived sql.NullString
	if err := row.Scan(
		&item.ID,
		&item.Name,
		&item.NameNorm,
		&item.Brand,
		&item.DefaultCategoryID,
		&item.DefaultCategory,
		&item.Calories,
		&item.ProteinG,
		&item.CarbsG,
		&item.FatG,
		&item.FiberG,
		&item.SugarG,
		&item.SodiumMg,
		&item.Micronutrients,
		&item.ServingAmount,
		&item.ServingUnit,
		&item.SourceType,
		&item.SourceProvider,
		&item.SourceRef,
		&item.Notes,
		&item.Metadata,
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

func scanSavedFoodRows(rows *sql.Rows) (*model.SavedFood, error) {
	var item model.SavedFood
	var lastUsed sql.NullString
	var archived sql.NullString
	if err := rows.Scan(
		&item.ID,
		&item.Name,
		&item.NameNorm,
		&item.Brand,
		&item.DefaultCategoryID,
		&item.DefaultCategory,
		&item.Calories,
		&item.ProteinG,
		&item.CarbsG,
		&item.FatG,
		&item.FiberG,
		&item.SugarG,
		&item.SodiumMg,
		&item.Micronutrients,
		&item.ServingAmount,
		&item.ServingUnit,
		&item.SourceType,
		&item.SourceProvider,
		&item.SourceRef,
		&item.Notes,
		&item.Metadata,
		&item.UsageCount,
		&lastUsed,
		&archived,
		&item.CreatedAt,
		&item.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan saved food: %w", err)
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
