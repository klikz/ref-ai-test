package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const RadiatorComponentTypeID = 3

type RadiatorComponent struct {
	ID                      int    `json:"id"`
	ComponentID             int    `json:"component_id"`
	FactoryCode             string `json:"factory_code"`
	FullNameUz              string `json:"full_name_uz"`
	ManufacturerCode        string `json:"manufacturer_code"`
	StandardNameUz          string `json:"standard_name_uz"`
	OdooCode                string `json:"odoo_code"`
	Type                    string `json:"type"`
	Unit                    string `json:"unit"`
	PhotoPath               string `json:"photo_path"`
	FinPressComponentID     int    `json:"fin_press_component_id"`
	FinPressComponentRefID  int    `json:"fin_press_component_ref_id"`
	FinPressFactoryCode     string `json:"fin_press_factory_code"`
	FinPressFullNameUz      string `json:"fin_press_full_name_uz"`
	FinPressManufacturerCode string `json:"fin_press_manufacturer_code"`
	FinPressPhotoPath       string `json:"fin_press_photo_path"`
	FinPressIndex1          string `json:"fin_press_index1"`
	FinPressIndex2          string `json:"fin_press_index2"`
	Index1                  string `json:"index1"`
	Index2                  string `json:"index2"`
	SeriyaRaqami            string `json:"seriya_raqami"`
	CTime                   string `json:"c_time"`
}

const radiatorComponentSelect = `
		SELECT rc.id,
			rc.component_id,
			COALESCE(cr.factory_code, ''),
			COALESCE(cr.full_name_uz, ''),
			cr.manufacturer_code,
			cr.standard_name_uz,
			cr.odoo_code,
			ctr.name,
			mur.name,
			cr.photo_path,
			rc.fin_press_component_id,
			cf.id,
			COALESCE(cf.factory_code, ''),
			COALESCE(cf.full_name_uz, ''),
			cf.manufacturer_code,
			cf.photo_path,
			COALESCE(fpc.index1, ''),
			COALESCE(fpc.index2, ''),
			COALESCE(rc.index1, ''),
			COALESCE(rc.index2, ''),
			COALESCE(rc.seriya_raqami, ''),
			to_char(rc.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.radiator_components rc
		INNER JOIN production.components cr ON cr.id = rc.component_id AND cr.type_id = $1
		INNER JOIN production.fin_press_components fpc ON fpc.id = rc.fin_press_component_id
		INNER JOIN production.components cf ON cf.id = fpc.component_id
		INNER JOIN production.component_types ctr ON ctr.id = cr.type_id
		INNER JOIN production.measurement_units mur ON mur.id = cr.unit_id
		WHERE cr.status = true AND cf.status = true`

func scanRadiatorComponent(scanner interface {
	Scan(dest ...any) error
}) (RadiatorComponent, error) {
	item := RadiatorComponent{}
	err := scanner.Scan(
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
		&item.FinPressComponentID,
		&item.FinPressComponentRefID,
		&item.FinPressFactoryCode,
		&item.FinPressFullNameUz,
		&item.FinPressManufacturerCode,
		&item.FinPressPhotoPath,
		&item.FinPressIndex1,
		&item.FinPressIndex2,
		&item.Index1,
		&item.Index2,
		&item.SeriyaRaqami,
		&item.CTime,
	)
	return item, err
}

func (r *Repo) RadiatorComponentsGetAll() ([]RadiatorComponent, error) {
	rows, err := r.store.db.Query(radiatorComponentSelect+`
		ORDER BY NULLIF(cr.factory_code, ''), cr.manufacturer_code`,
		RadiatorComponentTypeID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	allData := []RadiatorComponent{}
	for rows.Next() {
		item, err := scanRadiatorComponent(rows)
		if err != nil {
			return allData, err
		}
		allData = append(allData, item)
	}
	if err = rows.Err(); err != nil {
		return allData, err
	}
	return allData, nil
}

func (r *Repo) RadiatorComponentGetByID(id int) (RadiatorComponent, error) {
	if id <= 0 {
		return RadiatorComponent{}, errors.New("komponent topilmadi")
	}
	row := r.store.db.QueryRow(radiatorComponentSelect+`
		AND rc.id = $2`,
		RadiatorComponentTypeID, id,
	)
	item, err := scanRadiatorComponent(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return RadiatorComponent{}, errors.New("komponent topilmadi")
		}
		return RadiatorComponent{}, err
	}
	return item, nil
}

