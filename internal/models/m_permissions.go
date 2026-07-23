package models

type Permissions struct {
	ID      int    `json:"id"`
	Route   string `json:"route"`
	Comment string `json:"comment"`
	IsFlag  bool   `json:"is_flag"`
}
