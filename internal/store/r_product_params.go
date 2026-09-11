package store

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/lib/pq"
)

type ProductParams struct {
	ID               int64
	SerialNumber     string
	CompressorSerial string
	AccSerial        string
	DoorSerial       string // legacy / display: freeze
	FreezeDoorSerial string
	RefDoorSerial    string
	GsCode           string
	ModelID          int
	Modeli           string
	CreatedAt        string
	UserName         string
}

func productParamsDuplicateError(err error) error {
	var pqErr *pq.Error
	if !errors.As(err, &pqErr) || pqErr.Code != "23505" {
		if strings.Contains(err.Error(), "duplicate key") || strings.Contains(err.Error(), "повторяющееся значение") {
			return errors.New("bunday ma'lumot allaqachon kiritilgan")
		}
		return err
	}

	detail := strings.ToLower(pqErr.Detail)
	switch {
	case strings.Contains(detail, "(compressor_serial)"):
		return errors.New("bu kompressor nomer allaqachon kiritilgan")
	case strings.Contains(detail, "(serial_number)"):
		return errors.New("bu serial allaqachon mavjud")
	case strings.Contains(detail, "(acc_serial)"):
		return errors.New("bu aksessuar nomer allaqachon kiritilgan")
	case strings.Contains(detail, "(door_serial)"):
		return errors.New("bu door nomer allaqachon kiritilgan")
	case strings.Contains(detail, "(freeze_door_serial)"):
		return errors.New("bu freeze door nomer allaqachon kiritilgan")
	case strings.Contains(detail, "(ref_door_serial)"):
		return errors.New("bu ref door nomer allaqachon kiritilgan")
	default:
		return errors.New("bunday ma'lumot allaqachon kiritilgan")
	}
}

func (r *Repo) ProductParamsCompressorExists(compressor string) (bool, error) {
	compressor = strings.TrimSpace(compressor)
	if compressor == "" {
		return false, nil
	}

	var id int64
	err := r.store.db.QueryRow(`
		SELECT id FROM lines.product_params
		WHERE compressor_serial = $1
		LIMIT 1`, compressor).Scan(&id)
	if err == nil {
		return true, nil
	}
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	return false, err
}

func (r *Repo) ProductParamsInsertT1(serial, compressor string, modelID, userID int) error {
	serial = strings.TrimSpace(serial)
	compressor = strings.TrimSpace(compressor)
	if serial == "" {
		return errors.New("serial bo'sh")
	}
	if compressor == "" {
		return errors.New("kompressor nomer bo'sh")
	}

	_, err := r.store.db.Exec(`
		INSERT INTO lines.product_params (serial_number, compressor_serial, model_id, c_user_id)
		VALUES ($1, $2, $3, $4)`,
		serial, compressor, modelID, userID,
	)
	if err != nil {
		if dupErr := productParamsDuplicateError(err); dupErr != nil {
			return dupErr
		}
		return err
	}
	return nil
}

func (r *Repo) ProductParamsInsertYigish(serial, gsCode string, modelID, userID int) error {
	serial = strings.TrimSpace(serial)
	gsCode = strings.TrimSpace(gsCode)
	if serial == "" {
		return errors.New("serial bo'sh")
	}

	_, err := r.store.db.Exec(`
		INSERT INTO lines.product_params (serial_number, model_id, gscode, c_user_id)
		VALUES ($1, $2, NULLIF($3, ''), $4)`,
		serial, modelID, gsCode, userID,
	)
	if err != nil {
		if dupErr := productParamsDuplicateError(err); dupErr != nil {
			return dupErr
		}
		return err
	}
	return nil
}

func (r *Repo) ProductParamsDeleteBySerial(serial string) error {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return nil
	}
	_, err := r.store.db.Exec(`DELETE FROM lines.product_params WHERE serial_number = $1`, serial)
	return err
}

// ProductParamsUpsertQadoqlash updates compressor/acc/door/model by serial if row exists; otherwise inserts.
func (r *Repo) ProductParamsUpsertQadoqlash(serial, compressor, accSerial, freezeDoorSerial, refDoorSerial string, modelID, userID int) error {
	serial = strings.TrimSpace(serial)
	compressor = strings.TrimSpace(compressor)
	accSerial = strings.TrimSpace(accSerial)
	freezeDoorSerial = strings.TrimSpace(freezeDoorSerial)
	refDoorSerial = strings.TrimSpace(refDoorSerial)
	if serial == "" {
		return errors.New("serial bo'sh")
	}
	if userID <= 0 {
		return errors.New("foydalanuvchi aniqlanmadi")
	}

	existing, err := r.ProductParamsGetBySerial(serial)
	if err != nil {
		return err
	}
	if existing.ID > 0 {
		_, err = r.store.db.Exec(`
			UPDATE lines.product_params
			SET compressor_serial = NULLIF($2, ''),
				acc_serial = NULLIF($3, ''),
				door_serial = NULLIF($4, ''),
				freeze_door_serial = NULLIF($4, ''),
				ref_door_serial = NULLIF($5, ''),
				model_id = NULLIF($6, 0),
				u_user_id = $7,
				u_time = now()
			WHERE serial_number = $1`,
			serial, compressor, accSerial, freezeDoorSerial, refDoorSerial, modelID, userID,
		)
		if err != nil {
			return productParamsDuplicateError(err)
		}
		return nil
	}

	_, err = r.store.db.Exec(`
		INSERT INTO lines.product_params (
			serial_number, compressor_serial, acc_serial, door_serial,
			freeze_door_serial, ref_door_serial, model_id, c_user_id
		) VALUES ($1, NULLIF($2, ''), NULLIF($3, ''), NULLIF($4, ''),
			NULLIF($4, ''), NULLIF($5, ''), NULLIF($6, 0), $7)`,
		serial, compressor, accSerial, freezeDoorSerial, refDoorSerial, modelID, userID,
	)
	if err != nil {
		return productParamsDuplicateError(err)
	}
	return nil
}

