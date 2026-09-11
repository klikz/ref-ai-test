package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	EshikDoorFreeze = "freeze"
	EshikDoorRef    = "ref"
)

type EshikModelPart struct {
	ID               int    `json:"id"`
	EshikModelID     int    `json:"eshik_model_id"`
	DoorCode         string `json:"door_code"`
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
}

type EshikModel struct {
	ID        int            `json:"id"`
	ModelName string         `json:"model_name"`
	Freeze    EshikModelPart `json:"freeze"`
	Ref       EshikModelPart `json:"ref"`
	CTime     string         `json:"c_time"`
}

type EshikPartInput struct {
	ComponentID  int
	SeriyaRaqami string
	Index1       string
	Index2       string
}

func normalizeEshikDoorCode(doorCode string) string {
	switch strings.ToLower(strings.TrimSpace(doorCode)) {
	case EshikDoorFreeze, "freeze_door", "freeze_door_code":
		return EshikDoorFreeze
	case EshikDoorRef, "ref_door", "ref_door_code":
		return EshikDoorRef
	default:
		return ""
	}
}

func (p EshikModelPart) IsComplete() bool {
	return p.ID > 0 && p.ComponentID > 0 && strings.TrimSpace(p.SeriyaRaqami) != ""
}

func (m EshikModel) Parts() []EshikModelPart {
	parts := make([]EshikModelPart, 0, 2)
	if m.Freeze.ID > 0 || m.Freeze.ComponentID > 0 {
		parts = append(parts, m.Freeze)
	}
	if m.Ref.ID > 0 || m.Ref.ComponentID > 0 {
		parts = append(parts, m.Ref)
	}
	return parts
}

func (m EshikModel) PartByDoor(doorCode string) (EshikModelPart, bool) {
	doorCode = normalizeEshikDoorCode(doorCode)
	switch doorCode {
	case EshikDoorFreeze:
		if m.Freeze.ID > 0 || m.Freeze.ComponentID > 0 {
			return m.Freeze, true
		}
	case EshikDoorRef:
		if m.Ref.ID > 0 || m.Ref.ComponentID > 0 {
			return m.Ref, true
		}
	}
	return EshikModelPart{}, false
}

const eshikModelPartSelect = `
		SELECT p.id,
			p.eshik_model_id,
			p.door_code,
			p.component_id,
			COALESCE(c.factory_code, ''),
			COALESCE(c.full_name_uz, ''),
			c.manufacturer_code,
			c.standard_name_uz,
			c.odoo_code,
			COALESCE(c.comment, ''),
			ct.name,
			mu.name,
			c.photo_path,
			COALESCE(p.index1, ''),
			COALESCE(p.index2, ''),
			COALESCE(p.seriya_raqami, '')
		FROM production.eshik_model_parts p
		INNER JOIN production.components c ON c.id = p.component_id
		INNER JOIN production.component_types ct ON ct.id = c.type_id
		INNER JOIN production.measurement_units mu ON mu.id = c.unit_id
		WHERE c.status = true`

func scanEshikModelPart(scanner interface {
	Scan(dest ...any) error
}) (EshikModelPart, error) {
	item := EshikModelPart{}
	err := scanner.Scan(
		&item.ID,
		&item.EshikModelID,
		&item.DoorCode,
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
	)
	item.DoorCode = normalizeEshikDoorCode(item.DoorCode)
	return item, err
}

func attachEshikParts(models []EshikModel, parts []EshikModelPart) []EshikModel {
	byModel := map[int][]EshikModelPart{}
	for _, part := range parts {
		byModel[part.EshikModelID] = append(byModel[part.EshikModelID], part)
	}
	out := make([]EshikModel, 0, len(models))
	for _, model := range models {
		for _, part := range byModel[model.ID] {
			part.EshikModelID = model.ID
			switch part.DoorCode {
			case EshikDoorFreeze:
				model.Freeze = part
			case EshikDoorRef:
				model.Ref = part
			}
		}
		out = append(out, model)
	}
	return out
}

