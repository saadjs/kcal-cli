package service

import (
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/model"
)

type ExportEntry struct {
	Name       string  `json:"name"`
	Calories   int     `json:"calories"`
	ProteinG   float64 `json:"protein_g"`
	CarbsG     float64 `json:"carbs_g"`
	FatG       float64 `json:"fat_g"`
	Category   string  `json:"category"`
	ConsumedAt string  `json:"consumed_at"`
	Notes      string  `json:"notes"`
	SourceType string  `json:"source_type"`
	SourceID   int64   `json:"source_id,omitempty"`
	Metadata   string  `json:"metadata_json,omitempty"`
}

type ExportRecipeIngredient struct {
	RecipeName string  `json:"recipe_name"`
	Name       string  `json:"name"`
	Amount     float64 `json:"amount"`
	AmountUnit string  `json:"amount_unit"`
	Calories   int     `json:"calories"`
	ProteinG   float64 `json:"protein_g"`
	CarbsG     float64 `json:"carbs_g"`
	FatG       float64 `json:"fat_g"`
}

type ExportData struct {
	Categories        []string                 `json:"categories"`
	Entries           []ExportEntry            `json:"entries"`
	Goals             []model.Goal             `json:"goals"`
	BodyMeasurements  []model.BodyMeasurement  `json:"body_measurements"`
	BodyGoals         []model.BodyGoal         `json:"body_goals"`
	Recipes           []model.Recipe           `json:"recipes"`
	RecipeIngredients []ExportRecipeIngredient `json:"recipe_ingredients"`
}

type ImportMode string

const (
	ImportModeFail    ImportMode = "fail"
	ImportModeSkip    ImportMode = "skip"
	ImportModeMerge   ImportMode = "merge"
	ImportModeReplace ImportMode = "replace"
)

type ImportOptions struct {
	Mode   ImportMode
	DryRun bool
}

type ImportReport struct {
	Inserted  int      `json:"inserted"`
	Updated   int      `json:"updated"`
	Skipped   int      `json:"skipped"`
	Conflicts int      `json:"conflicts"`
	Warnings  []string `json:"warnings,omitempty"`
}

