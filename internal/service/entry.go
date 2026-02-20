package service

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/saad/kcal-cli/internal/model"
)

type CreateEntryInput struct {
	Name       string
	Calories   int
	ProteinG   float64
	CarbsG     float64
	FatG       float64
	Category   string
	Consumed   time.Time
	Notes      string
	SourceType string
	SourceID   *int64
	Metadata   string
}

type ListEntriesFilter struct {
	Date     string
	FromDate string
	ToDate   string
	Category string
	Limit    int
}

type UpdateEntryInput struct {
	ID       int64
	Name     string
	Calories int
	ProteinG float64
	CarbsG   float64
	FatG     float64
	Category string
	Consumed time.Time
	Notes    string
}

func CreateEntry(db *sql.DB, in CreateEntryInput) (int64, error) {
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return 0, fmt.Errorf("entry name is required")
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
	if in.Consumed.IsZero() {
		in.Consumed = time.Now()
	}
	categoryID, err := categoryIDByName(db, in.Category)
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(in.SourceType) == "" {
		in.SourceType = "manual"
	}
	metadata, err := normalizeEntryMetadata(in.Metadata)
	if err != nil {
		return 0, err
	}

	res, err := db.Exec(`
INSERT INTO entries(name, calories, protein_g, carbs_g, fat_g, category_id, consumed_at, notes, source_type, source_id, metadata_json)
VALUES(?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
`, in.Name, in.Calories, in.ProteinG, in.CarbsG, in.FatG, categoryID, in.Consumed.Format(time.RFC3339), strings.TrimSpace(in.Notes), in.SourceType, in.SourceID, metadata)
	if err != nil {
		return 0, fmt.Errorf("insert entry: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("resolve inserted entry id: %w", err)
	}
	return id, nil
}

func ListEntries(db *sql.DB, f ListEntriesFilter) ([]model.Entry, error) {
	if err := validateListEntriesFilter(f); err != nil {
		return nil, err
	}

	query := `
SELECT e.id, e.name, e.calories, e.protein_g, e.carbs_g, e.fat_g, e.category_id, c.name, e.consumed_at, IFNULL(e.notes, ''), e.source_type, e.source_id, IFNULL(e.metadata_json, '')
FROM entries e
JOIN categories c ON c.id = e.category_id
WHERE 1=1`
	args := make([]any, 0)

	if strings.TrimSpace(f.Date) != "" {
		start, end, err := dayBounds(f.Date)
		if err != nil {
			return nil, err
		}
		query += ` AND e.consumed_at >= ? AND e.consumed_at < ?`
		args = append(args, start, end)
	}
	if strings.TrimSpace(f.FromDate) != "" {
		from, err := parseDateStart(f.FromDate)
		if err != nil {
			return nil, err
		}
		query += ` AND e.consumed_at >= ?`
		args = append(args, from)
	}
	if strings.TrimSpace(f.ToDate) != "" {
		to, err := parseDateEndExclusive(f.ToDate)
		if err != nil {
			return nil, err
		}
		query += ` AND e.consumed_at < ?`
		args = append(args, to)
	}
	if strings.TrimSpace(f.Category) != "" {
		query += ` AND c.name = ?`
		args = append(args, normalizeName(f.Category))
	}
	query += ` ORDER BY e.consumed_at DESC`

	if f.Limit <= 0 {
		f.Limit = 50
	}
	query += ` LIMIT ?`
	args = append(args, f.Limit)

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("list entries: %w", err)
	}
	defer rows.Close()

	entries := make([]model.Entry, 0)
	for rows.Next() {
		var e model.Entry
		var consumedAtRaw string
		var sourceID sql.NullInt64
		if err := rows.Scan(&e.ID, &e.Name, &e.Calories, &e.ProteinG, &e.CarbsG, &e.FatG, &e.CategoryID, &e.Category, &consumedAtRaw, &e.Notes, &e.SourceType, &sourceID, &e.Metadata); err != nil {
			return nil, fmt.Errorf("scan entry: %w", err)
		}
		consumedAt, err := time.Parse(time.RFC3339, consumedAtRaw)
		if err != nil {
			return nil, fmt.Errorf("parse consumed_at for entry %d: %w", e.ID, err)
		}
		e.ConsumedAt = consumedAt
		if sourceID.Valid {
			v := sourceID.Int64
			e.SourceID = &v
		}
		entries = append(entries, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate entries: %w", err)
	}
	return entries, nil
}

func UpdateEntry(db *sql.DB, in UpdateEntryInput) error {
	if in.ID <= 0 {
		return fmt.Errorf("entry id must be > 0")
	}
	in.Name = strings.TrimSpace(in.Name)
	if in.Name == "" {
		return fmt.Errorf("entry name is required")
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
	if in.Consumed.IsZero() {
		return fmt.Errorf("consumed time is required")
	}
	categoryID, err := categoryIDByName(db, in.Category)
	if err != nil {
		return err
	}

	res, err := db.Exec(`
UPDATE entries
SET name = ?, calories = ?, protein_g = ?, carbs_g = ?, fat_g = ?, category_id = ?, consumed_at = ?, notes = ?, updated_at = CURRENT_TIMESTAMP
WHERE id = ?
`, in.Name, in.Calories, in.ProteinG, in.CarbsG, in.FatG, categoryID, in.Consumed.Format(time.RFC3339), strings.TrimSpace(in.Notes), in.ID)
	if err != nil {
		return fmt.Errorf("update entry %d: %w", in.ID, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected for entry %d: %w", in.ID, err)
	}
	if affected == 0 {
		return fmt.Errorf("entry %d not found", in.ID)
	}
	return nil
}

func DeleteEntry(db *sql.DB, id int64) error {
	if id <= 0 {
		return fmt.Errorf("entry id must be > 0")
	}
	res, err := db.Exec(`DELETE FROM entries WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete entry %d: %w", id, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected for entry %d: %w", id, err)
	}
	if affected == 0 {
		return fmt.Errorf("entry %d not found", id)
	}
	return nil
}

func dayBounds(date string) (string, string, error) {
	start, err := parseDateStart(date)
	if err != nil {
		return "", "", err
	}
	t, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return "", "", fmt.Errorf("parse RFC3339 %q: %w", start, err)
	}
	return start, t.Add(24 * time.Hour).Format(time.RFC3339), nil
}

func parseDateStart(value string) (string, error) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(value), time.Local)
	if err != nil {
		return "", fmt.Errorf("invalid date %q, expected YYYY-MM-DD", value)
	}
	return t.Format(time.RFC3339), nil
}

func parseDateEndExclusive(value string) (string, error) {
	start, err := parseDateStart(value)
	if err != nil {
		return "", err
	}
	t, err := time.Parse(time.RFC3339, start)
	if err != nil {
		return "", fmt.Errorf("parse end date %q: %w", value, err)
	}
	return t.Add(24 * time.Hour).Format(time.RFC3339), nil
}

func normalizeEntryMetadata(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}
	if !json.Valid([]byte(value)) {
		return "", fmt.Errorf("entry metadata must be valid JSON")
	}
	var decoded map[string]any
	if err := json.Unmarshal([]byte(value), &decoded); err != nil {
		return "", fmt.Errorf("entry metadata must be a JSON object: %w", err)
	}
	normalized, err := json.Marshal(decoded)
	if err != nil {
		return "", fmt.Errorf("marshal entry metadata: %w", err)
	}
	return string(normalized), nil
}

func validateListEntriesFilter(f ListEntriesFilter) error {
	if strings.TrimSpace(f.Date) != "" && (strings.TrimSpace(f.FromDate) != "" || strings.TrimSpace(f.ToDate) != "") {
		return fmt.Errorf("--date cannot be combined with --from or --to")
	}
	return nil
}
