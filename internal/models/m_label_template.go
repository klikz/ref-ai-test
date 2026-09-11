package models

import "encoding/json"

type LabelTemplate struct {
	ID                 int             `json:"id"`
	Name               string          `json:"name"`
	LineID             int             `json:"line_id"`
	LineName           string          `json:"line_name,omitempty"`
	WidthMm            float64         `json:"width_mm"`
	HeightMm           float64         `json:"height_mm"`
	DPI                int             `json:"dpi"`
	PrintRotationDeg   int             `json:"print_rotation_deg"`
	Density            int             `json:"density"`
	Speed              int             `json:"speed"`
	GapMm              float64         `json:"gap_mm"`
	UsePrinterDefaults bool            `json:"use_printer_defaults"`
	SizeOnly           bool            `json:"size_only"`
	Definition         json.RawMessage `json:"definition"`
}
