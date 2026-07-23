package utils

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/lib/pq"
	"github.com/rs/zerolog"
)

// EnrichErrorLog adds structured DB error fields to a zerolog event.
func EnrichErrorLog(event *zerolog.Event, err error) {
	if event == nil || err == nil {
		return
	}

	event.Str("error", err.Error())
	event.Str("error_type", fmt.Sprintf("%T", err))

	if errors.Is(err, sql.ErrNoRows) {
		event.Str("db_kind", "no_rows")
	}

	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		event.Str("db_kind", "postgres")
		event.Str("pg_code", string(pqErr.Code))
		if pqErr.Message != "" {
			event.Str("pg_message", pqErr.Message)
		}
		if pqErr.Detail != "" {
			event.Str("pg_detail", pqErr.Detail)
		}
		if pqErr.Hint != "" {
			event.Str("pg_hint", pqErr.Hint)
		}
		if pqErr.Constraint != "" {
			event.Str("pg_constraint", pqErr.Constraint)
		}
		if pqErr.Table != "" {
			event.Str("pg_table", pqErr.Table)
		}
		if pqErr.Column != "" {
			event.Str("pg_column", pqErr.Column)
		}
		if pqErr.Schema != "" {
			event.Str("pg_schema", pqErr.Schema)
		}
		if pqErr.Where != "" {
			event.Str("pg_where", pqErr.Where)
		}
		if pqErr.Severity != "" {
			event.Str("pg_severity", pqErr.Severity)
		}
		return
	}

	// Include wrapped errors (e.g. fmt.Errorf("context: %w", err)).
	type unwrapper interface{ Unwrap() error }
	current := err
	depth := 0
	for depth < 8 {
		u, ok := current.(unwrapper)
		if !ok {
			break
		}
		next := u.Unwrap()
		if next == nil || next == current {
			break
		}
		depth++
		field := fmt.Sprintf("cause_%d", depth)
		event.Str(field, next.Error())
		event.Str(field+"_type", fmt.Sprintf("%T", next))

		if errors.Is(next, sql.ErrNoRows) {
			event.Str("db_kind", "no_rows")
		}
		var wrappedPQ *pq.Error
		if errors.As(next, &wrappedPQ) {
			event.Str("db_kind", "postgres")
			event.Str("pg_code", string(wrappedPQ.Code))
			if wrappedPQ.Message != "" {
				event.Str("pg_message", wrappedPQ.Message)
			}
			if wrappedPQ.Detail != "" {
				event.Str("pg_detail", wrappedPQ.Detail)
			}
			if wrappedPQ.Constraint != "" {
				event.Str("pg_constraint", wrappedPQ.Constraint)
			}
			if wrappedPQ.Table != "" {
				event.Str("pg_table", wrappedPQ.Table)
			}
			if wrappedPQ.Column != "" {
				event.Str("pg_column", wrappedPQ.Column)
			}
		}
		current = next
	}
}

// FormatDBError returns a developer-friendly error summary for logs.
func FormatDBError(err error) string {
	if err == nil {
		return ""
	}
	if errors.Is(err, sql.ErrNoRows) {
		return "sql: no rows in result set"
	}
	var pqErr *pq.Error
	if errors.As(err, &pqErr) {
		parts := []string{fmt.Sprintf("postgres %s: %s", pqErr.Code, pqErr.Message)}
		if pqErr.Detail != "" {
			parts = append(parts, "detail="+pqErr.Detail)
		}
		if pqErr.Constraint != "" {
			parts = append(parts, "constraint="+pqErr.Constraint)
		}
		if pqErr.Table != "" {
			parts = append(parts, "table="+pqErr.Table)
		}
		if pqErr.Column != "" {
			parts = append(parts, "column="+pqErr.Column)
		}
		return strings.Join(parts, " | ")
	}
	return err.Error()
}
