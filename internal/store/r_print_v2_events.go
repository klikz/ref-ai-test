package store

import (
	"encoding/json"
	"strings"
	"time"

	"github.com/klikz/api_v3/internal/models"
)

func (r *Repo) PrintV2EventInsert(ev models.PrintV2Event) error {
	var meta any
	if len(ev.Meta) > 0 {
		meta = ev.Meta
	}
	var printerID any
	if ev.PrinterV2ID > 0 {
		printerID = ev.PrinterV2ID
	}
	var templateID any
	if ev.TemplateID > 0 {
		templateID = ev.TemplateID
	}
	var lineID any
	if ev.LineID > 0 {
		lineID = ev.LineID
	}
	_, err := r.store.db.Exec(`
		INSERT INTO lines.print_v2_events (
			ok, duration_ms, line_id, line_name, printer_v2_id, printer_name, template_id,
			print_language, effective_language, serial, stage,
			error_message, error_detail, meta
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14)`,
		ev.OK, ev.DurationMs, lineID, ev.LineName, printerID, ev.PrinterName, templateID,
		ev.PrintLanguage, ev.EffectiveLanguage, ev.Serial, ev.Stage,
		ev.ErrorMessage, ev.ErrorDetail, meta,
	)
	return err
}

func (r *Repo) PrintV2MetricsSummary(from, to time.Time) (models.PrintV2MetricsSummary, error) {
	var s models.PrintV2MetricsSummary
	err := r.store.db.QueryRow(`
		SELECT
			COALESCE(SUM(CASE WHEN ok THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(CASE WHEN NOT ok THEN 1 ELSE 0 END), 0),
			COALESCE(SUM(duration_ms), 0),
			COALESCE((
				SELECT duration_ms FROM lines.print_v2_events
				WHERE created_at >= $1 AND created_at < $2
				ORDER BY created_at DESC LIMIT 1
			), 0)
		FROM lines.print_v2_events
		WHERE created_at >= $1 AND created_at < $2`,
		from, to,
	).Scan(&s.Success, &s.Fail, &s.TotalMs, &s.LastMs)
	return s, err
}

func (r *Repo) PrintV2MetricsByLine(from, to time.Time) ([]models.PrintV2LineMetrics, error) {
	// Fixed REF production lines for metrics UI.
	rows, err := r.store.db.Query(`
		WITH metric_lines(ord, line_id, label) AS (
			VALUES
				(1, 1,  'Boshlang''ich yig''uv'),
				(2, 20, 'Eshik yig''uv'),
				(3, 12, 'Yakuniy yig''uv')
		),
		agg AS (
			SELECT
				COALESCE(e.line_id, 0) AS line_id,
				COALESCE(SUM(CASE WHEN e.ok THEN 1 ELSE 0 END), 0) AS success,
				COALESCE(SUM(CASE WHEN NOT e.ok THEN 1 ELSE 0 END), 0) AS fail,
				COALESCE(SUM(e.duration_ms), 0) AS total_ms,
				COALESCE((
					ARRAY_AGG(e.duration_ms ORDER BY e.created_at DESC)
				)[1], 0) AS last_ms
			FROM lines.print_v2_events e
			WHERE e.created_at >= $1 AND e.created_at < $2
			GROUP BY COALESCE(e.line_id, 0)
		)
		SELECT
			m.line_id,
			COALESCE(NULLIF(trim(ll.name), ''), m.label) AS line_name,
			COALESCE(a.success, 0),
			COALESCE(a.fail, 0),
			COALESCE(a.total_ms, 0),
			COALESCE(a.last_ms, 0)
		FROM metric_lines m
		LEFT JOIN lines.lines_list ll ON ll.line_id = m.line_id
		LEFT JOIN agg a ON a.line_id = m.line_id
		ORDER BY m.ord`,
		from, to,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.PrintV2LineMetrics, 0, 3)
	for rows.Next() {
		var m models.PrintV2LineMetrics
		if err := rows.Scan(&m.LineID, &m.LineName, &m.Success, &m.Fail, &m.TotalMs, &m.LastMs); err != nil {
			return nil, err
		}
		items = append(items, m)
	}
	return items, rows.Err()
}

func (r *Repo) PrintV2EventsList(from, to time.Time, onlyErrors bool, limit, offset int) ([]models.PrintV2Event, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	if offset < 0 {
		offset = 0
	}
	q := `
		SELECT id, created_at, ok, duration_ms,
		       COALESCE(line_id, 0), COALESCE(line_name, ''),
		       COALESCE(printer_v2_id, 0), printer_name, COALESCE(template_id, 0),
		       print_language, effective_language, serial, stage,
		       error_message, error_detail, COALESCE(meta, '{}'::jsonb)
		FROM lines.print_v2_events
		WHERE created_at >= $1 AND created_at < $2`
	args := []any{from, to}
	if onlyErrors {
		q += ` AND ok = false`
	}
	q += ` ORDER BY created_at DESC LIMIT $3 OFFSET $4`
	args = append(args, limit, offset)

	rows, err := r.store.db.Query(q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.PrintV2Event, 0)
	for rows.Next() {
		var ev models.PrintV2Event
		var meta []byte
		if err := rows.Scan(
			&ev.ID, &ev.CreatedAt, &ev.OK, &ev.DurationMs,
			&ev.LineID, &ev.LineName,
			&ev.PrinterV2ID, &ev.PrinterName, &ev.TemplateID,
			&ev.PrintLanguage, &ev.EffectiveLanguage, &ev.Serial, &ev.Stage,
			&ev.ErrorMessage, &ev.ErrorDetail, &meta,
		); err != nil {
			return nil, err
		}
		if len(meta) > 0 {
			ev.Meta = json.RawMessage(meta)
		}
		items = append(items, ev)
	}
	return items, rows.Err()
}

func (r *Repo) PrintV2EventsReset(from, to time.Time) (int64, error) {
	res, err := r.store.db.Exec(`
		DELETE FROM lines.print_v2_events
		WHERE created_at >= $1 AND created_at < $2`, from, to)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ParseMetricsDateRange parses YYYY-MM-DD (local calendar days) into [from, to).
// Empty from/to defaults to today local.
func ParseMetricsDateRange(dateFrom, dateTo string) (time.Time, time.Time, error) {
	loc := time.Local
	now := time.Now().In(loc)
	startOfDay := func(t time.Time) time.Time {
		y, m, d := t.Date()
		return time.Date(y, m, d, 0, 0, 0, 0, loc)
	}

	from := startOfDay(now)
	to := from.Add(24 * time.Hour)

	dateFrom = strings.TrimSpace(dateFrom)
	dateTo = strings.TrimSpace(dateTo)

	if dateFrom != "" {
		t, err := time.ParseInLocation("2006-01-02", dateFrom, loc)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		from = startOfDay(t)
	}
	if dateTo != "" {
		t, err := time.ParseInLocation("2006-01-02", dateTo, loc)
		if err != nil {
			return time.Time{}, time.Time{}, err
		}
		to = startOfDay(t).Add(24 * time.Hour)
	} else if dateFrom != "" {
		to = from.Add(24 * time.Hour)
	}
	if !to.After(from) {
		to = from.Add(24 * time.Hour)
	}
	return from, to, nil
}
