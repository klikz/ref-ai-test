package store

import (
	"testing"

	"github.com/klikz/api_v3/internal/models"
)

func TestApplyModelImportSerialDefaults(t *testing.T) {
	model := models.ModelInfo{
		Acc_serial:        " ",
		Compressor_serial: "",
		Door_code:         "  ",
	}
	ApplyModelImportSerialDefaults(&model)
	if model.Acc_serial != "*" {
		t.Fatalf("acc_serial = %q, want *", model.Acc_serial)
	}
	if model.Compressor_serial != "*" {
		t.Fatalf("compressor_serial = %q, want *", model.Compressor_serial)
	}
	if model.Door_code != "*" {
		t.Fatalf("door_code = %q, want *", model.Door_code)
	}
}

func TestApplyModelImportSerialDefaultsKeepsValues(t *testing.T) {
	model := models.ModelInfo{
		Acc_serial:        "ACC1, ACC2",
		Compressor_serial: "CMP1; CMP2",
		Door_code:         "DR01, DR02",
	}
	ApplyModelImportSerialDefaults(&model)
	if model.Acc_serial != "ACC1\nACC2" {
		t.Fatalf("acc_serial = %q", model.Acc_serial)
	}
	if model.Compressor_serial != "CMP1\nCMP2" {
		t.Fatalf("compressor_serial = %q", model.Compressor_serial)
	}
	if model.Door_code != "DR01\nDR02" {
		t.Fatalf("door_code = %q", model.Door_code)
	}
}

func TestModelAccSerialMatchesWildcard(t *testing.T) {
	if !ModelAccSerialMatches("ANY123", "*") {
		t.Fatal("wildcard acc_serial should match any scanned value")
	}
}