func ExportDataSnapshot(db *sql.DB) (*ExportData, error) {
	out := &ExportData{}

	catRows, err := db.Query(`SELECT name FROM categories WHERE archived_at IS NULL ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("export categories: %w", err)
	}
	for catRows.Next() {
		var name string
		if err := catRows.Scan(&name); err != nil {
			_ = catRows.Close()
			return nil, fmt.Errorf("scan export category: %w", err)
		}
		out.Categories = append(out.Categories, name)
	}
	_ = catRows.Close()

	entryRows, err := db.Query(`
SELECT e.name, e.calories, e.protein_g, e.carbs_g, e.fat_g, c.name, e.consumed_at, IFNULL(e.notes,''), e.source_type, IFNULL(e.source_id,0), IFNULL(e.metadata_json,'')
FROM entries e
JOIN categories c ON c.id = e.category_id
ORDER BY e.consumed_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("export entries: %w", err)
	}
	for entryRows.Next() {
		var item ExportEntry
		if err := entryRows.Scan(&item.Name, &item.Calories, &item.ProteinG, &item.CarbsG, &item.FatG, &item.Category, &item.ConsumedAt, &item.Notes, &item.SourceType, &item.SourceID, &item.Metadata); err != nil {
			_ = entryRows.Close()
			return nil, fmt.Errorf("scan export entry: %w", err)
		}
		out.Entries = append(out.Entries, item)
	}
	_ = entryRows.Close()

	goalRows, err := db.Query(`SELECT id, calories, protein_g, carbs_g, fat_g, effective_date, created_at FROM goals ORDER BY effective_date ASC`)
	if err != nil {
		return nil, fmt.Errorf("export goals: %w", err)
	}
	for goalRows.Next() {
		var g model.Goal
		var created string
		if err := goalRows.Scan(&g.ID, &g.Calories, &g.ProteinG, &g.CarbsG, &g.FatG, &g.EffectiveDate, &created); err != nil {
			_ = goalRows.Close()
			return nil, fmt.Errorf("scan export goal: %w", err)
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, created)
		out.Goals = append(out.Goals, g)
	}
	_ = goalRows.Close()

	bodyRows, err := db.Query(`SELECT id, measured_at, weight_kg, body_fat_pct, IFNULL(notes,'') FROM body_measurements ORDER BY measured_at ASC`)
	if err != nil {
		return nil, fmt.Errorf("export body measurements: %w", err)
	}
	for bodyRows.Next() {
		var b model.BodyMeasurement
		var measured string
		var bf sql.NullFloat64
		if err := bodyRows.Scan(&b.ID, &measured, &b.WeightKg, &bf, &b.Notes); err != nil {
			_ = bodyRows.Close()
			return nil, fmt.Errorf("scan export body measurement: %w", err)
		}
		b.MeasuredAt, _ = time.Parse(time.RFC3339, measured)
		if bf.Valid {
			v := bf.Float64
			b.BodyFatPct = &v
		}
		out.BodyMeasurements = append(out.BodyMeasurements, b)
	}
	_ = bodyRows.Close()

	bodyGoalRows, err := db.Query(`SELECT id, target_weight_kg, target_body_fat_pct, IFNULL(target_date,''), effective_date, created_at FROM body_goals ORDER BY effective_date ASC`)
	if err != nil {
		return nil, fmt.Errorf("export body goals: %w", err)
	}
	for bodyGoalRows.Next() {
		var g model.BodyGoal
		var targetBF sql.NullFloat64
		var created string
		if err := bodyGoalRows.Scan(&g.ID, &g.TargetWeightKg, &targetBF, &g.TargetDate, &g.EffectiveDate, &created); err != nil {
			_ = bodyGoalRows.Close()
			return nil, fmt.Errorf("scan export body goal: %w", err)
		}
		g.CreatedAt, _ = time.Parse(time.RFC3339, created)
		if targetBF.Valid {
			v := targetBF.Float64
			g.TargetBodyFatPct = &v
		}
		out.BodyGoals = append(out.BodyGoals, g)
	}
	_ = bodyGoalRows.Close()

	recipeRows, err := db.Query(`SELECT id, name, calories_total, protein_total_g, carbs_total_g, fat_total_g, servings, IFNULL(notes,''), created_at, updated_at FROM recipes ORDER BY name ASC`)
	if err != nil {
		return nil, fmt.Errorf("export recipes: %w", err)
	}
	for recipeRows.Next() {
		var r model.Recipe
		var created, updated string
		if err := recipeRows.Scan(&r.ID, &r.Name, &r.CaloriesTotal, &r.ProteinTotalG, &r.CarbsTotalG, &r.FatTotalG, &r.Servings, &r.Notes, &created, &updated); err != nil {
			_ = recipeRows.Close()
			return nil, fmt.Errorf("scan export recipe: %w", err)
		}
		r.CreatedAt, _ = time.Parse(time.RFC3339, created)
		r.UpdatedAt, _ = time.Parse(time.RFC3339, updated)
		out.Recipes = append(out.Recipes, r)
	}
	_ = recipeRows.Close()

	ingRows, err := db.Query(`
SELECT r.name, i.name, i.amount, i.amount_unit, i.calories, i.protein_g, i.carbs_g, i.fat_g
FROM recipe_ingredients i
JOIN recipes r ON r.id = i.recipe_id
ORDER BY r.name, i.id ASC`)
	if err != nil {
		return nil, fmt.Errorf("export recipe ingredients: %w", err)
	}
	for ingRows.Next() {
		var i ExportRecipeIngredient
		if err := ingRows.Scan(&i.RecipeName, &i.Name, &i.Amount, &i.AmountUnit, &i.Calories, &i.ProteinG, &i.CarbsG, &i.FatG); err != nil {
			_ = ingRows.Close()
			return nil, fmt.Errorf("scan export recipe ingredient: %w", err)
		}
		out.RecipeIngredients = append(out.RecipeIngredients, i)
	}
	_ = ingRows.Close()

	return out, nil
}

func ImportDataSnapshot(db *sql.DB, data *ExportData) (ImportReport, error) {
	return ImportDataSnapshotWithOptions(db, data, ImportOptions{Mode: ImportModeMerge})
}

