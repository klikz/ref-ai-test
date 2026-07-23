package store

import (
	"database/sql"
	"errors"
	"strings"
)

type KlapanPrintSession struct {
	ID                int64  `json:"session_id"`
	KlapanComponentID int    `json:"klapan_component_id"`
	FactoryCode       string `json:"factory_code"`
	FullNameUz        string `json:"full_name_uz"`
	SeriyaRaqami      string `json:"seriya_raqami"`
	Index1            string `json:"index1"`
	Index2            string `json:"index2"`
	Serial            string `json:"serial"`
	Counter           int    `json:"counter"`
	UserID            int    `json:"user_id"`
	UserName          string `json:"user_name"`
	CTime             string `json:"c_time"`
}

func (s KlapanPrintSession) ToKlapanComponent() KlapanComponent {
	return KlapanComponent{
		ID:           int(s.KlapanComponentID),
		FactoryCode:  s.FactoryCode,
		FullNameUz:   s.FullNameUz,
		SeriyaRaqami: s.SeriyaRaqami,
		Index1:       s.Index1,
		Index2:       s.Index2,
	}
}

const klapanPrintSessionSelect = `
		SELECT s.id,
			COALESCE(s.klapan_component_id, 0),
			COALESCE(NULLIF(s.factory_code, ''), cr.factory_code, ''),
			COALESCE(NULLIF(s.full_name_uz, ''), cr.full_name_uz, ''),
			COALESCE(NULLIF(s.seriya_raqami, ''), kc.seriya_raqami, ''),
			COALESCE(NULLIF(s.index1, ''), kc.index1, ''),
			COALESCE(NULLIF(s.index2, ''), kc.index2, ''),
			s.serial,
			s.counter,
			COALESCE(s.user_id, 0),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(s.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.klapan_print_sessions s
		LEFT JOIN production.klapan_components kc ON kc.id = s.klapan_component_id
		LEFT JOIN production.components cr ON cr.id = kc.component_id
		LEFT JOIN auth.users u ON u.id = s.user_id`

func scanKlapanPrintSession(scanner interface {
	Scan(dest ...any) error
}) (KlapanPrintSession, error) {
	item := KlapanPrintSession{}
	err := scanner.Scan(
		&item.ID,
		&item.KlapanComponentID,
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

func (r *Repo) KlapanPrintSessionCreate(component KlapanComponent, serial string, counter, userID int) (int64, error) {
	if component.ID <= 0 {
		return 0, errors.New("klapan komponenti noto'g'ri")
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
		INSERT INTO production.klapan_print_sessions (
			klapan_component_id, serial, counter, user_id,
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

func (r *Repo) KlapanPrintSessionGetByID(sessionID int64) (KlapanPrintSession, error) {
	item, err := scanKlapanPrintSession(r.store.db.QueryRow(klapanPrintSessionSelect+`
		WHERE s.id = $1`, sessionID))
	if err != nil {
		if err == sql.ErrNoRows {
			return item, errors.New("sessiya topilmadi")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) KlapanPrintSessionGetBySerial(serial string) (KlapanPrintSession, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return KlapanPrintSession{}, errors.New("serial bo'sh")
	}

	item, err := scanKlapanPrintSession(r.store.db.QueryRow(klapanPrintSessionSelect+`
		WHERE s.serial = $1
		ORDER BY s.c_time DESC, s.id DESC
		LIMIT 1`, serial))
	if err != nil {
		if err == sql.ErrNoRows {
			return item, errors.New("serial topilmadi")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) KlapanPrintSessionsGetLast(klapanComponentID, limit int) ([]KlapanPrintSession, error) {
	if limit <= 0 {
		limit = LastRecordsLimit
	}

	query := klapanPrintSessionSelect
	args := []any{}
	if klapanComponentID > 0 {
		query += `
		WHERE s.klapan_component_id = $1
		ORDER BY s.c_time DESC, s.id DESC
		LIMIT $2`
		args = append(args, klapanComponentID, limit)
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

	items := []KlapanPrintSession{}
	for rows.Next() {
		item, err := scanKlapanPrintSession(rows)
		if err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}
