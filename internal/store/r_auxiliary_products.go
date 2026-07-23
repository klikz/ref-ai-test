package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
)

const (
	AuxiliaryLineFinPressID = FinPressLineID
	AuxiliaryLineRadiatorID = RadiatorLineID
)

type AuxiliaryProduct struct {
	ID               int    `json:"id"`
	LineID           int    `json:"line_id"`
	ComponentID      int    `json:"component_id"`
	FactoryCode      string `json:"factory_code"`
	FullNameUz       string `json:"full_name_uz"`
	ManufacturerCode string `json:"manufacturer_code"`
	Serial           string `json:"serial"`
	UserID           int    `json:"user_id"`
	UserName         string `json:"user_name"`
	CTime            string `json:"c_time"`
}

func (r *Repo) auxiliaryProductDuplicateCheck(lineID int, serial string) error {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return errors.New("serial bo'sh")
	}

	var existingID int64
	err := r.store.db.QueryRow(`
		SELECT id
		FROM lines.auxiliary_products
		WHERE line_id = $1 AND serial = $2
		LIMIT 1`,
		lineID, serial,
	).Scan(&existingID)
	if err == nil {
		return errors.New("bu serial nomer ushbu liniyada allaqachon kiritilgan")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return err
	}
	return nil
}

func auxiliaryProductDuplicateError(err error) error {
	var pqErr *pq.Error
	if errors.As(err, &pqErr) && pqErr.Code == "23505" {
		if pqErr.Constraint == "auxiliary_products_line_serial_un" {
			return errors.New("bu serial nomer ushbu liniyada allaqachon kiritilgan")
		}
	}
	if strings.Contains(err.Error(), "auxiliary_products_line_serial_un") ||
		strings.Contains(err.Error(), "duplicate key") ||
		strings.Contains(err.Error(), "повторяющееся значение") {
		return errors.New("bu serial nomer ushbu liniyada allaqachon kiritilgan")
	}
	return nil
}

func (r *Repo) auxiliaryProductInsertTx(tx *sql.Tx, lineID, componentID, userID int, serial string) (int64, error) {
	serial = strings.TrimSpace(serial)
	if lineID <= 0 || componentID <= 0 {
		return 0, errors.New("liniya yoki komponent noto'g'ri")
	}
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}
	if serial == "" {
		return 0, errors.New("serial bo'sh")
	}

	var id int64
	err := tx.QueryRow(`
		INSERT INTO lines.auxiliary_products (line_id, component_id, serial, user_id)
		VALUES ($1, $2, $3, $4)
		RETURNING id`,
		lineID, componentID, serial, userID,
	).Scan(&id)
	if err != nil {
		if dupErr := auxiliaryProductDuplicateError(err); dupErr != nil {
			return 0, dupErr
		}
		return 0, err
	}
	return id, nil
}
