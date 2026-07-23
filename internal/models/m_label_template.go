package models

import "encoding/json"

type LabelTemplate struct {
	ID               int             `json:"id"`
	Name             string          `json:"name"`
	LineID           int             `json:"line_id"`
	LineName         string          `json:"line_name,omitempty"`
	WidthMm          float64         `json:"width_mm"`
	HeightMm         float64         `json:"height_mm"`
	DPI              int             `json:"dpi"`
	PrintRotationDeg int             `json:"print_rotation_deg"`
	Definition       json.RawMessage `json:"definition"`
}
