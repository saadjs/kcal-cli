package service_test

import (
	"strings"
	"testing"
	"time"

	"github.com/saad/kcal-cli/internal/service"
)

func TestCategoryAndEntryLifecycle(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	if err := service.AddCategory(db, "supper"); err != nil {
		t.Fatalf("add category: %v", err)
	}

	consumed := time.Date(2026, 2, 20, 19, 30, 0, 0, time.Local)
	id, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:       "Chicken bowl",
		Calories:   550,
		ProteinG:   45,
		CarbsG:     40,
		FatG:       18,
		Category:   "supper",
		Consumed:   consumed,
		SourceType: "manual",
	})
	if err != nil {
		t.Fatalf("create entry: %v", err)
	}
	if id <= 0 {
		t.Fatalf("expected inserted entry id > 0, got %d", id)
	}

	entries, err := service.ListEntries(db, service.ListEntriesFilter{
		Category: "supper",
		Date:     "2026-02-20",
	})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry in supper, got %d", len(entries))
	}

	if err := service.DeleteCategory(db, "supper", ""); err == nil {
		t.Fatalf("expected delete category without reassign to fail")
	}
	if err := service.DeleteCategory(db, "supper", "dinner"); err != nil {
		t.Fatalf("delete category with reassignment: %v", err)
	}

	reassigned, err := service.ListEntries(db, service.ListEntriesFilter{Category: "dinner"})
	if err != nil {
		t.Fatalf("list reassigned entries: %v", err)
	}
	if len(reassigned) != 1 {
		t.Fatalf("expected reassigned entry count 1, got %d", len(reassigned))
	}
}

func TestListEntriesRejectsConflictingDateFilters(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.ListEntries(db, service.ListEntriesFilter{
		Date:     "2026-02-20",
		FromDate: "2026-02-01",
	})
	if err == nil {
		t.Fatalf("expected conflicting date filters to fail")
	}
}

func TestCreateEntryStoresMetadataJSON(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:           "Metadata meal",
		Calories:       100,
		ProteinG:       10,
		CarbsG:         10,
		FatG:           2,
		FiberG:         3,
		SugarG:         4,
		SodiumMg:       120,
		Micronutrients: `{"vitamin_c":{"value":30,"unit":"mg"}}`,
		Category:       "breakfast",
		Consumed:       time.Date(2026, 2, 20, 8, 0, 0, 0, time.Local),
		SourceType:     "manual",
		Metadata:       `{"tag":"check","source":"import"}`,
	})
	if err != nil {
		t.Fatalf("create entry with metadata: %v", err)
	}

	entries, err := service.ListEntries(db, service.ListEntriesFilter{
		Date:  "2026-02-20",
		Limit: 5,
	})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].FiberG != 3 || entries[0].SugarG != 4 || entries[0].SodiumMg != 120 {
		t.Fatalf("expected richer nutrients, got %+v", entries[0])
	}
	if !strings.Contains(entries[0].Micronutrients, `"vitamin_c"`) {
		t.Fatalf("expected micronutrients to be stored, got: %s", entries[0].Micronutrients)
	}
	if !strings.Contains(entries[0].Metadata, `"tag":"check"`) {
		t.Fatalf("expected metadata to be stored, got: %s", entries[0].Metadata)
	}
}

func TestCreateEntryRejectsInvalidMetadataJSON(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:       "Bad metadata meal",
		Calories:   100,
		ProteinG:   10,
		CarbsG:     10,
		FatG:       2,
		Category:   "breakfast",
		Consumed:   time.Date(2026, 2, 20, 8, 0, 0, 0, time.Local),
		SourceType: "manual",
		Metadata:   `{"tag":`,
	})
	if err == nil {
		t.Fatalf("expected invalid metadata JSON to fail")
	}
}

func TestCreateEntryNormalizesMicronutrientsKeys(t *testing.T) {
	t.Parallel()
	db := newTestDB(t)
	defer db.Close()

	_, err := service.CreateEntry(db, service.CreateEntryInput{
		Name:           "Bad micronutrients meal",
		Calories:       100,
		ProteinG:       10,
		CarbsG:         10,
		FatG:           2,
		Category:       "breakfast",
		Consumed:       time.Date(2026, 2, 20, 8, 0, 0, 0, time.Local),
		SourceType:     "manual",
		Micronutrients: `{"vitamin c":{"value":10,"unit":"mg"}}`,
	})
	if err != nil {
		t.Fatalf("expected micronutrients with spaced key to be normalized, got: %v", err)
	}
	entries, err := service.ListEntries(db, service.ListEntriesFilter{Date: "2026-02-20"})
	if err != nil {
		t.Fatalf("list entries: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected one entry, got %d", len(entries))
	}
	if !strings.Contains(entries[0].Micronutrients, `"vitamin_c"`) {
		t.Fatalf("expected normalized micronutrient key vitamin_c, got: %s", entries[0].Micronutrients)
	}
}
