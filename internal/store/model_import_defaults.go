package store

import (
	"strings"

	"github.com/klikz/api_v3/internal/models"
)

// ApplyModelImportSerialDefaults sets acc_serial, compressor_serial and door codes
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

	// Legacy single door_code fills both when new fields empty.
	legacy := NormalizeModelDoorCodeStorage(model.Door_code)
	model.Freeze_door_code = NormalizeModelDoorCodeStorage(model.Freeze_door_code)
	model.Ref_door_code = NormalizeModelDoorCodeStorage(model.Ref_door_code)
	if len(ParseModelAccSerialPrefixes(model.Freeze_door_code)) == 0 {
		if len(ParseModelAccSerialPrefixes(legacy)) > 0 {
			model.Freeze_door_code = legacy
		} else {
			model.Freeze_door_code = "*"
		}
	}
	if len(ParseModelAccSerialPrefixes(model.Ref_door_code)) == 0 {
		if len(ParseModelAccSerialPrefixes(legacy)) > 0 {
			model.Ref_door_code = legacy
		} else {
			model.Ref_door_code = "*"
		}
	}
	model.Door_code = model.Freeze_door_code

	model.RangiKodi = NormalizeRangiKodi(model.RangiKodi)
}

// NormalizeRangiKodi keeps digits only. Excel often emits 101 as "101.0".
func NormalizeRangiKodi(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}
	s = strings.ReplaceAll(s, " ", "")
	s = strings.ReplaceAll(s, ",", ".")
	if strings.HasSuffix(s, ".0") {
		s = strings.TrimSuffix(s, ".0")
	}
	var b strings.Builder
	for _, r := range s {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
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
