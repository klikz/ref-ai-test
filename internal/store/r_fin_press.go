package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
)

type FinPressComponent struct {
	ID               int    `json:"id"`
	ComponentID      int    `json:"component_id"`
	FactoryCode      string `json:"factory_code"`
	FullNameUz       string `json:"full_name_uz"`
	ManufacturerCode string `json:"manufacturer_code"`
	StandardNameUz   string `json:"standard_name_uz"`
	OdooCode         string `json:"odoo_code"`
	Type             string `json:"type"`
	Unit             string `json:"unit"`
	PhotoPath        string `json:"photo_path"`
	Index1           string `json:"index1"`
	Index2           string `json:"index2"`
	CTime            string `json:"c_time"`
}

func (r *Repo) FinPressComponentsGetAll() ([]FinPressComponent, error) {
	rows, err := r.store.db.Query(`
		SELECT fpc.id, c.id, COALESCE(c.factory_code, ''), COALESCE(c.full_name_uz, ''),
			c.manufacturer_code, c.standard_name_uz, c.odoo_code,
			ct.name, mu.name, c.photo_path, COALESCE(fpc.index1, ''), COALESCE(fpc.index2, ''),
			to_char(fpc.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.fin_press_components fpc
		INNER JOIN production.components c ON c.id = fpc.component_id
		INNER JOIN production.component_types ct ON ct.id = c.type_id
		INNER JOIN production.measurement_units mu ON mu.id = c.unit_id
		WHERE c.status = true
		ORDER BY NULLIF(c.factory_code, ''), c.manufacturer_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []FinPressComponent{}
	for rows.Next() {
		item := FinPressComponent{}
		if err := rows.Scan(
			&item.ID,
			&item.ComponentID,
			&item.FactoryCode,
			&item.FullNameUz,
			&item.ManufacturerCode,
			&item.StandardNameUz,
			&item.OdooCode,
			&item.Type,
			&item.Unit,
			&item.PhotoPath,
			&item.Index1,
			&item.Index2,
			&item.CTime,
		); err != nil {
			return allData, err
		}
		allData = append(allData, item)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) FinPressComponentAdd(componentID, userID int, index1, index2 string) error {
	_, err := r.store.db.Exec(`
		INSERT INTO production.fin_press_components (component_id, c_user_id, index1, index2)
		VALUES ($1, $2, $3, $4)`,
		componentID, userID, index1, index2,
	)
	if err != nil {
		if strings.Contains(err.Error(), "fin_press_components_component_un") ||
			strings.Contains(err.Error(), "duplicate key") ||
			strings.Contains(err.Error(), "повторяющееся значение") {
			return errors.New("komponent allaqachon fin press ro'yxatida")
		}
		return err
	}
	return nil
}

func (r *Repo) FinPressComponentGetByComponentID(componentID int) (FinPressComponent, error) {
	item := FinPressComponent{}
	err := r.store.db.QueryRow(`
		SELECT fpc.id, c.id, COALESCE(c.factory_code, ''), COALESCE(c.full_name_uz, ''),
			c.manufacturer_code, c.standard_name_uz, COALESCE(c.odoo_code, ''),
			ct.name, mu.name, COALESCE(c.photo_path, ''), COALESCE(fpc.index1, ''), COALESCE(fpc.index2, ''),
			to_char(fpc.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.fin_press_components fpc
		INNER JOIN production.components c ON c.id = fpc.component_id
		INNER JOIN production.component_types ct ON ct.id = c.type_id
		INNER JOIN production.measurement_units mu ON mu.id = c.unit_id
		WHERE fpc.component_id = $1 AND c.status = true`,
		componentID,
	).Scan(
		&item.ID, &item.ComponentID, &item.FactoryCode, &item.FullNameUz,
		&item.ManufacturerCode, &item.StandardNameUz,
		&item.OdooCode, &item.Type, &item.Unit, &item.PhotoPath, &item.Index1, &item.Index2, &item.CTime,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return item, errors.New("komponent fin press ro'yxatida topilmadi")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) FinPressComponentUpdateIndexes(id int, index1, index2 string) error {
	res, err := r.store.db.Exec(`
		UPDATE production.fin_press_components
		SET index1 = $2, index2 = $3
		WHERE id = $1`,
		id, index1, index2,
	)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("komponent topilmadi")
	}
	return nil
}

func (r *Repo) FinPressComponentDelete(id int) error {
	res, err := r.store.db.Exec(`
		DELETE FROM production.fin_press_components
		WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("komponent topilmadi")
	}
	return nil
}

type FinPressPrintSession struct {
	ID               int64  `json:"session_id"`
	ComponentID      int    `json:"component_id"`
	ManufacturerCode string `json:"manufacturer_code"`
	Count            int    `json:"count"`
	UserID           int    `json:"user_id"`
	UserName         string `json:"user_name"`
	Status           string `json:"status"`
	CTime            string `json:"c_time"`
}

func (r *Repo) FinPressPrintSessionCreate(componentID, userID, count int) (int64, error) {
	if componentID <= 0 || count <= 0 {
		return 0, errors.New("komponent yoki miqdor noto'g'ri")
	}
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}

	var finPressID int
	err := r.store.db.QueryRow(`
		SELECT fpc.id
		FROM production.fin_press_components fpc
		INNER JOIN production.components c ON c.id = fpc.component_id
		WHERE fpc.component_id = $1 AND c.status = true`,
		componentID,
	).Scan(&finPressID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("komponent fin press ro'yxatida topilmadi")
		}
		return 0, err
	}

	var sessionID int64
	err = r.store.db.QueryRow(`
		INSERT INTO production.fin_press_print_sessions (component_id, count, user_id, status)
		VALUES ($1, $2, $3, 'pending')
		RETURNING id`,
		componentID, count, userID,
	).Scan(&sessionID)
	if err != nil {
		return 0, err
	}
	return sessionID, nil
}