func (r *Repo) RadiatorComponentAdd(componentID, finPressComponentID, userID int, seriyaRaqami, index1, index2 string) error {
	if componentID <= 0 {
		return errors.New("radiator komponenti tanlanmagan")
	}
	if finPressComponentID <= 0 {
		return errors.New("fin press komponenti tanlanmagan")
	}

	var exists int
	err := r.store.db.QueryRow(`
		SELECT 1
		FROM production.components
		WHERE id = $1 AND type_id = $2 AND status = true`,
		componentID, RadiatorComponentTypeID,
	).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("radiator komponenti topilmadi yoki type_id noto'g'ri")
		}
		return err
	}

	err = r.store.db.QueryRow(`
		SELECT 1
		FROM production.fin_press_components fpc
		INNER JOIN production.components c ON c.id = fpc.component_id
		WHERE fpc.id = $1 AND c.status = true`,
		finPressComponentID,
	).Scan(&exists)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("fin press komponenti topilmadi")
		}
		return err
	}

	_, err = r.store.db.Exec(`
		INSERT INTO production.radiator_components (component_id, fin_press_component_id, c_user_id, seriya_raqami, index1, index2)
		VALUES ($1, $2, $3, $4, $5, $6)`,
		componentID, finPressComponentID, userID, seriyaRaqami, index1, index2,
	)
	if err != nil {
		if strings.Contains(err.Error(), "radiator_components_component_un") {
			return errors.New("radiator komponenti allaqachon ro'yxatda")
		}
		return err
	}
	return nil
}

func (r *Repo) RadiatorComponentUpdateFields(id int, seriyaRaqami, index1, index2 string) error {
	res, err := r.store.db.Exec(`
		UPDATE production.radiator_components
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

func (r *Repo) RadiatorComponentDelete(id int) error {
	res, err := r.store.db.Exec(`
		DELETE FROM production.radiator_components
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

func (r *Repo) RadiatorChiqishBalancePrecheck(radiatorLineID, finPressComponentID int) error {
	if radiatorLineID <= 0 {
		return errors.New("liniya noto'g'ri")
	}
	if finPressComponentID <= 0 {
		return errors.New("komponent noto'g'ri")
	}

	current, err := r.LinesBalanceQuantity(radiatorLineID, finPressComponentID)
	if err != nil {
		return err
	}
	if current < 1 {
		return fmt.Errorf("balans yetarli emas (joriy: %.4f)", current)
	}
	return nil
}

func (r *Repo) RadiatorChiqishBalanceApply(
	radiatorLineID int,
	radiatorComponentID, finPressComponentID int,
	userID int,
	sessionID int64,
	serial string,
) (int, error) {
	if radiatorLineID <= 0 {
		return 0, errors.New("liniya noto'g'ri")
	}
	if radiatorComponentID <= 0 || finPressComponentID <= 0 {
		return 0, errors.New("komponent noto'g'ri")
	}
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return 0, errors.New("serial bo'sh")
	}
	if err := r.auxiliaryProductDuplicateCheck(radiatorLineID, serial); err != nil {
		return 0, err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	comment := fmt.Sprintf("Radiator chiqish (sessiya #%d, %s)", sessionID, serial)

	// Fin press komponenti radiator liniyasida qabul qilingan (Radiator Kirish)
	_, err = r.linesBalanceApplyChangeTx(tx, BalanceChangeParams{
		LineID:         radiatorLineID,
		ComponentID:    finPressComponentID,
		QuantityChange: -1,
		UserID:         userID,
		Source:         "radiator_print",
		Comment:        comment,
	})
	if err != nil {
		return 0, err
	}

	_, err = r.linesBalanceApplyChangeTx(tx, BalanceChangeParams{
		LineID:         radiatorLineID,
		ComponentID:    radiatorComponentID,
		QuantityChange: 1,
		UserID:         userID,
		Source:         "radiator_print",
		Comment:        comment,
	})
	if err != nil {
		return 0, err
	}

	auxiliaryProductID, err := r.auxiliaryProductInsertTx(tx, radiatorLineID, radiatorComponentID, userID, serial)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		UPDATE production.radiator_print_sessions
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
	return int(auxiliaryProductID), nil
}
