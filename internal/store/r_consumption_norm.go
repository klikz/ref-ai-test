package store

import (
	"database/sql"
	"fmt"
	"strings"

	"github.com/klikz/api_v3/internal/models"
	"github.com/lib/pq"
)

type ConsumptionNormWriteItem struct {
	SortOrder     int
	GroupLevel    int
	ComponentID   int
	Quantity      float64
	ConsumeLineID sql.NullInt64
	ReceiveLineID sql.NullInt64
}

func (r *Repo) ConsumptionNormModelsSummary() ([]models.ConsumptionNormModelSummary, error) {
	rows, err := r.store.db.Query(`
		SELECT m.id, COALESCE(m.qisqa_nomi, ''), COALESCE(m.modeli, ''), COALESCE(m.seriya_raqami, ''),
			COALESCE(m.brend, ''), COALESCE(m.umumiy_hajmi_l, ''), COALESCE(cnt.item_count, 0)
		FROM production.models m
		LEFT JOIN (
			SELECT model_id, COUNT(*) AS item_count
			FROM production.consumption_norm_items
			WHERE status = true
			GROUP BY model_id
		) cnt ON cnt.model_id = m.id
		WHERE m.status = true
		ORDER BY COALESCE(m.qisqa_nomi, m.modeli), m.id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.ConsumptionNormModelSummary{}
	for rows.Next() {
		item := models.ConsumptionNormModelSummary{}
		if err := rows.Scan(
			&item.ID,
			&item.ModelNomi,
			&item.Modeli,
			&item.SeriyaRaqami,
			&item.Brend,
			&item.UmumiyHajmiL,
			&item.ItemCount,
		); err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repo) ConsumptionNormItemsByModel(modelID int) ([]models.ConsumptionNormItem, error) {
	rows, err := r.store.db.Query(`
		SELECT cn.id, cn.model_id, cn.sort_order, cn.group_level, cn.component_id,
			COALESCE(c.manufacturer_code, ''), COALESCE(c.factory_code, ''), COALESCE(c.odoo_code, ''), COALESCE(c.standard_name_uz, ''),
			cn.quantity,
			COALESCE(cn.consume_line_id, 0), COALESCE(cl.name, ''),
			COALESCE(cn.receive_line_id, 0), COALESCE(rl.name, '')
		FROM production.consumption_norm_items cn
		INNER JOIN production.components c ON c.id = cn.component_id
		LEFT JOIN lines.lines_list cl ON cl.line_id = cn.consume_line_id
		LEFT JOIN lines.lines_list rl ON rl.line_id = cn.receive_line_id
		WHERE cn.status = true AND cn.model_id = $1
		ORDER BY cn.sort_order, cn.id`, modelID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.ConsumptionNormItem{}
	for rows.Next() {
		item := models.ConsumptionNormItem{}
		if err := rows.Scan(
			&item.ID,
			&item.ModelID,
			&item.SortOrder,
			&item.GroupLevel,
			&item.ComponentID,
			&item.ManufacturerCode,
			&item.FactoryCode,
			&item.OdooCode,
			&item.StandardNameUz,
			&item.Quantity,
			&item.ConsumeLineID,
			&item.ConsumeLineName,
			&item.ReceiveLineID,
			&item.ReceiveLineName,
		); err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repo) ConsumptionNormReplace(modelID, userID int, items []ConsumptionNormWriteItem) error {
	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE production.consumption_norm_items
		SET status = false, u_time = NOW(), u_user_id = $2
		WHERE model_id = $1 AND status = true`, modelID, userID); err != nil {
		return err
	}

	if err := consumptionNormInsertBatch(tx, modelID, userID, items); err != nil {
		return err
	}

	return tx.Commit()
}

func consumptionNormInsertBatch(tx *sql.Tx, modelID, userID int, items []ConsumptionNormWriteItem) error {
	if len(items) == 0 {
		return nil
	}
	var sb strings.Builder
	args := make([]any, 0, len(items)*8)
	sb.WriteString(`INSERT INTO production.consumption_norm_items
		(model_id, sort_order, group_level, component_id, quantity, consume_line_id, receive_line_id, c_user_id) VALUES `)
	for i, item := range items {
		if i > 0 {
			sb.WriteString(",")
		}
		n := i*8 + 1
		sb.WriteString(fmt.Sprintf("($%d,$%d,$%d,$%d,$%d,$%d,$%d,$%d)", n, n+1, n+2, n+3, n+4, n+5, n+6, n+7))
		args = append(args, modelID, item.SortOrder, item.GroupLevel, item.ComponentID, item.Quantity, item.ConsumeLineID, item.ReceiveLineID, userID)
	}
	_, err := tx.Exec(sb.String(), args...)
	return err
}

