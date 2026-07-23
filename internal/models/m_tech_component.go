package models

type TechComponent struct {
	ID                    int     `json:"id"`
	DetalTuriKodi         string  `json:"detal_turi_kodi"`
	FactoryCode           string  `json:"factory_code"`
	ManufacturerCode      string  `json:"manufacturer_code"`
	FullNameUz            string  `json:"full_name_uz"`
	StandardNameUz        string  `json:"standard_name_uz"`
	FullNameRu            string  `json:"full_name_ru"`
	StandardNameRu        string  `json:"standard_name_ru"`
	SpecificationUz       string  `json:"specification_uz"`
	Type                  string  `json:"type"`
	TypeId                int     `json:"type_id"`
	Unit                  string  `json:"unit"`
	UnitId                int     `json:"unit_id"`
	NetWeightPcs          float64 `json:"net_weight_pcs"`
	NetWeightSet          float64 `json:"net_weight_set"`
	TechnologicalWastePcs float64 `json:"technological_waste_pcs"`
	TechnologicalWasteSet float64 `json:"technological_waste_set"`
	Comment               string  `json:"comment"`
	Available             float64 `json:"available"`
	OdooCode              string  `json:"odoo_code"`
	PhotoPath             string  `json:"photo_path"`

	// Deprecated aliases kept for older handlers and API clients.
	NetWeightKg        float64 `json:"net_weight_kg,omitempty"`
	TechnologicalWaste float64 `json:"technological_waste,omitempty"`
	MainCode           string  `json:"main_code,omitempty"`
	NameLong           string  `json:"name_long,omitempty"`
	NameShortUz        string  `json:"name_short_uz,omitempty"`
	SpecsUz            string  `json:"specs_uz,omitempty"`
	TechWaste          float64 `json:"tech_waste,omitempty"`
	NameRus            string  `json:"name_rus,omitempty"`
	Weight             float64 `json:"weight,omitempty"`
}