func (r *Repo) eshikModelPartsByModelIDs(modelIDs []int) ([]EshikModelPart, error) {
	if len(modelIDs) == 0 {
		return nil, nil
	}
	rows, err := r.store.db.Query(eshikModelPartSelect+`
		AND p.eshik_model_id = ANY($1)
		ORDER BY p.eshik_model_id, p.door_code`, intSliceParam(modelIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	parts := []EshikModelPart{}
	for rows.Next() {
		part, err := scanEshikModelPart(rows)
		if err != nil {
			return parts, err
		}
		parts = append(parts, part)
	}
	return parts, rows.Err()
}

func (r *Repo) EshikModelsGetAll() ([]EshikModel, error) {
	rows, err := r.store.db.Query(`
		SELECT m.id, m.model_name, to_char(m.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.eshik_models m
		ORDER BY m.model_name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	models := []EshikModel{}
	ids := []int{}
	for rows.Next() {
		item := EshikModel{}
		if err := rows.Scan(&item.ID, &item.ModelName, &item.CTime); err != nil {
			return models, err
		}
		models = append(models, item)
		ids = append(ids, item.ID)
	}
	if err := rows.Err(); err != nil {
		return models, err
	}
	parts, err := r.eshikModelPartsByModelIDs(ids)
	if err != nil {
		return models, err
	}
	return attachEshikParts(models, parts), nil
}

func (r *Repo) EshikModelGetByID(id int) (EshikModel, error) {
	if id <= 0 {
		return EshikModel{}, errors.New("model topilmadi")
	}
	item := EshikModel{}
	err := r.store.db.QueryRow(`
		SELECT m.id, m.model_name, to_char(m.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM production.eshik_models m
		WHERE m.id = $1`, id).Scan(&item.ID, &item.ModelName, &item.CTime)
	if err != nil {
		if err == sql.ErrNoRows {
			return EshikModel{}, errors.New("model topilmadi")
		}
		return EshikModel{}, err
	}
	parts, err := r.eshikModelPartsByModelIDs([]int{id})
	if err != nil {
		return EshikModel{}, err
	}
	attached := attachEshikParts([]EshikModel{item}, parts)
	if len(attached) == 0 {
		return EshikModel{}, errors.New("model topilmadi")
	}
	return attached[0], nil
}

func (r *Repo) EshikModelPartGetByID(id int) (EshikModelPart, error) {
	if id <= 0 {
		return EshikModelPart{}, errors.New("eshik qismi topilmadi")
	}
	row := r.store.db.QueryRow(eshikModelPartSelect+` AND p.id = $1`, id)
	item, err := scanEshikModelPart(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return EshikModelPart{}, errors.New("eshik qismi topilmadi")
		}
		return EshikModelPart{}, err
	}
	return item, nil
}

func (r *Repo) EshikModelIDByComponentID(componentID int) (int, error) {
	if componentID <= 0 {
		return 0, nil
	}
	var modelID int
	err := r.store.db.QueryRow(`
		SELECT eshik_model_id
		FROM production.eshik_model_parts
		WHERE component_id = $1`, componentID).Scan(&modelID)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	return modelID, nil
}

func (r *Repo) EshikComponentIDsByModelID(modelID int) (freezeID, refID int, err error) {
	if modelID <= 0 {
		return 0, 0, errors.New("model tanlanmagan")
	}
	rows, err := r.store.db.Query(`
		SELECT door_code, component_id
		FROM production.eshik_model_parts
		WHERE eshik_model_id = $1`, modelID)
	if err != nil {
		return 0, 0, err
	}
	defer rows.Close()
	for rows.Next() {
		var door string
		var componentID int
		if err := rows.Scan(&door, &componentID); err != nil {
			return 0, 0, err
		}
		switch normalizeEshikDoorCode(door) {
		case EshikDoorFreeze:
			freezeID = componentID
		case EshikDoorRef:
			refID = componentID
		}
	}
	if err := rows.Err(); err != nil {
		return 0, 0, err
	}
	if freezeID <= 0 || refID <= 0 {
		return 0, 0, errors.New("modelda freeze va ref komponentlari to'liq emas")
	}
	return freezeID, refID, nil
}

func (r *Repo) EshikModelIDsByComponentIDs(componentIDs []int) (map[int]int, error) {
	result := map[int]int{}
	if len(componentIDs) == 0 {
		return result, nil
	}
	rows, err := r.store.db.Query(`
		SELECT component_id, eshik_model_id
		FROM production.eshik_model_parts
		WHERE component_id = ANY($1)`, intSliceParam(componentIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var componentID, modelID int
		if err := rows.Scan(&componentID, &modelID); err != nil {
			return result, err
		}
		result[componentID] = modelID
	}
	return result, rows.Err()
}

func validateEshikPartInput(label string, in EshikPartInput) error {
	if in.ComponentID <= 0 {
		return fmt.Errorf("%s komponent tanlanmagan", label)
	}
	if strings.TrimSpace(in.SeriyaRaqami) == "" {
		return fmt.Errorf("%s serial prefix bo'sh", label)
	}
	return nil
}

func (r *Repo) ensureActiveComponent(componentID int) error {
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
	return nil
}

func (r *Repo) EshikModelAdd(modelName string, userID int, freeze, ref EshikPartInput) error {
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return errors.New("model nomi bo'sh")
	}
	if err := validateEshikPartInput("Freeze", freeze); err != nil {
		return err
	}
	if err := validateEshikPartInput("Ref", ref); err != nil {
		return err
	}
	if freeze.ComponentID == ref.ComponentID {
		return errors.New("freeze va ref uchun bir xil komponent tanlab bo'lmaydi")
	}
	if err := r.ensureActiveComponent(freeze.ComponentID); err != nil {
		return err
	}
	if err := r.ensureActiveComponent(ref.ComponentID); err != nil {
		return err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var modelID int64
	err = tx.QueryRow(`
		INSERT INTO production.eshik_models (model_name, c_user_id)
		VALUES ($1, $2)
		RETURNING id`,
		modelName, nullInt(userID),
	).Scan(&modelID)
	if err != nil {
		if strings.Contains(err.Error(), "eshik_models_name_un") || strings.Contains(err.Error(), "duplicate key") {
			return errors.New("bunday model nomi allaqachon bor")
		}
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO production.eshik_model_parts (
			eshik_model_id, door_code, component_id, seriya_raqami, index1, index2
		) VALUES
			($1, $2, $3, $4, $5, $6),
			($1, $7, $8, $9, $10, $11)`,
		modelID,
		EshikDoorFreeze, freeze.ComponentID, strings.TrimSpace(freeze.SeriyaRaqami), strings.TrimSpace(freeze.Index1), strings.TrimSpace(freeze.Index2),
		EshikDoorRef, ref.ComponentID, strings.TrimSpace(ref.SeriyaRaqami), strings.TrimSpace(ref.Index1), strings.TrimSpace(ref.Index2),
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repo) EshikModelUpdate(id int, modelName string, freeze, ref EshikPartInput) error {
	if id <= 0 {
		return errors.New("model topilmadi")
	}
	modelName = strings.TrimSpace(modelName)
	if modelName == "" {
		return errors.New("model nomi bo'sh")
	}
	if err := validateEshikPartInput("Freeze", freeze); err != nil {
		return err
	}
	if err := validateEshikPartInput("Ref", ref); err != nil {
		return err
	}
	if freeze.ComponentID == ref.ComponentID {
		return errors.New("freeze va ref uchun bir xil komponent tanlab bo'lmaydi")
	}
	if err := r.ensureActiveComponent(freeze.ComponentID); err != nil {
		return err
	}
	if err := r.ensureActiveComponent(ref.ComponentID); err != nil {
		return err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE production.eshik_models
		SET model_name = $2
		WHERE id = $1`, id, modelName)
	if err != nil {
		if strings.Contains(err.Error(), "eshik_models_name_un") || strings.Contains(err.Error(), "duplicate key") {
			return errors.New("bunday model nomi allaqachon bor")
		}
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("model topilmadi")
	}

	upsertPart := func(door string, in EshikPartInput) error {
		_, err := tx.Exec(`
			INSERT INTO production.eshik_model_parts (
				eshik_model_id, door_code, component_id, seriya_raqami, index1, index2
			) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (eshik_model_id, door_code) DO UPDATE SET
				component_id = EXCLUDED.component_id,
				seriya_raqami = EXCLUDED.seriya_raqami,
				index1 = EXCLUDED.index1,
				index2 = EXCLUDED.index2`,
			id, door, in.ComponentID,
			strings.TrimSpace(in.SeriyaRaqami),
			strings.TrimSpace(in.Index1),
			strings.TrimSpace(in.Index2),
		)
		return err
	}
	if err := upsertPart(EshikDoorFreeze, freeze); err != nil {
		return err
	}
	if err := upsertPart(EshikDoorRef, ref); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repo) EshikModelDelete(id int) error {
	res, err := r.store.db.Exec(`DELETE FROM production.eshik_models WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if affected == 0 {
		return errors.New("model topilmadi")
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
	eshikModelPartID int,
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

	if eshikModelPartID > 0 {
		_ = r.EshikDailyCounterUndo(eshikModelPartID)
	}
	return nil
}
