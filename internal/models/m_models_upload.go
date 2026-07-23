package models

type IncomeFile struct {
	MainCode    string  `json:"main_code"`
	FactoryCode string  `json:"factory_code"`
	NameLong    string  `json:"detal_nomi"`
	NameShortUz string  `json:"nomi_uz"`
	NameShortEn string  `json:"nomi_en"`
	SpecsUz     string  `json:"specs_uz"`
	SpecsEn     string  `json:"specs_en"`
	TypeId      int     `json:"type_id"`
	UnitId      int     `json:"unit_id"`
	TechWaste   float64 `json:"tech_waste"`
	NgWaste     float64 `json:"ng_waste"`
	Comment     string  `json:"comment"`
}
