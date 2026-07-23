package models

type WareIncome struct {
	FactoryCode string  `json:"factory_code" xlsx:"factory_code"`
	ComponentId int     `json:"component_id"`
	Quantity    float64 `json:"quantity" xlsx:"quantity"`
}
