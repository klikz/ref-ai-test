package store

import (
	"testing"

	"github.com/xuri/excelize/v2"
)

func TestPlanEnsureSheetCreatesMissingSheet(t *testing.T) {
	f := excelize.NewFile()
	defer f.Close()

	idx, err := f.GetSheetIndex("T1")
	if err != nil {
		t.Fatalf("GetSheetIndex err: %v", err)
	}
	if idx != -1 {
		t.Fatalf("expected -1 for missing sheet, got %d", idx)
	}

	if err := planEnsureSheet(f, "T1"); err != nil {
		t.Fatalf("planEnsureSheet: %v", err)
	}

	idx, err = f.GetSheetIndex("T1")
	if err != nil {
		t.Fatalf("GetSheetIndex after create: %v", err)
	}
	if idx < 0 {
		t.Fatalf("T1 sheet should exist, idx=%d", idx)
	}
}
