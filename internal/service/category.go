package service

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/saadjs/kcal-cli/internal/model"
)

func AddCategory(db *sql.DB, name string) error {
	name = normalizeName(name)
	if name == "" {
		return fmt.Errorf("category name is required")
	}
	if _, err := db.Exec(`INSERT INTO categories(name, is_default) VALUES(?, 0)`, name); err != nil {
		return fmt.Errorf("add category %q: %w", name, err)
	}
	return nil
}

func ListCategories(db *sql.DB) ([]model.Category, error) {
	rows, err := db.Query(`SELECT id, name, is_default, created_at FROM categories WHERE archived_at IS NULL ORDER BY name`)
	if err != nil {
		return nil, fmt.Errorf("list categories: %w", err)
	}
	defer rows.Close()

	categories := make([]model.Category, 0)
	for rows.Next() {
		var c model.Category
		var isDefault int
		if err := rows.Scan(&c.ID, &c.Name, &isDefault, &c.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan category: %w", err)
		}
		c.IsDefault = isDefault == 1
		categories = append(categories, c)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate categories: %w", err)
	}
	return categories, nil
}

func RenameCategory(db *sql.DB, oldName, newName string) error {
	oldName = normalizeName(oldName)
	newName = normalizeName(newName)
	if oldName == "" || newName == "" {
		return fmt.Errorf("old and new category names are required")
	}
	res, err := db.Exec(`UPDATE categories SET name = ? WHERE name = ?`, newName, oldName)
	if err != nil {
		return fmt.Errorf("rename category %q to %q: %w", oldName, newName, err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected for rename: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("category %q not found", oldName)
	}
	return nil
}

func DeleteCategory(db *sql.DB, name, reassign string) error {
	name = normalizeName(name)
	reassign = normalizeName(reassign)
	if name == "" {
		return fmt.Errorf("category name is required")
	}
	if name == reassign && name != "" {
		return fmt.Errorf("reassign category must be different from deleted category")
	}

	id, err := categoryIDByName(db, name)
	if err != nil {
		return err
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM entries WHERE category_id = ?`, id).Scan(&count); err != nil {
		return fmt.Errorf("count entries for category %q: %w", name, err)
	}

	if count > 0 {
		if strings.TrimSpace(reassign) == "" {
			return fmt.Errorf("category %q has %d entries; use --reassign to move them", name, count)
		}
		targetID, err := categoryIDByName(db, reassign)
		if err != nil {
			return fmt.Errorf("reassign target: %w", err)
		}
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin delete category tx: %w", err)
		}
		if _, err := tx.Exec(`UPDATE entries SET category_id = ? WHERE category_id = ?`, targetID, id); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("reassign entries: %w", err)
		}
		if _, err := tx.Exec(`DELETE FROM categories WHERE id = ?`, id); err != nil {
			_ = tx.Rollback()
			return fmt.Errorf("delete category: %w", err)
		}
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit delete category tx: %w", err)
		}
		return nil
	}

	if _, err := db.Exec(`DELETE FROM categories WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete category %q: %w", name, err)
	}
	return nil
}
