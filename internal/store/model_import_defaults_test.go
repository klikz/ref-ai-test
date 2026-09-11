package store

import (
	"testing"

	"github.com/klikz/api_v3/internal/models"
)

func TestApplyModelImportSerialDefaultsEmptyDoors(t *testing.T) {
	model := models.ModelInfo{
		Acc_serial:        "  ",
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
	if model.Freeze_door_code != "*" {
		t.Fatalf("freeze_door_code = %q, want *", model.Freeze_door_code)
	}
	if model.Ref_door_code != "*" {
		t.Fatalf("ref_door_code = %q, want *", model.Ref_door_code)
	}
}

func TestApplyModelImportSerialDefaultsLegacyDoor(t *testing.T) {
	model := models.ModelInfo{
		Acc_serial:        "AC01",
		Compressor_serial: "CP01",
		Door_code:         "DR01, DR02",
	}
	ApplyModelImportSerialDefaults(&model)
	if model.Freeze_door_code != "DR01\nDR02" {
		t.Fatalf("freeze_door_code = %q", model.Freeze_door_code)
	}
	if model.Ref_door_code != "DR01\nDR02" {
		t.Fatalf("ref_door_code = %q", model.Ref_door_code)
	}
}

func TestApplyModelImportSerialDefaultsSeparateDoors(t *testing.T) {
	model := models.ModelInfo{
		Acc_serial:         "*",
		Compressor_serial:  "*",
		Freeze_door_code:   "FR01",
		Ref_door_code:      "RF01, RF02",
		Door_code:          "IGNORE",
	}
	ApplyModelImportSerialDefaults(&model)
	if model.Freeze_door_code != "FR01" {
		t.Fatalf("freeze_door_code = %q", model.Freeze_door_code)
	}
	if model.Ref_door_code != "RF01\nRF02" {
		t.Fatalf("ref_door_code = %q", model.Ref_door_code)
	}
}
