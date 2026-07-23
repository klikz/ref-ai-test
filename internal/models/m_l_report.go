package models

type LReport struct {
	Id        int    `json:"id"`
	Serial    string `json:"serial"`
	AccSerial string `json:"acc_serial"`
	ModelId   int    `json:"model_id"`
	Model     string `json:"model"`
	ModelNomi string `json:"model_nomi"`
	OdooCode  string `json:"odoo_code"`
	LineID    int    `json:"line_id"`
	LineName  string `json:"line_name"`
	Time      string `json:"time"`
	Gs1       string `json:"gs1"`
	GsCode    string `json:"gs_code"`
}
