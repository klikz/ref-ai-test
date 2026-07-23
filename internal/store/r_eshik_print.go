package store

import (
	"database/sql"
	"errors"
	"strings"
)

type EshikPrintSession struct {
	ID               int64  `json:"session_id"`
	EshikComponentID int    `json:"eshik_component_id"`
	FactoryCode      string `json:"factory_code"`
	FullNameUz       string `json:"full_name_uz"`
	SeriyaRaqami     string `json:"seriya_raqami"`
	Index1           string `json:"index1"`
	Index2           string `json:"index2"`
	Serial           string `json:"serial"`
	Counter          int    `json:"counter"`
	UserID           int    `json:"user_id"`
	UserName         string `json:"user_name"`
	CTime            string `json:"c_time"`
}

func (s EshikPrintSession) ToEshikComponent() EshikComponent {
	return EshikComponent{
		ID:           int(s.EshikComponentID),
		FactoryCode:  s.FactoryCode,
		FullNameUz:   s.FullNameUz,
		SeriyaRaqami: s.SeriyaRaqami,
		Index1:       s.Index1,
		Index2:       s.Index2,
	}
}

const eshikPrintSessionSelect = `
		SELECT s.id,
			COALESCE(s.eshik_component_id, 0),
			COALESCE(NULLIF(s.factory_code, ''), cr.factory_code, ''),
			COALESCE(NULLIF(s.full_name_uz, ''), cr.full_name_uz, ''),
			COALESCE(NULLIF(s.seriya_raqami, ''), ec.seriya_raqami, ''),
			COALESCE(NULLIF(s.index1, ''), ec.index1, ''),
			COALESCE(NULLIF(s.index2, ''), ec.index2, ''),
			s.serial,
			s.counter,
			COALESCE(s.user_id, 0),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(s.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.eshik_print_sessions s
		LEFT JOIN production.eshik_components ec ON ec.id = s.eshik_component_id
		LEFT JOIN production.components cr ON cr.id = ec.component_id
		LEFT JOIN auth.users u ON u.id = s.user_id`

func scanEshikPrintSession(scanner interface {
	Scan(dest ...any) error
}) (EshikPrintSession, error) {
	item := EshikPrintSession{}
	err := scanner.Scan(
		&item.ID,
		&item.EshikComponentID,
		&item.FactoryCode,
		&item.FullNameUz,
		&item.SeriyaRaqami,
		&item.Index1,
		&item.Index2,
		&item.Serial,
		&item.Counter,
		&item.UserID,
		&item.UserName,
		&item.CTime,
	)
	return item, err
}

func (r *Repo) EshikPrintSessionCreate(component EshikComponent, serial string, counter, userID int) (int64, error) {
	if component.ID <= 0 {
		return 0, errors.New("eshik komponenti noto'g'ri")
	}
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return 0, errors.New("serial bo'sh")
	}

	var sessionID int64
	err := r.store.db.QueryRow(`
		INSERT INTO production.eshik_print_sessions (
			eshik_component_id, serial, counter, user_id,
			factory_code, full_name_uz, seriya_raqami, index1, index2
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id`,
		component.ID,
		serial,
		counter,
		userID,
		strings.TrimSpace(component.FactoryCode),
		strings.TrimSpace(component.FullNameUz),
		strings.TrimSpace(component.SeriyaRaqami),
		strings.TrimSpace(component.Index1),
		strings.TrimSpace(component.Index2),
	).Scan(&sessionID)
	if err != nil {
		return 0, err
	}
	return sessionID, nil
}

func (r *Repo) EshikPrintSessionDelete(sessionID int64) error {
	if sessionID <= 0 {
		return nil
	}
	_, err := r.store.db.Exec(`DELETE FROM production.eshik_print_sessions WHERE id = $1`, sessionID)
	return err
}

func (r *Repo) EshikPrintSessionGetByID(sessionID int64) (EshikPrintSession, error) {
	item, err := scanEshikPrintSession(r.store.db.QueryRow(eshikPrintSessionSelect+`
		WHERE s.id = $1`, sessionID))
	if err != nil {
		if err == sql.ErrNoRows {
			return item, errors.New("sessiya topilmadi")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) EshikPrintSessionsGetLast(eshikComponentID, limit int) ([]EshikPrintSession, error) {
	if limit <= 0 {
		limit = LastRecordsLimit
	}

	query := eshikPrintSessionSelect
	args := []any{}
	if eshikComponentID > 0 {
		query += `
		WHERE s.eshik_component_id = $1
		ORDER BY s.c_time DESC, s.id DESC
		LIMIT $2`
		args = append(args, eshikComponentID, limit)
	} else {
		query += `
		ORDER BY s.c_time DESC, s.id DESC
		LIMIT $1`
		args = append(args, limit)
	}

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []EshikPrintSession{}
	for rows.Next() {
		item, err := scanEshikPrintSession(rows)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
