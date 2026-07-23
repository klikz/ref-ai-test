package models

type LineLookup struct {
	LineID int    `json:"line_id"`
	Name   string `json:"name"`
}

type ConsumptionNormModelSummary struct {
	ID           int    `json:"id"`
	ModelNomi    string `json:"model_nomi"`
	Modeli       string `json:"modeli"`
	SeriyaRaqami string `json:"seriya_raqami"`
	Brend        string `json:"brend"`
	UmumiyHajmiL string `json:"umumiy_hajmi_l"`
	ItemCount    int    `json:"item_count"`
}

type ConsumptionNormItem struct {
	ID              int     `json:"id"`
	ModelID         int     `json:"model_id"`
	SortOrder       int     `json:"sort_order"`
	GroupLevel      int     `json:"group_level"`
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code"`
	FactoryCode      string  `json:"factory_code"`
	OdooCode        string  `json:"odoo_code"`
	StandardNameUz  string  `json:"standard_name_uz"`
	Quantity        float64 `json:"quantity"`
	ConsumeLineID   int     `json:"consume_line_id"`
	ConsumeLineName string  `json:"consume_line_name"`
	ReceiveLineID   int     `json:"receive_line_id"`
	ReceiveLineName string  `json:"receive_line_name"`
}