func (r *Repo) ConsumptionNormUpdateItemLines(itemID, userID int, consumeLineID, receiveLineID sql.NullInt64) error {
	_, err := r.store.db.Exec(`
		UPDATE production.consumption_norm_items
		SET consume_line_id = $1,
			receive_line_id = $2,
			u_time = NOW(),
			u_user_id = $3
		WHERE id = $4 AND status = true`,
		consumeLineID,
		receiveLineID,
		userID,
		itemID,
	)
	return err
}

func consumptionNormDescendantIDs(items []models.ConsumptionNormItem, itemID int) []int {
	index := -1
	for i, item := range items {
		if item.ID == itemID {
			index = i
			break
		}
	}
	if index < 0 {
		return nil
	}

	level := items[index].GroupLevel
	ids := []int{items[index].ID}
	for i := index + 1; i < len(items); i++ {
		if items[i].GroupLevel <= level {
			break
		}
		ids = append(ids, items[i].ID)
	}
	return ids
}

func consumptionNormInsertPosition(items []models.ConsumptionNormItem, parentItemID int) (groupLevel, sortOrder int, err error) {
	if parentItemID <= 0 {
		groupLevel = 0
		sortOrder = 1
		for _, item := range items {
			if item.SortOrder >= sortOrder {
				sortOrder = item.SortOrder + 1
			}
		}
		return groupLevel, sortOrder, nil
	}

	parentIndex := -1
	for i, item := range items {
		if item.ID == parentItemID {
			parentIndex = i
			break
		}
	}
	if parentIndex < 0 {
		return 0, 0, sql.ErrNoRows
	}

	parent := items[parentIndex]
	groupLevel = parent.GroupLevel + 1
	insertAfter := parentIndex
	for i := parentIndex + 1; i < len(items); i++ {
		if items[i].GroupLevel <= parent.GroupLevel {
			break
		}
		insertAfter = i
	}
	return groupLevel, items[insertAfter].SortOrder + 1, nil
}

func (r *Repo) ComponentExistsByID(componentID int) (bool, error) {
	var exists bool
	err := r.store.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM production.components WHERE id = $1 AND status = true
		)`, componentID).Scan(&exists)
	return exists, err
}

func (r *Repo) ConsumptionNormItemByID(itemID int) (models.ConsumptionNormItem, error) {
	item := models.ConsumptionNormItem{}
	err := r.store.db.QueryRow(`
		SELECT cn.id, cn.model_id, cn.sort_order, cn.group_level, cn.component_id,
			COALESCE(c.manufacturer_code, ''), COALESCE(c.factory_code, ''), COALESCE(c.odoo_code, ''), COALESCE(c.standard_name_uz, ''),
			cn.quantity,
			COALESCE(cn.consume_line_id, 0), COALESCE(cl.name, ''),
			COALESCE(cn.receive_line_id, 0), COALESCE(rl.name, '')
		FROM production.consumption_norm_items cn
		INNER JOIN production.components c ON c.id = cn.component_id
		LEFT JOIN lines.lines_list cl ON cl.line_id = cn.consume_line_id
		LEFT JOIN lines.lines_list rl ON rl.line_id = cn.receive_line_id
		WHERE cn.status = true AND cn.id = $1`, itemID).Scan(
		&item.ID,
		&item.ModelID,
		&item.SortOrder,
		&item.GroupLevel,
		&item.ComponentID,
		&item.ManufacturerCode,
		&item.FactoryCode,
		&item.OdooCode,
		&item.StandardNameUz,
		&item.Quantity,
		&item.ConsumeLineID,
		&item.ConsumeLineName,
		&item.ReceiveLineID,
		&item.ReceiveLineName,
	)
	return item, err
}

func (r *Repo) ConsumptionNormInsertItem(
	modelID, userID, parentItemID, componentID int,
	quantity float64,
	consumeLineID, receiveLineID sql.NullInt64,
) (models.ConsumptionNormItem, error) {
	item := models.ConsumptionNormItem{}
	items, err := r.ConsumptionNormItemsByModel(modelID)
	if err != nil {
		return item, err
	}

	if parentItemID > 0 {
		parent, err := r.ConsumptionNormItemByID(parentItemID)
		if err != nil {
			return item, err
		}
		if parent.ModelID != modelID {
			return item, sql.ErrNoRows
		}
	}

	groupLevel, sortOrder, err := consumptionNormInsertPosition(items, parentItemID)
	if err != nil {
		return item, err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return item, err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(`
		UPDATE production.consumption_norm_items
		SET sort_order = sort_order + 1,
			u_time = NOW(),
			u_user_id = $3
		WHERE model_id = $1 AND status = true AND sort_order >= $2`,
		modelID, sortOrder, userID,
	); err != nil {
		return item, err
	}

	var newID int
	err = tx.QueryRow(`
		INSERT INTO production.consumption_norm_items
		(model_id, sort_order, group_level, component_id, quantity, consume_line_id, receive_line_id, c_user_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id`,
		modelID,
		sortOrder,
		groupLevel,
		componentID,
		quantity,
		consumeLineID,
		receiveLineID,
		userID,
	).Scan(&newID)
	if err != nil {
		return item, err
	}

	if err := tx.Commit(); err != nil {
		return item, err
	}

	return r.ConsumptionNormItemByID(newID)
}

func (r *Repo) ConsumptionNormDeleteItem(itemID, userID int) (int, error) {
	target, err := r.ConsumptionNormItemByID(itemID)
	if err != nil {
		return 0, err
	}

	items, err := r.ConsumptionNormItemsByModel(target.ModelID)
	if err != nil {
		return 0, err
	}

	ids := consumptionNormDescendantIDs(items, itemID)
	if len(ids) == 0 {
		return 0, sql.ErrNoRows
	}

	_, err = r.store.db.Exec(`
		UPDATE production.consumption_norm_items
		SET status = false,
			u_time = NOW(),
			u_user_id = $2
		WHERE id = ANY($1) AND status = true`,
		pq.Array(ids),
		userID,
	)
	return len(ids), err
}

func (r *Repo) ConsumptionNormItemExists(itemID int) (bool, error) {
	var exists bool
	err := r.store.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM production.consumption_norm_items WHERE id = $1 AND status = true
		)`, itemID).Scan(&exists)
	return exists, err
}