func (r *Repo) FinPressPrintSessionGetByID(sessionID int64) (FinPressPrintSession, error) {
	item := FinPressPrintSession{}
	err := r.store.db.QueryRow(`
		SELECT s.id, s.component_id, s.count, COALESCE(s.user_id, 0),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			COALESCE(s.status, 'pending'),
			to_char(s.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.fin_press_print_sessions s
		LEFT JOIN auth.users u ON u.id = s.user_id
		WHERE s.id = $1`,
		sessionID,
	).Scan(&item.ID, &item.ComponentID, &item.Count, &item.UserID, &item.UserName, &item.Status, &item.CTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return item, errors.New("sessiya topilmadi")
		}
		return item, err
	}
	return item, nil
}

func (r *Repo) FinPressPrintSessionsGetLast(componentID, limit int) ([]FinPressPrintSession, error) {
	if limit <= 0 {
		limit = LastRecordsLimit
	}

	query := `
		SELECT s.id, s.component_id, COALESCE(c.manufacturer_code, ''), s.count, COALESCE(s.user_id, 0),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			COALESCE(s.status, 'pending'),
			to_char(s.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.fin_press_print_sessions s
		LEFT JOIN auth.users u ON u.id = s.user_id
		LEFT JOIN production.components c ON c.id = s.component_id`

	args := []any{}
	if componentID > 0 {
		query += `
		WHERE s.component_id = $1
		ORDER BY s.c_time DESC, s.id DESC
		LIMIT $2`
		args = append(args, componentID, limit)
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

	items := []FinPressPrintSession{}
	for rows.Next() {
		item := FinPressPrintSession{}
		if err := rows.Scan(
			&item.ID, &item.ComponentID, &item.ManufacturerCode, &item.Count, &item.UserID, &item.UserName, &item.Status, &item.CTime,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

type FinPressReceiveResult struct {
	SessionID        int64   `json:"session_id"`
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code"`
	StandardNameUz   string  `json:"standard_name_uz"`
	PhotoPath        string  `json:"photo_path"`
	Count            int     `json:"count"`
	QuantityAfter    float64 `json:"quantity_after"`
}

func ParseFinPressSessionScan(raw string) (int64, error) {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0, errors.New("sessiya kodi bo'sh")
	}

	if strings.HasPrefix(strings.ToUpper(value), "FP:") {
		value = strings.TrimSpace(value[3:])
	}

	sessionID, err := strconv.ParseInt(value, 10, 64)
	if err != nil || sessionID <= 0 {
		return 0, errors.New("sessiya kodi noto'g'ri")
	}
	return sessionID, nil
}

func (r *Repo) FinPressSessionReceive(sessionID int64, radiatorLineID, userID int) (FinPressReceiveResult, error) {
	result := FinPressReceiveResult{}
	if sessionID <= 0 {
		return result, errors.New("sessiya raqami noto'g'ri")
	}
	if radiatorLineID <= 0 {
		return result, errors.New("radiator liniyasi topilmadi")
	}
	if userID <= 0 {
		return result, errors.New("foydalanuvchi aniqlanmadi")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	var componentID int
	var count int
	var status string
	err = tx.QueryRow(`
		SELECT s.component_id, s.count, COALESCE(s.status, 'pending')
		FROM production.fin_press_print_sessions s
		WHERE s.id = $1
		FOR UPDATE`,
		sessionID,
	).Scan(&componentID, &count, &status)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, errors.New("sessiya topilmadi")
		}
		return result, err
	}
	if status != "pending" {
		return result, errors.New("sessiya allaqachon qabul qilingan")
	}
	if count <= 0 {
		return result, errors.New("sessiya miqdori noto'g'ri")
	}

	err = tx.QueryRow(`
		SELECT c.manufacturer_code, c.standard_name_uz, COALESCE(c.photo_path, '')
		FROM production.components c
		WHERE c.id = $1 AND c.status = true`,
		componentID,
	).Scan(&result.ManufacturerCode, &result.StandardNameUz, &result.PhotoPath)
	if err != nil {
		if err == sql.ErrNoRows {
			return result, errors.New("komponent topilmadi")
		}
		return result, err
	}

	finPressLineID, err := r.LinesFinPressLineID()
	if err != nil {
		return result, err
	}
	if finPressLineID == radiatorLineID {
		return result, errors.New("fin press va radiator liniyalari bir xil")
	}

	source := "fin_press_receive"
	commentIn := fmt.Sprintf("Fin press sessiya #%d qabul", sessionID)
	commentOut := fmt.Sprintf("Radiator qabul (sessiya #%d)", sessionID)
	qty := float64(count)

	_, err = r.linesBalanceApplyChangeTx(tx, BalanceChangeParams{
		LineID:         finPressLineID,
		ComponentID:    componentID,
		QuantityChange: -qty,
		UserID:         userID,
		Source:         source,
		Comment:        commentOut,
	})
	if err != nil {
		return result, err
	}

	quantityAfter, err := r.linesBalanceApplyChangeTx(tx, BalanceChangeParams{
		LineID:         radiatorLineID,
		ComponentID:    componentID,
		QuantityChange: qty,
		UserID:         userID,
		Source:         source,
		Comment:        commentIn,
	})
	if err != nil {
		return result, err
	}

	_, err = tx.Exec(`
		UPDATE production.fin_press_print_sessions
		SET status = 'received',
			received_at = now(),
			received_by_user_id = $2
		WHERE id = $1`,
		sessionID, userID,
	)
	if err != nil {
		return result, err
	}

	if err = tx.Commit(); err != nil {
		return result, err
	}

	result.SessionID = sessionID
	result.ComponentID = componentID
	result.Count = count
	result.QuantityAfter = quantityAfter
	return result, nil
}
