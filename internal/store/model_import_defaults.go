package store

import (
	"strings"

	"github.com/klikz/api_v3/internal/models"
)

// ApplyModelImportSerialDefaults sets acc_serial, compressor_serial and door_code
// to "*" when they are empty after normalization (Excel import / bulk upsert).
func ApplyModelImportSerialDefaults(model *models.ModelInfo) {
	if model == nil {
		return
	}

	model.Acc_serial = NormalizeModelAccSerialStorage(model.Acc_serial)
	if len(ParseModelAccSerialPrefixes(model.Acc_serial)) == 0 {
		model.Acc_serial = "*"
	}

	model.Compressor_serial = NormalizeModelCompressorSerialStorage(model.Compressor_serial)
	if len(ParseModelAccSerialPrefixes(model.Compressor_serial)) == 0 {
		model.Compressor_serial = "*"
	}

	model.Door_code = NormalizeModelDoorCodeStorage(model.Door_code)
	if len(ParseModelAccSerialPrefixes(model.Door_code)) == 0 {
		model.Door_code = "*"
	}
}

func NormalizeModelDoorCodeStorage(raw string) string {
	return strings.Join(ParseModelAccSerialPrefixes(raw), "\n")
}

func modelAccSerialWildcard(stored string) bool {
	for _, prefix := range ParseModelAccSerialPrefixes(stored) {
		if prefix == "*" {
			return true
		}
	}
	return false
}
