package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type T3ScanPhoto struct {
	ID         int64  `json:"id"`
	ProductID  int    `json:"product_id"`
	LineID     int    `json:"line_id"`
	Serial     string `json:"serial"`
	FilePath   string `json:"file_path"`
	CapturedAt string `json:"captured_at"`
	UserID     int    `json:"user_id"`
}

func (r *Repo) T3ScanPhotoInsert(productID, lineID, userID int, serial, filePath string) (int64, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		serial = "test"
	}
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return 0, fmt.Errorf("file_path bo'sh")
	}

	var productIDArg any
	if productID > 0 {
		productIDArg = productID
	}

	var id int64
	err := r.store.db.QueryRow(`
		INSERT INTO lines.packing_scan_photos (product_id, line_id, serial, file_path, user_id)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING id`,
		productIDArg, lineID, serial, filePath, nullIfZero(userID),
	).Scan(&id)
	return id, err
}

func (r *Repo) T3ScanPhotoDelete(id int64) (string, error) {
	if id <= 0 {
		return "", nil
	}
	var filePath string
	err := r.store.db.QueryRow(`
		DELETE FROM lines.packing_scan_photos
		WHERE id = $1
		RETURNING file_path`,
		id,
	).Scan(&filePath)
	if errors.Is(err, sql.ErrNoRows) {
		return "", nil
	}
	return filePath, err
}

func (r *Repo) T3ScanPhotosList(dateFrom, dateTo string, limit, offset int) ([]T3ScanPhoto, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 500 {
		limit = 500
	}
	if offset < 0 {
		offset = 0
	}

	query := `
		SELECT id, COALESCE(product_id, 0), line_id, serial, file_path,
			to_char(captured_at, 'YYYY-MM-DD HH24:MI:SS'),
			COALESCE(user_id, 0)
		FROM lines.packing_scan_photos
		WHERE 1=1`
	args := []any{}
	argN := 1

	if strings.TrimSpace(dateFrom) != "" {
		query += fmt.Sprintf(" AND captured_at >= $%d::date", argN)
		args = append(args, dateFrom)
		argN++
	}
	if strings.TrimSpace(dateTo) != "" {
		query += fmt.Sprintf(" AND captured_at < ($%d::date + INTERVAL '1 day')", argN)
		args = append(args, dateTo)
		argN++
	}

	query += fmt.Sprintf(" ORDER BY captured_at DESC LIMIT $%d OFFSET $%d", argN, argN+1)
	args = append(args, limit, offset)

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []T3ScanPhoto{}
	for rows.Next() {
		row := T3ScanPhoto{}
		if err := rows.Scan(
			&row.ID, &row.ProductID, &row.LineID, &row.Serial, &row.FilePath,
			&row.CapturedAt, &row.UserID,
		); err != nil {
			return items, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func (r *Repo) T3ScanPhotoLatestBySerial(serial string) (*T3ScanPhoto, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil, nil
	}

	row := T3ScanPhoto{}
	err := r.store.db.QueryRow(`
		SELECT id, COALESCE(product_id, 0), line_id, serial, file_path,
			to_char(captured_at, 'YYYY-MM-DD HH24:MI:SS'),
			COALESCE(user_id, 0)
		FROM lines.packing_scan_photos
		WHERE serial = $1
		ORDER BY captured_at DESC
		LIMIT 1`,
		serial,
	).Scan(
		&row.ID, &row.ProductID, &row.LineID, &row.Serial, &row.FilePath,
		&row.CapturedAt, &row.UserID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &row, nil
}

func sanitizeT3ScanSerial(label string) string {
	label = strings.TrimSpace(label)
	if label == "" {
		return "test"
	}
	var b strings.Builder
	for _, r := range label {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r >= '0' && r <= '9', r == '_', r == '-':
			b.WriteRune(r)
		}
	}
	if b.Len() == 0 {
		return "test"
	}
	return b.String()
}

func nullIfZero(v int) sql.NullInt64 {
	if v <= 0 {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(v), Valid: true}
}
