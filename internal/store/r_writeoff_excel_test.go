package store

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestWriteoffWorkbookRoundTrip(t *testing.T) {
	lines := []WriteoffCatalogItem{
		{LineID: 4, LineName: "T1", Label: "T1", ItemType: "line"},
		{LineID: 8, LineName: "Fin Press", Label: "Fin Press", ItemType: "line"},
	}
	catalogs := map[int][]WriteoffCatalogItem{
		8: {{ItemID: 201, ItemType: WriteoffItemComponent, Label: "Comp B", LineID: 8}},
	}

	data, err := buildWriteoffWorkbook(0, nil, lines, catalogs)
	if err != nil {
		t.Fatalf("buildWriteoffWorkbook: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("empty workbook bytes")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("OpenReader: %v", err)
	}
	defer f.Close()

	if idx, _ := f.GetSheetIndex("Writeoff"); idx < 0 {
		t.Fatal("Writeoff sheet missing")
	}

	path := filepath.Join(t.TempDir(), "writeoff_template.xlsx")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
}
