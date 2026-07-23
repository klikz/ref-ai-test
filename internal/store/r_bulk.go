package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/klikz/api_v3/internal/models"
)

const bulkBatchSize = 500

func (r *Repo) GsCodesBulkAdd(codes []string, modelID, userID int) (int, error) {
	if len(codes) == 0 {
		return 0, nil
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	inserted := 0
	for start := 0; start < len(codes); start += bulkBatchSize {
		end := start + bulkBatchSize
		if end > len(codes) {
			end = len(codes)
		}
		batch := codes[start:end]
		batchInserted, err := gsCodesInsertBatch(tx, batch, modelID, userID)
		if err != nil {
			return inserted, err
		}
		inserted += batchInserted
	}

	if err := tx.Commit(); err != nil {
		return inserted, err
	}
	return inserted, nil
}

func gsCodesInsertBatch(tx *sql.Tx, codes []string, modelID, userID int) (int, error) {
	if len(codes) == 0 {
		return 0, nil
	}

	var sb strings.Builder
	args := make([]any, 0, len(codes)*3)
	sb.WriteString(`INSERT INTO production.gscodes (model_id, "data", c_user_id, status) VALUES `)
	for i, code := range codes {
		if i > 0 {
			sb.WriteString(",")
		}
		n := i*3 + 1
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,true)", n, n+1, n+2))
		args = append(args, modelID, code, userID)
	}
	sb.WriteString(" ON CONFLICT DO NOTHING")
	result, err := tx.Exec(sb.String(), args...)
	if err != nil {
		return 0, err
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rows), nil
}

func (r *Repo) TechComponentsBulkInsert(items []models.TechComponent, userID int) ([]string, error) {
	if len(items) == 0 {
		return nil, nil
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	var failed []string
	for start := 0; start < len(items); start += bulkBatchSize {
		end := start + bulkBatchSize
		if end > len(items) {
			end = len(items)
		}
		batch := items[start:end]
		if err := techComponentsInsertBatch(tx, batch, userID); err != nil {
			return failed, err
		}
	}

	if err := tx.Commit(); err != nil {
		return failed, err
	}
	return failed, nil
}

func techComponentsInsertBatch(tx *sql.Tx, items []models.TechComponent, userID int) error {
	valid := make([]models.TechComponent, 0, len(items))
	for _, item := range items {
		normalizeTechComponent(&item)
		if item.FactoryCode == "" || item.UnitId == 0 || item.TypeId == 0 {
			continue
		}
		valid = append(valid, item)
	}
	if len(valid) == 0 {
		return nil
	}

	var sb strings.Builder
	args := make([]any, 0, len(valid)*17)
	sb.WriteString(`INSERT INTO production.components (detal_turi_kodi, factory_code, manufacturer_code, full_name_uz, standard_name_uz,
		full_name_ru, standard_name_ru, specification_uz, net_weight_pcs, net_weight_set, technological_waste_pcs, technological_waste_set,
		unit_id, type_id, comment, c_user_id, odoo_code) VALUES `)
	for i, item := range valid {
		if i > 0 {
			sb.WriteString(",")
		}
		n := i*17 + 1
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)",
			n, n+1, n+2, n+3, n+4, n+5, n+6, n+7, n+8, n+9, n+10, n+11, n+12, n+13, n+14, n+15, n+16))
		args = append(args,
			item.DetalTuriKodi, item.FactoryCode, item.ManufacturerCode, item.FullNameUz, item.StandardNameUz,
			item.FullNameRu, item.StandardNameRu, item.SpecificationUz, item.NetWeightPcs, item.NetWeightSet,
			item.TechnologicalWastePcs, item.TechnologicalWasteSet, item.UnitId, item.TypeId,
			item.Comment, userID, item.OdooCode,
		)
	}
	sb.WriteString(` ON CONFLICT (factory_code) WHERE factory_code <> '' DO NOTHING`)
	_, err := tx.Exec(sb.String(), args...)
	return err
}
