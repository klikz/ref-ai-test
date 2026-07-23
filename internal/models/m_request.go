package models

type Request struct {
	ID       int    `json:"id"`
	UserName string `json:"username"`
	Password string `json:"password"`
	Token    string `json:"token,omitempty"`
	Name     string `json:"name"`
}
