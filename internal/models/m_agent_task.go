package models

import "time"

type AgentTask struct {
	ID        int64     `json:"id"`
	Prompt    string    `json:"prompt"`
	Origin    string    `json:"origin"`
	Status    string    `json:"status"`
	CreatedBy int       `json:"created_by"`
	GitSHA    string    `json:"git_sha"`
	LogText   string    `json:"log_text"`
	ErrorText string    `json:"error_text"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
