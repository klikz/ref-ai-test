package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

type EshikComponent struct {
	ID               int    `json:"id"`
	ComponentID      int    `json:"component_id"`
	FactoryCode      string `json:"factory_code"`
	FullNameUz       string `json:"full_name_uz"`
	ManufacturerCode string `json:"manufacturer_code"`
	StandardNameUz   string `json:"standard_name_uz"`
	OdooCode         string `json:"odoo_code"`
	Comment          string `json:"comment"`
	Type             string `json:"type"`
	Unit             string `json:"unit"`
	PhotoPath        string `json:"photo_path"`
	Index1           string `json:"index1"`
	Index2           string `json:"index2"`
	SeriyaRaqami     string `json:"seriya_raqami"`
	CTime            string `json:"c_time"`
}

const eshikComponentSelect = `
		SELECT ec.id,
			ec.component_id,
			COALESCE(c.factory_code, ''),
			COALESCE(c.full_name_uz, ''),
			c.manufacturer_code,
			c.standard_name_uz,
			c.odoo_code,
			COALESCE(c.comment, ''),
			ct.name,
			mu.name,
			c.photo_path,
			COALESCE(ec.index1, ''),
			COALESCE(ec.index2, ''),
			COALESCE(ec.seriya_raqami, ''),
			to_char(ec.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.eshik_components ec
		INNER JOIN production.components c ON c.id = ec.component_id
		INNER JOIN production.component_types ct ON ct.id = c.type_id
		INNER JOIN production.measurement_units mu ON mu.id = c.unit_id
		WHERE c.status = true`

func scanEshikComponent(scanner interface {
	Scan(dest ...any) error
}) (EshikComponent, error) {
	item := EshikComponent{}
	err := scanner.Scan(
		&item.ID,
		&item.ComponentID,
		&item.FactoryCode,
		&item.FullNameUz,
		&item.ManufacturerCode,
		&item.StandardNameUz,
		&item.OdooCode,
		&item.Comment,
		&item.Type,
		&item.Unit,
		&item.PhotoPath,
		&item.Index1,
		&item.Index2,
		&item.SeriyaRaqami,
		&item.CTime,
	)
	return item, err
}

func (r *Repo) EshikComponentsGetAll() ([]EshikComponent, error) {
	rows, err := r.store.db.Query(eshikComponentSelect + `
		ORDER BY NULLIF(c.factory_code, ''), c.manufacturer_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []EshikComponent{}
	for rows.Next() {
		item, err := scanEshikComponent(rows)
		if err != nil {
			return allData, err
		}
		allData = append(allData, item)
	}
	return allData, rows.Err()
}

func (r *Repo) EshikComponentGetByID(id int) (EshikComponent, error) {
	if id <= 0 {
		return EshikComponent{}, errors.New("komponent topilmadi")
	}
	row := r.store.db.QueryRow(eshikComponentSelect+`
		AND ec.id = $1`, id)
	item, err := scanEshikComponent(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return EshikComponent{}, errors.New("komponent topilmadi")
		}
		return EshikComponent{}, err
	}
	return item, nil
}

func (r *Repo) EshikComponentAdd(componentID, userID int, seriyaRaqami, index1, index2 string) error {
	if componentID <= 0 {
		return errors.New("komponent tanlanmagan")
	}

	var exists int
	err := r.store.db.QueryRow(`
		SELECT 1 FROM production.components WHERE id = $1 AND status = true`,
		componentID,
	).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("komponent topilmadi")
		}
		return err
	}

	_, err = r.store.db.Exec(`
		INSERT INTO production.eshik_components (component_id, c_user_id, seriya_raqami, index1, index2)
		VALUES ($1, $2, $3, $4, $5)`,
		componentID, userID, seriyaRaqami, index1, index2,
	)
	if err != nil {
		if strings.Contains(err.Error(), "eshik_components_component_un") ||
			strings.Contains(err.Error(), "duplicate key") {
			return errors.New("komponent allaqachon eshik ro'yxatida")
		}
		return err
	}
	return nil
}

func (r *Repo) EshikComponentUpdateFields(id int, seriyaRaqami, index1, index2 string) error {
	res, err := r.store.db.Exec(`
		UPDATE production.eshik_components
		SET seriya_raqami = $2, index1 = $3, index2 = $4
		WHERE id = $1`,
		id, seriyaRaqami, index1, index2,
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

func (r *Repo) EshikComponentDelete(id int) error {
	res, err := r.store.db.Exec(`
		DELETE FROM production.eshik_components
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

func (r *Repo) EshikChiqishBalanceApply(
	eshikLineID int,
	componentID int,
	userID int,
	sessionID int64,
	serial string,
) (int64, error) {
	if eshikLineID <= 0 {
		return 0, errors.New("liniya noto'g'ri")
	}
	if componentID <= 0 {
		return 0, errors.New("komponent noto'g'ri")
	}
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return 0, errors.New("serial bo'sh")
	}
	if err := r.auxiliaryProductDuplicateCheck(eshikLineID, serial); err != nil {
		return 0, err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	comment := fmt.Sprintf("Eshik liniyasi (sessiya #%d, %s)", sessionID, serial)

	_, err = r.linesBalanceApplyChangeTx(tx, BalanceChangeParams{
		LineID:         eshikLineID,
		ComponentID:    componentID,
		QuantityChange: 1,
		UserID:         userID,
		Source:         "eshik_print",
		Comment:        comment,
	})
	if err != nil {
		return 0, err
	}

	auxiliaryProductID, err := r.auxiliaryProductInsertTx(tx, eshikLineID, componentID, userID, serial)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		UPDATE production.eshik_print_sessions
		SET auxiliary_product_id = $2
		WHERE id = $1`,
		sessionID, auxiliaryProductID,
	)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return auxiliaryProductID, nil
}

func (r *Repo) EshikPrintRollback(
	eshikLineID int,
	sessionID int64,
	eshikComponentID int,
	componentID int,
	auxiliaryProductID int64,
	userID int,
) error {
	if sessionID <= 0 {
		return nil
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if auxiliaryProductID > 0 {
		_, err = tx.Exec(`DELETE FROM lines.auxiliary_products WHERE id = $1`, auxiliaryProductID)
		if err != nil {
			return err
		}
	}

	if componentID > 0 && eshikLineID > 0 && userID > 0 {
		_, err = r.linesBalanceApplyChangeTx(tx, BalanceChangeParams{
			LineID:         eshikLineID,
			ComponentID:    componentID,
			QuantityChange: -1,
			UserID:         userID,
			Source:         "eshik_print_rollback",
			Comment:        fmt.Sprintf("Eshik print rollback (sessiya #%d)", sessionID),
		})
		if err != nil {
			return err
		}
	}

	_, err = tx.Exec(`DELETE FROM production.eshik_print_sessions WHERE id = $1`, sessionID)
	if err != nil {
		return err
	}

	if err = tx.Commit(); err != nil {
		return err
	}

	if eshikComponentID > 0 {
		_ = r.EshikDailyCounterUndo(eshikComponentID)
	}
	return nil
}
