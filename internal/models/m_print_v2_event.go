package models

import (
	"encoding/json"
	"time"
)

type PrintV2Event struct {
	ID                int64           `json:"id"`
	CreatedAt         time.Time       `json:"created_at"`
	OK                bool            `json:"ok"`
	DurationMs        int             `json:"duration_ms"`
	LineID            int             `json:"line_id"`
	LineName          string          `json:"line_name"`
	PrinterV2ID       int             `json:"printer_v2_id"`
	PrinterName       string          `json:"printer_name"`
	TemplateID        int             `json:"template_id"`
	PrintLanguage     string          `json:"print_language"`
	EffectiveLanguage string          `json:"effective_language"`
	Serial            string          `json:"serial"`
	Stage             string          `json:"stage"`
	ErrorMessage      string          `json:"error_message"`
	ErrorDetail       string          `json:"error_detail"`
	Meta              json.RawMessage `json:"meta,omitempty"`
}

type PrintV2MetricsSummary struct {
	Success  int64  `json:"success"`
	Fail     int64  `json:"fail"`
	TotalMs  int64  `json:"total_ms"`
	LastMs   int64  `json:"last_ms"`
	InFlight int64  `json:"in_flight"`
	DateFrom string `json:"date_from,omitempty"`
	DateTo   string `json:"date_to,omitempty"`
}

type PrintV2LineMetrics struct {
	LineID   int    `json:"line_id"`
	LineName string `json:"line_name"`
	Success  int64  `json:"success"`
	Fail     int64  `json:"fail"`
	TotalMs  int64  `json:"total_ms"`
	LastMs   int64  `json:"last_ms"`
}
