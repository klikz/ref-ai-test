package store

import (
	"database/sql"
	"errors"
	"strings"
)

type EshikPrintSession struct {
	ID               int64  `json:"session_id"`
	EshikModelID     int    `json:"eshik_model_id"`
	EshikModelPartID int    `json:"eshik_model_part_id"`
	DoorCode         string `json:"door_code"`
	ModelName        string `json:"model_name"`
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

func (s EshikPrintSession) ToEshikModelPart() EshikModelPart {
	return EshikModelPart{
		ID:           s.EshikModelPartID,
		EshikModelID: s.EshikModelID,
		DoorCode:     normalizeEshikDoorCode(s.DoorCode),
		FactoryCode:  s.FactoryCode,
		FullNameUz:   s.FullNameUz,
		SeriyaRaqami: s.SeriyaRaqami,
		Index1:       s.Index1,
		Index2:       s.Index2,
	}
}

const eshikPrintSessionSelect = `
		SELECT s.id,
			COALESCE(s.eshik_model_id, p.eshik_model_id, 0),
			COALESCE(s.eshik_model_part_id, 0),
			COALESCE(NULLIF(s.door_code, ''), p.door_code, ''),
			COALESCE(NULLIF(s.model_name, ''), m.model_name, ''),
			COALESCE(NULLIF(s.factory_code, ''), cr.factory_code, ''),
			COALESCE(NULLIF(s.full_name_uz, ''), cr.full_name_uz, ''),
			COALESCE(NULLIF(s.seriya_raqami, ''), p.seriya_raqami, ''),
			COALESCE(NULLIF(s.index1, ''), p.index1, ''),
			COALESCE(NULLIF(s.index2, ''), p.index2, ''),
			s.serial,
			s.counter,
			COALESCE(s.user_id, 0),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(s.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.eshik_print_sessions s
		LEFT JOIN production.eshik_model_parts p ON p.id = s.eshik_model_part_id
		LEFT JOIN production.eshik_models m ON m.id = COALESCE(s.eshik_model_id, p.eshik_model_id)
		LEFT JOIN production.components cr ON cr.id = p.component_id
		LEFT JOIN auth.users u ON u.id = s.user_id`

func scanEshikPrintSession(scanner interface {
	Scan(dest ...any) error
}) (EshikPrintSession, error) {
	item := EshikPrintSession{}
	err := scanner.Scan(
		&item.ID,
		&item.EshikModelID,
		&item.EshikModelPartID,
		&item.DoorCode,
		&item.ModelName,
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
	item.DoorCode = normalizeEshikDoorCode(item.DoorCode)
	return item, err
}

func (r *Repo) EshikPrintSessionCreate(model EshikModel, part EshikModelPart, serial string, counter, userID int) (int64, error) {
	if part.ID <= 0 {
		return 0, errors.New("eshik qismi noto'g'ri")
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
			eshik_model_id, eshik_model_part_id, door_code, model_name,
			serial, counter, user_id,
			factory_code, full_name_uz, seriya_raqami, index1, index2
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id`,
		model.ID,
		part.ID,
		normalizeEshikDoorCode(part.DoorCode),
		strings.TrimSpace(model.ModelName),
		serial,
		counter,
		userID,
		strings.TrimSpace(part.FactoryCode),
		strings.TrimSpace(part.FullNameUz),
		strings.TrimSpace(part.SeriyaRaqami),
		strings.TrimSpace(part.Index1),
		strings.TrimSpace(part.Index2),
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

func (r *Repo) EshikPrintSessionGetBySerial(serial string) (EshikPrintSession, error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return EshikPrintSession{}, errors.New("serial bo'sh")
	}
	item, err := scanEshikPrintSession(r.store.db.QueryRow(eshikPrintSessionSelect+`
		WHERE s.serial = $1
		ORDER BY s.id DESC
		LIMIT 1`, serial))
	if err != nil {
		if err == sql.ErrNoRows {
			return item, errors.New("ushbu serial Eshik liniyasida chiqmagan")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) EshikPrintSessionsGetLast(eshikModelID, limit int) ([]EshikPrintSession, error) {
	if limit <= 0 {
		limit = LastRecordsLimit
	}

	query := eshikPrintSessionSelect
	args := []any{}
	if eshikModelID > 0 {
		query += `
		WHERE COALESCE(s.eshik_model_id, p.eshik_model_id, 0) = $1
		ORDER BY s.c_time DESC, s.id DESC
		LIMIT $2`
		args = append(args, eshikModelID, limit)
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