func (r *Repo) LineExistsByID(lineID int) (bool, error) {
	var exists bool
	err := r.store.db.QueryRow(`
		SELECT EXISTS(
			SELECT 1 FROM lines.lines_list WHERE line_id = $1 AND status = true
		)`, lineID).Scan(&exists)
	return exists, err
}

func (r *Repo) LinesListForLookup() ([]models.LineLookup, error) {
	rows, err := r.store.db.Query(`
		SELECT line_id, COALESCE(name, '')
		FROM lines.lines_list
		WHERE status = true
		ORDER BY name`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := []models.LineLookup{}
	for rows.Next() {
		item := models.LineLookup{}
		if err := rows.Scan(&item.LineID, &item.Name); err != nil {
			return result, err
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

func (r *Repo) ModelResolveForConsumptionNorm(modelID int, modelName string) (models.ModelInfo, error) {
	model := models.ModelInfo{}
	if modelID > 0 {
		err := r.store.db.QueryRow(`
			SELECT id, COALESCE(qisqa_nomi, ''), COALESCE(modeli, ''), COALESCE(seriya_raqami, ''), COALESCE(brend, '')
			FROM production.models
			WHERE id = $1 AND status = true`, modelID).Scan(
			&model.ID, &model.Qisqa_nomi, &model.Modeli, &model.Seriya_raqami, &model.Brend,
		)
		return model, err
	}

	name := strings.ToLower(strings.TrimSpace(modelName))
	if name == "" {
		return model, sql.ErrNoRows
	}

	err := r.store.db.QueryRow(`
		SELECT id, COALESCE(qisqa_nomi, ''), COALESCE(modeli, ''), COALESCE(seriya_raqami, ''), COALESCE(brend, '')
		FROM production.models
		WHERE status = true
		  AND (
			LOWER(TRIM(COALESCE(qisqa_nomi, ''))) = $1
			OR LOWER(TRIM(COALESCE(modeli, ''))) = $1
		  )
		ORDER BY id
		LIMIT 1`, name).Scan(
		&model.ID, &model.Qisqa_nomi, &model.Modeli, &model.Seriya_raqami, &model.Brend,
	)
	return model, err
}

func (r *Repo) ComponentsIDsByCodes(codes []string) (map[string]int, error) {
	result := map[string]int{}
	if len(codes) == 0 {
		return result, nil
	}

	rows, err := r.store.db.Query(`
		SELECT id, factory_code, odoo_code, manufacturer_code
		FROM production.components
		WHERE status = true
		  AND (
			factory_code = ANY($1)
			OR odoo_code = ANY($1)
			OR manufacturer_code = ANY($1)
		  )`, pq.Array(codes))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var factoryCode, odooCode, manufacturerCode string
		if err := rows.Scan(&id, &factoryCode, &odooCode, &manufacturerCode); err != nil {
			return result, err
		}
		for _, code := range []string{factoryCode, odooCode, manufacturerCode} {
			if code != "" {
				result[code] = id
			}
		}
	}
	return result, rows.Err()
}