func ImportDataSnapshotWithOptions(db *sql.DB, data *ExportData, opts ImportOptions) (ImportReport, error) {
	report := ImportReport{}
	mode := normalizeImportMode(opts.Mode)

	tx, err := db.Begin()
	if err != nil {
		return report, fmt.Errorf("begin import tx: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if mode == ImportModeReplace {
		if err := clearUserData(tx); err != nil {
			return report, err
		}
	}

	for _, c := range data.Categories {
		if strings.TrimSpace(c) == "" {
			continue
		}
		if opts.DryRun {
			report.Inserted++
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO categories(name, is_default) VALUES(?, 0)`, normalizeName(c)); err != nil {
			return report, fmt.Errorf("import category %q: %w", c, err)
		}
	}
	for _, g := range data.Goals {
		if opts.DryRun {
			report.Inserted++
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO goals(calories, protein_g, carbs_g, fat_g, effective_date) VALUES(?, ?, ?, ?, ?)`, g.Calories, g.ProteinG, g.CarbsG, g.FatG, g.EffectiveDate); err != nil {
			return report, fmt.Errorf("import goal %q: %w", g.EffectiveDate, err)
		}
	}

	for _, b := range data.BodyMeasurements {
		if opts.DryRun {
			report.Inserted++
			continue
		}
		if _, err := tx.Exec(`INSERT INTO body_measurements(measured_at, weight_kg, body_fat_pct, notes) VALUES(?, ?, ?, ?)`, b.MeasuredAt.Format(time.RFC3339), b.WeightKg, b.BodyFatPct, b.Notes); err != nil {
			return report, fmt.Errorf("import body measurement %s: %w", b.MeasuredAt.Format(time.RFC3339), err)
		}
	}
	for _, g := range data.BodyGoals {
		if opts.DryRun {
			report.Inserted++
			continue
		}
		if _, err := tx.Exec(`INSERT OR IGNORE INTO body_goals(target_weight_kg, target_body_fat_pct, target_date, effective_date) VALUES(?, ?, ?, ?)`, g.TargetWeightKg, g.TargetBodyFatPct, g.TargetDate, g.EffectiveDate); err != nil {
			return report, fmt.Errorf("import body goal %q: %w", g.EffectiveDate, err)
		}
	}
	for _, r := range data.Recipes {
		if opts.DryRun {
			report.Inserted++
			continue
		}
		if _, err := tx.Exec(`
INSERT INTO recipes(name, calories_total, protein_total_g, carbs_total_g, fat_total_g, servings, notes)
VALUES(?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET calories_total=excluded.calories_total, protein_total_g=excluded.protein_total_g, carbs_total_g=excluded.carbs_total_g, fat_total_g=excluded.fat_total_g, servings=excluded.servings, notes=excluded.notes, updated_at=CURRENT_TIMESTAMP
`, r.Name, r.CaloriesTotal, r.ProteinTotalG, r.CarbsTotalG, r.FatTotalG, r.Servings, r.Notes); err != nil {
			return report, fmt.Errorf("import recipe %q: %w", r.Name, err)
		}
	}
	for _, i := range data.RecipeIngredients {
		if opts.DryRun {
			report.Inserted++
			continue
		}
		var recipeID int64
		if err := tx.QueryRow(`SELECT id FROM recipes WHERE name = ?`, i.RecipeName).Scan(&recipeID); err != nil {
			return report, fmt.Errorf("find recipe %q for ingredient: %w", i.RecipeName, err)
		}
		if _, err := tx.Exec(`INSERT INTO recipe_ingredients(recipe_id, name, amount, amount_unit, calories, protein_g, carbs_g, fat_g) VALUES(?, ?, ?, ?, ?, ?, ?, ?)`, recipeID, i.Name, i.Amount, i.AmountUnit, i.Calories, i.ProteinG, i.CarbsG, i.FatG); err != nil {
			return report, fmt.Errorf("import ingredient %q: %w", i.Name, err)
		}
	}

	for idx, e := range data.Entries {
		if strings.TrimSpace(e.Category) == "" {
			report.Warnings = append(report.Warnings, fmt.Sprintf("entry[%d] missing category", idx))
			report.Conflicts++
			continue
		}
		if opts.DryRun {
			existingID, err := findExistingEntryID(tx, e)
			if err != nil {
				return report, err
			}
			if existingID > 0 {
				switch mode {
				case ImportModeFail:
					report.Conflicts++
					return report, fmt.Errorf("import conflict for entry %q at %s", e.Name, e.ConsumedAt)
				case ImportModeSkip:
					report.Skipped++
				case ImportModeMerge, ImportModeReplace:
					report.Updated++
				}
			} else {
				report.Inserted++
			}
			continue
		}

		if _, err := tx.Exec(`INSERT OR IGNORE INTO categories(name, is_default) VALUES(?, 0)`, normalizeName(e.Category)); err != nil {
			return report, fmt.Errorf("import entry category %q: %w", e.Category, err)
		}
		var categoryID int64
		if err := tx.QueryRow(`SELECT id FROM categories WHERE name = ?`, normalizeName(e.Category)).Scan(&categoryID); err != nil {
			return report, fmt.Errorf("resolve entry category %q: %w", e.Category, err)
		}
		existingID, err := findExistingEntryID(tx, e)
		if err != nil {
			return report, err
		}
		if existingID > 0 {
			switch mode {
			case ImportModeFail:
				report.Conflicts++
				return report, fmt.Errorf("import conflict for entry %q at %s", e.Name, e.ConsumedAt)
			case ImportModeSkip:
				report.Skipped++
				continue
			case ImportModeMerge, ImportModeReplace:
				sourceID := sql.NullInt64{}
				if e.SourceID > 0 {
					sourceID.Valid = true
					sourceID.Int64 = e.SourceID
				}
				if _, err := tx.Exec(`UPDATE entries SET calories=?, protein_g=?, carbs_g=?, fat_g=?, notes=?, source_type=?, source_id=?, metadata_json=?, updated_at=CURRENT_TIMESTAMP WHERE id=?`, e.Calories, e.ProteinG, e.CarbsG, e.FatG, e.Notes, e.SourceType, sourceID, e.Metadata, existingID); err != nil {
					return report, fmt.Errorf("merge entry %q: %w", e.Name, err)
				}
				report.Updated++
				continue
			}
		}
		sourceID := sql.NullInt64{}
		if e.SourceID > 0 {
			sourceID.Valid = true
			sourceID.Int64 = e.SourceID
		}
		if _, err := tx.Exec(`
INSERT INTO entries(name, calories, protein_g, carbs_g, fat_g, category_id, consumed_at, notes, source_type, source_id, metadata_json)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, e.Name, e.Calories, e.ProteinG, e.CarbsG, e.FatG, categoryID, e.ConsumedAt, e.Notes, e.SourceType, sourceID, e.Metadata); err != nil {
			return report, fmt.Errorf("import entry %q: %w", e.Name, err)
		}
		report.Inserted++
	}

	if opts.DryRun {
		return report, nil
	}
	if err := tx.Commit(); err != nil {
		return report, fmt.Errorf("commit import tx: %w", err)
	}
	return report, nil
}

func normalizeImportMode(mode ImportMode) ImportMode {
	switch mode {
	case ImportModeFail, ImportModeSkip, ImportModeMerge, ImportModeReplace:
		return mode
	default:
		return ImportModeMerge
	}
}

func findExistingEntryID(tx *sql.Tx, e ExportEntry) (int64, error) {
	var id int64
	err := tx.QueryRow(`
SELECT e.id
FROM entries e
JOIN categories c ON c.id = e.category_id
WHERE e.name = ? AND c.name = ? AND e.consumed_at = ? AND e.source_type = ?
LIMIT 1
`, e.Name, normalizeName(e.Category), e.ConsumedAt, e.SourceType).Scan(&id)
	if err == sql.ErrNoRows {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("check existing entry %q: %w", e.Name, err)
	}
	return id, nil
}

func clearUserData(tx *sql.Tx) error {
	stmts := []string{
		`DELETE FROM recipe_ingredients`,
		`DELETE FROM entries`,
		`DELETE FROM recipes`,
		`DELETE FROM goals`,
		`DELETE FROM body_measurements`,
		`DELETE FROM body_goals`,
		`DELETE FROM categories WHERE is_default = 0`,
	}
	for _, s := range stmts {
		if _, err := tx.Exec(s); err != nil {
			return fmt.Errorf("clear data for replace mode: %w", err)
		}
	}
	return nil
}