func scanProductParams(row interface {
	Scan(dest ...any) error
}) (ProductParams, error) {
	out := ProductParams{}
	err := row.Scan(
		&out.ID,
		&out.SerialNumber,
		&out.CompressorSerial,
		&out.AccSerial,
		&out.DoorSerial,
		&out.FreezeDoorSerial,
		&out.RefDoorSerial,
		&out.GsCode,
		&out.ModelID,
		&out.Modeli,
		&out.CreatedAt,
		&out.UserName,
	)
	if strings.TrimSpace(out.FreezeDoorSerial) == "" {
		out.FreezeDoorSerial = out.DoorSerial
	}
	if strings.TrimSpace(out.DoorSerial) == "" {
		out.DoorSerial = out.FreezeDoorSerial
	}
	return out, err
}

const productParamsSelect = `
	SELECT pp.id,
		COALESCE(pp.serial_number, ''),
		COALESCE(pp.compressor_serial, ''),
		COALESCE(pp.acc_serial, ''),
		COALESCE(pp.door_serial, ''),
		COALESCE(NULLIF(pp.freeze_door_serial, ''), pp.door_serial, ''),
		COALESCE(pp.ref_door_serial, ''),
		COALESCE(pp.gscode, ''),
		COALESCE(pp.model_id, 0),
		COALESCE(m.modeli, ''),
		COALESCE(to_char(pp.c_time, 'YYYY-MM-DD HH24:MI:SS'), ''),
		COALESCE(NULLIF(u.name, ''), u.login, '')
	FROM lines.product_params pp
	LEFT JOIN production.models m ON m.id = pp.model_id
	LEFT JOIN auth.users u ON u.id = pp.c_user_id`

func (r *Repo) ProductParamsGetBySerial(serial string) (ProductParams, error) {
	serial = strings.TrimSpace(serial)
	out := ProductParams{}
	if serial == "" {
		return out, errors.New("serial bo'sh")
	}

	out, err := scanProductParams(r.store.db.QueryRow(productParamsSelect+`
		WHERE pp.serial_number = $1`, serial))
	if errors.Is(err, sql.ErrNoRows) {
		return ProductParams{}, nil
	}
	return out, err
}

func (r *Repo) ProductParamsGetByAccSerial(accSerial string) (ProductParams, error) {
	accSerial = strings.TrimSpace(accSerial)
	out := ProductParams{}
	if accSerial == "" {
		return out, nil
	}
	out, err := scanProductParams(r.store.db.QueryRow(productParamsSelect+`
		WHERE pp.acc_serial = $1
		LIMIT 1`, accSerial))
	if errors.Is(err, sql.ErrNoRows) {
		return ProductParams{}, nil
	}
	return out, err
}

func (r *Repo) ProductParamsGetByDoorSerial(doorSerial string) (ProductParams, error) {
	doorSerial = strings.TrimSpace(doorSerial)
	out := ProductParams{}
	if doorSerial == "" {
		return out, nil
	}
	out, err := scanProductParams(r.store.db.QueryRow(productParamsSelect+`
		WHERE pp.door_serial = $1
		   OR pp.freeze_door_serial = $1
		   OR pp.ref_door_serial = $1
		LIMIT 1`, doorSerial))
	if errors.Is(err, sql.ErrNoRows) {
		return ProductParams{}, nil
	}
	return out, err
}

func (r *Repo) ProductParamsGetByCompressor(compressor string) (ProductParams, error) {
	compressor = strings.TrimSpace(compressor)
	out := ProductParams{}
	if compressor == "" {
		return out, errors.New("kompressor nomer bo'sh")
	}

	out, err := scanProductParams(r.store.db.QueryRow(productParamsSelect+`
		WHERE pp.compressor_serial = $1
		LIMIT 1`, compressor))
	if errors.Is(err, sql.ErrNoRows) {
		return ProductParams{}, nil
	}
	return out, err
}
