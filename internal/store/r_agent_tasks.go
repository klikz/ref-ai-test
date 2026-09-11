package store

import (
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/klikz/api_v3/internal/models"
)

func (r *Repo) AgentTaskList(limit int) ([]models.AgentTask, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := r.store.db.Query(`
		SELECT id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at
		FROM agent.tasks
		ORDER BY id DESC
		LIMIT $1
	`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make([]models.AgentTask, 0)
	for rows.Next() {
		item := models.AgentTask{}
		if err := rows.Scan(
			&item.ID, &item.Prompt, &item.Origin, &item.Status, &item.CreatedBy,
			&item.GitSHA, &item.LogText, &item.ErrorText, &item.CreatedAt, &item.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repo) AgentTaskByID(id int64) (models.AgentTask, error) {
	item := models.AgentTask{}
	err := r.store.db.QueryRow(`
		SELECT id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at
		FROM agent.tasks
		WHERE id = $1
	`, id).Scan(
		&item.ID, &item.Prompt, &item.Origin, &item.Status, &item.CreatedBy,
		&item.GitSHA, &item.LogText, &item.ErrorText, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (r *Repo) AgentTaskHasActive() (bool, error) {
	var exists bool
	err := r.store.db.QueryRow(`
		SELECT EXISTS (
			SELECT 1 FROM agent.tasks
			WHERE status IN ('queued', 'running', 'testing', 'promoting')
		)
	`).Scan(&exists)
	return exists, err
}

func (r *Repo) AgentTaskCreate(prompt, origin string, createdBy int) (models.AgentTask, error) {
	prompt = strings.TrimSpace(prompt)
	origin = strings.TrimSpace(strings.ToLower(origin))
	if prompt == "" {
		return models.AgentTask{}, errors.New("prompt is required")
	}
	if origin != "pc" && origin != "server" {
		origin = "server"
	}

	active, err := r.AgentTaskHasActive()
	if err != nil {
		return models.AgentTask{}, err
	}
	if active {
		return models.AgentTask{}, errors.New("vazifa bajarilmoqda — galma-gal ishlating")
	}

	item := models.AgentTask{}
	err = r.store.db.QueryRow(`
		INSERT INTO agent.tasks (prompt, origin, status, created_by)
		VALUES ($1, $2, 'queued', $3)
		RETURNING id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at
	`, prompt, origin, createdBy).Scan(
		&item.ID, &item.Prompt, &item.Origin, &item.Status, &item.CreatedBy,
		&item.GitSHA, &item.LogText, &item.ErrorText, &item.CreatedAt, &item.UpdatedAt,
	)
	return item, err
}

func (r *Repo) AgentTaskClaimNext() (models.AgentTask, error) {
	tx, err := r.store.db.Begin()
	if err != nil {
		return models.AgentTask{}, err
	}
	defer func() { _ = tx.Rollback() }()

	item := models.AgentTask{}
	err = tx.QueryRow(`
		SELECT id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at
		FROM agent.tasks
		WHERE status = 'queued' AND origin = 'server'
		ORDER BY id ASC
		FOR UPDATE SKIP LOCKED
		LIMIT 1
	`).Scan(
		&item.ID, &item.Prompt, &item.Origin, &item.Status, &item.CreatedBy,
		&item.GitSHA, &item.LogText, &item.ErrorText, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return models.AgentTask{}, sql.ErrNoRows
	}
	if err != nil {
		return models.AgentTask{}, err
	}

	err = tx.QueryRow(`
		UPDATE agent.tasks
		SET status = 'running',
		    log_text = 'Cursor SDK worker ishlamoqda...',
		    updated_at = $2
		WHERE id = $1
		RETURNING id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at
	`, item.ID, time.Now()).Scan(
		&item.ID, &item.Prompt, &item.Origin, &item.Status, &item.CreatedBy,
		&item.GitSHA, &item.LogText, &item.ErrorText, &item.CreatedAt, &item.UpdatedAt,
	)
	if err != nil {
		return models.AgentTask{}, err
	}
	if err := tx.Commit(); err != nil {
		return models.AgentTask{}, err
	}
	return item, nil
}

func (r *Repo) AgentTaskUpdateStatus(id int64, status, logText, errorText, gitSHA string) (models.AgentTask, error) {
	item := models.AgentTask{}
	err := r.store.db.QueryRow(`
		UPDATE agent.tasks
		SET status = $2,
		    log_text = CASE WHEN $3 = '' THEN log_text ELSE $3 END,
		    error_text = CASE WHEN $4 = '' THEN error_text ELSE $4 END,
		    git_sha = CASE WHEN $5 = '' THEN git_sha ELSE $5 END,
		    updated_at = $6
		WHERE id = $1
		RETURNING id, prompt, origin, status, created_by, git_sha, log_text, error_text, created_at, updated_at
	`, id, status, logText, errorText, gitSHA, time.Now()).Scan(
		&item.ID, &item.Prompt, &item.Origin, &item.Status, &item.CreatedBy,
		&item.GitSHA, &item.LogText, &item.ErrorText, &item.CreatedAt, &item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return item, errors.New("vazifa topilmadi")
	}
	return item, err
}
