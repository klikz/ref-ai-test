package models

type User struct {
	ID                int     `json:"id"`
	UserName          string  `json:"name"`
	Login             string  `json:"login"`
	EncryptedPassword string  `json:"-"`
	Password          string  `json:"password,omitempty"`
	Role              string  `json:"role,omitempty"`
	Token             string  `json:"token,omitempty"`
	Status            bool    `json:"status,omitempty"`
	Role_ID           float64 `json:"role_id"`
}

type ParsedToken struct {
	Login  string `json:"login"`
	UserID int    `json:"user_id"`
}
