package models

type PrinterV2 struct {
	ID                int    `json:"id"`
	LineID            int    `json:"line_id"`
	LineName          string `json:"line_name"`
	PrinterName       string `json:"printer_name"`
	Address           string `json:"address"`
	LabelTemplateID   int    `json:"label_template_id"`
	LabelTemplateName string `json:"label_template_name"`
	PrintLanguage     string `json:"print_language"`
}
