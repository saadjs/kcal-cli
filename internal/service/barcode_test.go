package service

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"github.com/saad/kcal-cli/internal/db"
)

type fakeBarcodeClient struct {
	calls int
	item  BarcodeLookupResult
}

func (f *fakeBarcodeClient) LookupBarcode(ctx context.Context, barcode string) (BarcodeLookupResult, []byte, error) {
	_ = ctx
	_ = barcode
	f.calls++
	return f.item, []byte(`{"ok":true}`), nil
}

func TestLookupBarcodeUsesCache(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	client := &fakeBarcodeClient{item: BarcodeLookupResult{
		Description:   "Protein Bar",
		Brand:         "Brand",
		ServingAmount: 1,
		ServingUnit:   "bar",
		Calories:      200,
		ProteinG:      20,
		CarbsG:        20,
		FatG:          7,
		SourceID:      111,
	}}

	_, err := lookupBarcodeWithClient(sqldb, BarcodeProviderUSDA, client, "012345678905")
	if err != nil {
		t.Fatalf("first lookup: %v", err)
	}
	_, err = lookupBarcodeWithClient(sqldb, BarcodeProviderUSDA, client, "012345678905")
	if err != nil {
		t.Fatalf("second lookup: %v", err)
	}
	if client.calls != 1 {
		t.Fatalf("expected 1 provider call due to cache hit, got %d", client.calls)
	}
}

func TestLookupBarcodeValidation(t *testing.T) {
	sqldb := newServiceDB(t)
	defer sqldb.Close()

	client := &fakeBarcodeClient{}
	_, err := lookupBarcodeWithClient(sqldb, BarcodeProviderUSDA, client, "abc")
	if err == nil {
		t.Fatalf("expected invalid barcode to fail")
	}
}

func newServiceDB(t *testing.T) *sql.DB {
	t.Helper()
	path := filepath.Join(t.TempDir(), "kcal.db")
	sqldb, err := db.Open(path)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	if err := db.ApplyMigrations(sqldb); err != nil {
		t.Fatalf("apply migrations: %v", err)
	}
	return sqldb
}
