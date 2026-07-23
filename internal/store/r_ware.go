package store

import (
	"database/sql"
	"errors"
	"fmt"
	"math"

	"github.com/lib/pq"
)

const wareQuantityScale = 1e8

func roundWareQuantity(quantity float64) float64 {
	if math.IsNaN(quantity) || math.IsInf(quantity, 0) {
		return 0
	}
	return math.Round(quantity*wareQuantityScale) / wareQuantityScale
}

// normalizeWareQuantity rounds to NUMERIC(18,8) and requires quantity > 0.
func normalizeWareQuantity(quantity float64) (float64, bool) {
	normalized := roundWareQuantity(quantity)
	if normalized <= 0 {
		return quantity, false
	}
	return normalized, true
}

func requireNormalizedWareQuantity(quantity float64) (float64, error) {
	normalized, ok := normalizeWareQuantity(quantity)
	if !ok {
		return 0, errors.New("Detal soni 0 bo'lishi mumkin emas")
	}
	return normalized, nil
}

type WareStockRow struct {
	ID               int     `json:"id"`
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code"`
	StandardNameUz   string  `json:"standard_name_uz"`
	Quantity         float64 `json:"quantity"`
}

type WareIncomeRow struct {
	ID               int64   `json:"id"`
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code"`
	StandardNameUz   string  `json:"standard_name_uz"`
	OdooCode         string  `json:"odoo_code"`
	Quantity         float64 `json:"quantity"`
	QuantityAfter    float64 `json:"quantity_after"`
	UserID           int     `json:"user_id"`
	UserName         string  `json:"user_name"`
	UserLogin        string  `json:"user_login"`
	Comment          string  `json:"comment"`
	CreatedAt        string  `json:"created_at"`
}

type WareIncomeFilter struct {
	ComponentID int
	DateFrom    string
	DateTo      string
}

type WareDeliveryNoteListRow struct {
	ID            int64   `json:"id"`
	ModelID       int     `json:"model_id"`
	ModelName     string  `json:"model_name"`
	ModelQuantity float64 `json:"model_quantity"`
	UserID        int     `json:"user_id"`
	UserName      string  `json:"user_name"`
	CreatedAt     string  `json:"created_at"`
	ItemCount     int     `json:"item_count"`
}

type WareDeliveryNoteItemRow struct {
	ID               int64   `json:"id"`
	DeliveryNoteID   int64   `json:"delivery_note_id"`
	ComponentID      int     `json:"component_id"`
	LineID           int     `json:"line_id"`
	LineName         string  `json:"line_name"`
	ComponentName    string  `json:"component_name"`
	ManufacturerCode string  `json:"manufacturer_code"`
	FactoryCode      string  `json:"factory_code"`
	Quantity         float64 `json:"quantity"`
	QuantityActual   float64 `json:"quantity_actual"`
	IsReady          bool    `json:"is_ready"`
}

type WareDeliveryNoteSnapshotListRow struct {
	ID               int64   `json:"id"`
	DeliveryNoteID   int64   `json:"delivery_note_id"`
	ModelID          int     `json:"model_id"`
	ModelName        string  `json:"model_name"`
	ModelQuantity    float64 `json:"model_quantity"`
	UserID           int     `json:"user_id"`
	UserName         string  `json:"user_name"`
	CreatedAt        string  `json:"created_at"`
	ItemCount        int     `json:"item_count"`
}

type WareDeliveryNoteSnapshotFilter struct {
	Limit    int
	DateFrom string
	DateTo   string
	ModelID  int
}

type WareDeliveryNoteSnapshotItemRow struct {
	ID               int64   `json:"id"`
	SnapshotID       int64   `json:"snapshot_id"`
	DeliveryNoteID   int64   `json:"delivery_note_id"`
	ComponentID      int     `json:"component_id"`
	LineID           int     `json:"line_id"`
	LineName         string  `json:"line_name"`
	ComponentName    string  `json:"component_name"`
	ManufacturerCode string  `json:"manufacturer_code"`
	FactoryCode      string  `json:"factory_code"`
	Quantity         float64 `json:"quantity"`
}

type WareStockSnapshotRow struct {
	SnapshotDate     string  `json:"snapshot_date"`
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code"`
	StandardNameUz   string  `json:"standard_name_uz"`
	Quantity         float64 `json:"quantity"`
}

func (r *Repo) WareIncomeApply(componentID, userID int, quantity float64, comment string) (float64, error) {
	if componentID <= 0 {
		return 0, errors.New("komponent tanlanmagan")
	}
	var err error
	quantity, err = requireNormalizedWareQuantity(quantity)
	if err != nil {
		return 0, errors.New("miqdor 0 dan katta bo'lishi kerak")
	}
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	var componentOK int
	err = tx.QueryRow(`
		SELECT 1 FROM production.components WHERE id = $1 AND status = true`, componentID,
	).Scan(&componentOK)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, errors.New("komponent topilmadi")
		}
		return 0, err
	}

	current := 0.0
	err = tx.QueryRow(`
		SELECT quantity FROM ware.stock WHERE component_id = $1`, componentID,
	).Scan(&current)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}

	current = roundWareQuantity(current)
	quantityAfter := roundWareQuantity(current + quantity)

	_, err = tx.Exec(`
		INSERT INTO ware.stock (component_id, quantity)
		VALUES ($1, $2)
		ON CONFLICT (component_id)
		DO UPDATE SET quantity = EXCLUDED.quantity`,
		componentID, quantityAfter,
	)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO ware.income (component_id, quantity, quantity_after, user_id, comment)
		VALUES ($1, $2, $3, $4, $5)`,
		componentID, quantity, quantityAfter, userID, comment,
	)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO ware.stock_snapshots (snapshot_date, component_id, quantity, updated_at)
		VALUES (CURRENT_DATE, $1, $2, now())
		ON CONFLICT (snapshot_date, component_id)
		DO UPDATE SET
			quantity = EXCLUDED.quantity,
			updated_at = now()`,
		componentID, quantityAfter,
	)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return quantityAfter, nil
}

type WareOutcomeParams struct {
	ComponentID         int
	Quantity            float64
	UserID              int
	DeliveryNoteID      int64
	DeliveryNoteItemID  int64
	LineID              int
	Comment             string
}

func syncWareOutcomeIDSequenceTx(tx *sql.Tx) error {
	_, err := tx.Exec(`
		SELECT setval(
			pg_get_serial_sequence('ware.outcome', 'id'),
			GREATEST(COALESCE((SELECT MAX(id) FROM ware.outcome), 0), 1)
		)`)
	return err
}

func (r *Repo) wareOutcomeApplyTx(tx *sql.Tx, p WareOutcomeParams) (float64, error) {
	if p.ComponentID <= 0 {
		return 0, errors.New("komponent tanlanmagan")
	}
	if p.UserID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}

	quantity, err := requireNormalizedWareQuantity(p.Quantity)
	if err != nil {
		return 0, err
	}
	p.Quantity = quantity

	current := 0.0
	err = tx.QueryRow(`
		SELECT quantity FROM ware.stock WHERE component_id = $1 FOR UPDATE`,
		p.ComponentID,
	).Scan(&current)
	if err != nil && err != sql.ErrNoRows {
		return 0, err
	}
	current = roundWareQuantity(current)

	if p.Quantity > current {
		return 0, fmt.Errorf("omborda yetarli emas (joriy: %.8f)", current)
	}

	quantityAfter := roundWareQuantity(current - p.Quantity)

	_, err = tx.Exec(`
		INSERT INTO ware.stock (component_id, quantity)
		VALUES ($1, $2)
		ON CONFLICT (component_id)
		DO UPDATE SET quantity = EXCLUDED.quantity`,
		p.ComponentID, quantityAfter,
	)
	if err != nil {
		return 0, err
	}

	if err := syncWareOutcomeIDSequenceTx(tx); err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO ware.outcome (
			component_id, quantity, quantity_after, user_id,
			delivery_note_id, delivery_note_item_id, line_id, comment
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, 0), NULLIF($6, 0), NULLIF($7, 0), $8)`,
		p.ComponentID,
		p.Quantity,
		quantityAfter,
		p.UserID,
		p.DeliveryNoteID,
		p.DeliveryNoteItemID,
		p.LineID,
		p.Comment,
	)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO ware.stock_snapshots (snapshot_date, component_id, quantity, updated_at)
		VALUES (CURRENT_DATE, $1, $2, now())
		ON CONFLICT (snapshot_date, component_id)
		DO UPDATE SET
			quantity = EXCLUDED.quantity,
			updated_at = now()`,
		p.ComponentID, quantityAfter,
	)
	if err != nil {
		return 0, err
	}

	return quantityAfter, nil
}

func (r *Repo) WareStockGetAll() ([]WareStockRow, error) {
	rows, err := r.store.db.Query(`
		SELECT s.id, c.id, c.manufacturer_code, c.standard_name_uz, s.quantity
		FROM ware.stock s
		INNER JOIN production.components c ON c.id = s.component_id
		WHERE c.status = true AND s.quantity <> 0
		ORDER BY c.manufacturer_code`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareStockRow{}
	for rows.Next() {
		item := WareStockRow{}
		if err := rows.Scan(&item.ID, &item.ComponentID, &item.ManufacturerCode, &item.StandardNameUz, &item.Quantity); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func (r *Repo) WareStockSnapshotGet(snapshotDate string) ([]WareStockSnapshotRow, error) {
	rows, err := r.store.db.Query(`
		SELECT to_char($1::date, 'YYYY-MM-DD') AS snapshot_date,
			c.id,
			COALESCE(c.manufacturer_code, ''),
			COALESCE(c.standard_name_uz, ''),
			COALESCE(ss.quantity, 0)
		FROM production.components c
		LEFT JOIN LATERAL (
			SELECT s.quantity
			FROM ware.stock_snapshots s
			WHERE s.component_id = c.id
			  AND s.snapshot_date <= $1::date
			ORDER BY s.snapshot_date DESC
			LIMIT 1
		) ss ON true
		WHERE c.status = true
		ORDER BY c.manufacturer_code`, snapshotDate,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareStockSnapshotRow{}
	for rows.Next() {
		item := WareStockSnapshotRow{}
		if err := rows.Scan(
			&item.SnapshotDate,
			&item.ComponentID,
			&item.ManufacturerCode,
			&item.StandardNameUz,
			&item.Quantity,
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

func (r *Repo) WareDeliveryNotesList(limit int) ([]WareDeliveryNoteListRow, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	rows, err := r.store.db.Query(`
		SELECT dn.id,
			COALESCE(dn.model_id, 0),
			COALESCE(dn.model_name, ''),
			COALESCE(dn.model_quantity, 0),
			dn.user_id,
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(dn.created_at, 'YYYY-MM-DD HH24:MI:SS'),
			COALESCE(cnt.item_count, 0)
		FROM ware.delivery_notes dn
		LEFT JOIN auth.users u ON u.id = dn.user_id
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS item_count
			FROM ware.delivery_note_items dni
			WHERE dni.delivery_note_id = dn.id
			  AND COALESCE(dni.is_received, false) = false
		) cnt ON true
		WHERE COALESCE(cnt.item_count, 0) > 0
		ORDER BY dn.id DESC
		LIMIT $1`, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareDeliveryNoteListRow{}
	for rows.Next() {
		item := WareDeliveryNoteListRow{}
		if err := rows.Scan(
			&item.ID,
			&item.ModelID,
			&item.ModelName,
			&item.ModelQuantity,
			&item.UserID,
			&item.UserName,
			&item.CreatedAt,
			&item.ItemCount,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func (r *Repo) WareDeliveryNoteItemsGet(deliveryNoteID int64) ([]WareDeliveryNoteItemRow, error) {
	if deliveryNoteID <= 0 {
		return nil, errors.New("nakladnoma tanlanmagan")
	}

	rows, err := r.store.db.Query(`
		SELECT id, delivery_note_id, component_id, line_id,
			COALESCE(line_name, ''),
			COALESCE(component_name, ''),
			COALESCE(manufacturer_code, ''),
			COALESCE(factory_code, ''),
			COALESCE(quantity, 0),
			COALESCE(quantity_actual, quantity, 0),
			COALESCE(is_ready, false)
		FROM ware.delivery_note_items
		WHERE delivery_note_id = $1
		  AND COALESCE(is_received, false) = false
		ORDER BY id`, deliveryNoteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareDeliveryNoteItemRow{}
	for rows.Next() {
		item := WareDeliveryNoteItemRow{}
		if err := rows.Scan(
			&item.ID,
			&item.DeliveryNoteID,
			&item.ComponentID,
			&item.LineID,
			&item.LineName,
			&item.ComponentName,
			&item.ManufacturerCode,
			&item.FactoryCode,
			&item.Quantity,
			&item.QuantityActual,
			&item.IsReady,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func (r *Repo) WareDeliveryNoteSnapshotsList(filter WareDeliveryNoteSnapshotFilter) ([]WareDeliveryNoteSnapshotListRow, error) {
	limit := filter.Limit
	if limit <= 0 || limit > 500 {
		limit = 200
	}

	query := `
		SELECT s.id,
			s.delivery_note_id,
			COALESCE(s.model_id, 0),
			COALESCE(s.model_name, ''),
			COALESCE(s.model_quantity, 0),
			s.user_id,
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(s.created_at, 'YYYY-MM-DD HH24:MI:SS'),
			COALESCE(cnt.item_count, 0)
		FROM ware.delivery_note_snapshots s
		LEFT JOIN auth.users u ON u.id = s.user_id
		LEFT JOIN LATERAL (
			SELECT COUNT(*)::int AS item_count
			FROM ware.delivery_note_snapshot_items si
			WHERE si.snapshot_id = s.id
		) cnt ON true
		WHERE 1 = 1`

	args := []any{}
	argPos := 1

	if filter.ModelID > 0 {
		query += fmt.Sprintf(" AND s.model_id = $%d", argPos)
		args = append(args, filter.ModelID)
		argPos++
	}
	if filter.DateFrom != "" {
		query += fmt.Sprintf(" AND s.created_at >= $%d::date", argPos)
		args = append(args, filter.DateFrom)
		argPos++
	}
	if filter.DateTo != "" {
		query += fmt.Sprintf(" AND s.created_at < ($%d::date + INTERVAL '1 day')", argPos)
		args = append(args, filter.DateTo)
		argPos++
	}

	query += fmt.Sprintf(" ORDER BY s.id DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareDeliveryNoteSnapshotListRow{}
	for rows.Next() {
		item := WareDeliveryNoteSnapshotListRow{}
		if err := rows.Scan(
			&item.ID,
			&item.DeliveryNoteID,
			&item.ModelID,
			&item.ModelName,
			&item.ModelQuantity,
			&item.UserID,
			&item.UserName,
			&item.CreatedAt,
			&item.ItemCount,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func (r *Repo) WareDeliveryNoteSnapshotItemsGet(deliveryNoteID int64) ([]WareDeliveryNoteSnapshotItemRow, error) {
	if deliveryNoteID <= 0 {
		return nil, errors.New("nakladnoma tanlanmagan")
	}

	rows, err := r.store.db.Query(`
		SELECT si.id,
			si.snapshot_id,
			s.delivery_note_id,
			si.component_id,
			si.line_id,
			COALESCE(si.line_name, ''),
			COALESCE(si.component_name, ''),
			COALESCE(si.manufacturer_code, ''),
			COALESCE(si.factory_code, ''),
			COALESCE(si.quantity, 0)
		FROM ware.delivery_note_snapshot_items si
		INNER JOIN ware.delivery_note_snapshots s ON s.id = si.snapshot_id
		WHERE s.delivery_note_id = $1
		ORDER BY si.id`, deliveryNoteID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareDeliveryNoteSnapshotItemRow{}
	for rows.Next() {
		item := WareDeliveryNoteSnapshotItemRow{}
		if err := rows.Scan(
			&item.ID,
			&item.SnapshotID,
			&item.DeliveryNoteID,
			&item.ComponentID,
			&item.LineID,
			&item.LineName,
			&item.ComponentName,
			&item.ManufacturerCode,
			&item.FactoryCode,
			&item.Quantity,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

// wareDeliveryNoteSnapshotCreateTx writes an immutable archive row at nakladnoya creation.
// Live note/item updates and deletes must never touch snapshot tables.
func (r *Repo) wareDeliveryNoteSnapshotCreateTx(
	tx *sql.Tx,
	noteID int64,
	input WareDeliveryNoteCreateInput,
) (int64, error) {
	var snapshotID int64
	err := tx.QueryRow(`
		INSERT INTO ware.delivery_note_snapshots (
			delivery_note_id, model_id, model_name, model_quantity, user_id, created_at
		)
		SELECT $1, NULLIF($2, 0), $3, $4, $5, dn.created_at
		FROM ware.delivery_notes dn
		WHERE dn.id = $1
		RETURNING id`,
		noteID,
		input.ModelID,
		input.ModelName,
		input.ModelQuantity,
		input.UserID,
	).Scan(&snapshotID)
	if err != nil {
		return 0, err
	}
	return snapshotID, nil
}

func (r *Repo) wareDeliveryNoteSnapshotItemCreateTx(
	tx *sql.Tx,
	snapshotID int64,
	item WareDeliveryNoteItemInput,
) error {
	_, err := tx.Exec(`
		INSERT INTO ware.delivery_note_snapshot_items (
			snapshot_id, component_id, line_id, line_name,
			component_name, manufacturer_code, factory_code, quantity
		)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		snapshotID,
		item.ComponentID,
		item.LineID,
		item.LineName,
		item.ComponentName,
		item.ManufacturerCode,
		item.FactoryCode,
		item.Quantity,
	)
	return err
}

func (r *Repo) WareDeliveryNoteItemUpdateQuantity(itemID int64, quantityActual float64) error {
	if itemID <= 0 {
		return errors.New("pozitsiya tanlanmagan")
	}
	var err error
	quantityActual, err = requireNormalizedWareQuantity(quantityActual)
	if err != nil {
		return err
	}

	result, err := r.store.db.Exec(`
		UPDATE ware.delivery_note_items
		SET quantity_actual = $1
		WHERE id = $2 AND is_ready = false`,
		quantityActual, itemID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("pozitsiya topilmadi yoki allaqachon tayyor")
	}
	return nil
}

func (r *Repo) WareDeliveryNoteItemConfirm(itemID int64, quantityActual float64) error {
	if itemID <= 0 {
		return errors.New("pozitsiya tanlanmagan")
	}
	var err error
	quantityActual, err = requireNormalizedWareQuantity(quantityActual)
	if err != nil {
		return err
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var componentID int
	var isReady bool
	var componentName, manufacturerCode, lineName string
	err = tx.QueryRow(`
		SELECT component_id, is_ready,
			COALESCE(component_name, ''),
			COALESCE(manufacturer_code, ''),
			COALESCE(line_name, '')
		FROM ware.delivery_note_items
		WHERE id = $1
		FOR UPDATE`,
		itemID,
	).Scan(&componentID, &isReady, &componentName, &manufacturerCode, &lineName)
	if err != nil {
		if err == sql.ErrNoRows {
			return errors.New("pozitsiya topilmadi")
		}
		return err
	}
	if isReady {
		return errors.New("pozitsiya allaqachon tayyor")
	}

	availableQty := 0.0
	err = tx.QueryRow(`
		SELECT quantity
		FROM ware.stock
		WHERE component_id = $1
		FOR UPDATE`,
		componentID,
	).Scan(&availableQty)
	if err != nil && err != sql.ErrNoRows {
		return err
	}

	if quantityActual > availableQty {
		return WareInsufficientStockError{
			Shortages: []WareStockShortage{{
				ComponentID:       componentID,
				ManufacturerCode:  manufacturerCode,
				ComponentName:     componentName,
				LineName:          lineName,
				RequiredQuantity:  quantityActual,
				AvailableQuantity: availableQty,
				MissingQuantity:   quantityActual - availableQty,
			}},
		}
	}

	result, err := tx.Exec(`
		UPDATE ware.delivery_note_items
		SET quantity_actual = $1, is_ready = true
		WHERE id = $2 AND is_ready = false`,
		quantityActual, itemID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("pozitsiya topilmadi yoki allaqachon tayyor")
	}

	return tx.Commit()
}

func (r *Repo) WareDeliveryNoteItemUnconfirm(itemID int64) error {
	if itemID <= 0 {
		return errors.New("pozitsiya tanlanmagan")
	}

	result, err := r.store.db.Exec(`
		UPDATE ware.delivery_note_items
		SET is_ready = false
		WHERE id = $1 AND is_ready = true`,
		itemID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("pozitsiya topilmadi yoki tayyor emas")
	}
	return nil
}

type WareDeliveryNoteBulkConfirmItemInput struct {
	ItemID         int64
	QuantityActual float64
}

func (r *Repo) WareDeliveryNoteItemsConfirmAll(
	deliveryNoteID int64,
	inputs []WareDeliveryNoteBulkConfirmItemInput,
) (int, error) {
	if deliveryNoteID <= 0 {
		return 0, errors.New("nakladnoma tanlanmagan")
	}

	qtyByItemID := make(map[int64]float64, len(inputs))
	for _, input := range inputs {
		if input.ItemID <= 0 {
			continue
		}
		qtyByItemID[input.ItemID] = input.QuantityActual
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	rows, err := tx.Query(`
		SELECT id, component_id,
			COALESCE(quantity_actual, quantity, 0),
			COALESCE(component_name, ''),
			COALESCE(manufacturer_code, ''),
			COALESCE(line_name, '')
		FROM ware.delivery_note_items
		WHERE delivery_note_id = $1 AND is_ready = false
		ORDER BY id
		FOR UPDATE`,
		deliveryNoteID,
	)
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	type pendingItem struct {
		ID               int64
		ComponentID      int
		DefaultQty       float64
		ComponentName    string
		ManufacturerCode string
		LineName         string
	}

	pending := make([]pendingItem, 0)
	componentIDs := make(map[int]struct{})
	for rows.Next() {
		item := pendingItem{}
		if err := rows.Scan(
			&item.ID,
			&item.ComponentID,
			&item.DefaultQty,
			&item.ComponentName,
			&item.ManufacturerCode,
			&item.LineName,
		); err != nil {
			return 0, err
		}
		pending = append(pending, item)
		componentIDs[item.ComponentID] = struct{}{}
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(pending) == 0 {
		return 0, errors.New("tasdiqlash uchun pozitsiyalar yo'q")
	}

	availableByComponent := make(map[int]float64)
	if len(componentIDs) > 0 {
		componentIDList := make([]int, 0, len(componentIDs))
		for id := range componentIDs {
			componentIDList = append(componentIDList, id)
		}

		stockRows, err := tx.Query(`
			SELECT component_id, quantity
			FROM ware.stock
			WHERE component_id = ANY($1)
			FOR UPDATE`,
			pq.Array(componentIDList),
		)
		if err != nil {
			return 0, err
		}
		defer stockRows.Close()

		for stockRows.Next() {
			var componentID int
			var qty float64
			if err := stockRows.Scan(&componentID, &qty); err != nil {
				return 0, err
			}
			availableByComponent[componentID] = qty
		}
		if err := stockRows.Err(); err != nil {
			return 0, err
		}
	}

	shortages := make([]WareStockShortage, 0)
	confirmedQtyByItem := make(map[int64]float64, len(pending))
	for _, item := range pending {
		quantityActual, ok := qtyByItemID[item.ID]
		if !ok || quantityActual <= 0 {
			quantityActual = item.DefaultQty
		}
		var err error
		quantityActual, err = requireNormalizedWareQuantity(quantityActual)
		if err != nil {
			return 0, err
		}

		availableQty := availableByComponent[item.ComponentID]
		if quantityActual > availableQty {
			shortages = append(shortages, WareStockShortage{
				ComponentID:       item.ComponentID,
				ManufacturerCode:  item.ManufacturerCode,
				ComponentName:     item.ComponentName,
				LineName:          item.LineName,
				RequiredQuantity:  quantityActual,
				AvailableQuantity: availableQty,
				MissingQuantity:   quantityActual - availableQty,
			})
		}
		confirmedQtyByItem[item.ID] = quantityActual
	}
	if len(shortages) > 0 {
		return 0, WareInsufficientStockError{Shortages: shortages}
	}

	confirmedCount := 0
	for _, item := range pending {
		result, err := tx.Exec(`
			UPDATE ware.delivery_note_items
			SET quantity_actual = $1, is_ready = true
			WHERE id = $2 AND is_ready = false`,
			confirmedQtyByItem[item.ID], item.ID,
		)
		if err != nil {
			return 0, err
		}
		rowsAffected, err := result.RowsAffected()
		if err != nil {
			return 0, err
		}
		confirmedCount += int(rowsAffected)
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return confirmedCount, nil
}

func (r *Repo) WareDeliveryNoteItemsUnconfirmAll(deliveryNoteID int64) (int, error) {
	if deliveryNoteID <= 0 {
		return 0, errors.New("nakladnoma tanlanmagan")
	}

	result, err := r.store.db.Exec(`
		UPDATE ware.delivery_note_items
		SET is_ready = false
		WHERE delivery_note_id = $1 AND is_ready = true`,
		deliveryNoteID,
	)
	if err != nil {
		return 0, err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	return int(rowsAffected), nil
}

func (r *Repo) WareDeliveryNoteItemDelete(itemID int64) error {
	if itemID <= 0 {
		return errors.New("pozitsiya tanlanmagan")
	}

	result, err := r.store.db.Exec(`
		DELETE FROM ware.delivery_note_items
		WHERE id = $1 AND is_ready = false`,
		itemID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("pozitsiya topilmadi yoki allaqachon tayyor")
	}
	return nil
}

type WareRequestRow struct {
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code"`
	ComponentName    string  `json:"component_name"`
	FactoryCode      string  `json:"factory_code"`
	LineID         int     `json:"line_id"`
	LineName       string  `json:"line_name"`
	NormQuantity   float64 `json:"norm_quantity"`
	LineQuantity   float64 `json:"line_quantity"`
	WareQuantity   float64 `json:"ware_quantity"`
	NeededQuantity float64 `json:"needed_quantity"`
}

type wareRequestAccumulator struct {
	ManufacturerCode string
	ComponentName    string
	FactoryCode      string
	LineName         string
	NormQuantity     float64
}

func (r *Repo) WareRequestBuild(modelID int, quantity float64) ([]WareRequestRow, error) {
	if modelID <= 0 {
		return nil, errors.New("model tanlanmagan")
	}
	if quantity <= 0 {
		return nil, errors.New("miqdor 0 dan katta bo'lishi kerak")
	}

	normItems, err := r.ConsumptionNormItemsByModel(modelID)
	if err != nil {
		return nil, err
	}
	if len(normItems) == 0 {
		return nil, errors.New("model uchun sarf normasi topilmadi")
	}

	aggregated := make(map[string]*wareRequestAccumulator)
	lineIDs := make(map[int]struct{})
	componentIDs := make(map[int]struct{})

	for _, item := range normItems {
		if item.Quantity <= 0 {
			continue
		}
		key := fmt.Sprintf("%d:%d", item.ComponentID, item.ConsumeLineID)
		if aggregated[key] == nil {
			aggregated[key] = &wareRequestAccumulator{
				ManufacturerCode: item.ManufacturerCode,
				ComponentName:    item.StandardNameUz,
				FactoryCode:      item.FactoryCode,
				LineName:         item.ConsumeLineName,
			}
		}
		aggregated[key].NormQuantity += item.Quantity * quantity
		lineIDs[item.ConsumeLineID] = struct{}{}
		componentIDs[item.ComponentID] = struct{}{}
	}

	if len(aggregated) == 0 {
		return nil, errors.New("sarf normasida miqdorli komponentlar topilmadi")
	}

	balanceMap, err := r.linesBalanceLookup(lineIDs, componentIDs)
	if err != nil {
		return nil, err
	}
	wareStockMap, err := r.wareStockLookup(componentIDs)
	if err != nil {
		return nil, err
	}

	result := make([]WareRequestRow, 0, len(aggregated))
	for key, acc := range aggregated {
		var componentID, lineID int
		_, _ = fmt.Sscanf(key, "%d:%d", &componentID, &lineID)

		lineQty := balanceMap[balanceKey(lineID, componentID)]
		shortage := acc.NormQuantity - lineQty
		if shortage < 0 {
			shortage = 0
		}

		result = append(result, WareRequestRow{
			ComponentID:      componentID,
			ManufacturerCode: acc.ManufacturerCode,
			ComponentName:    acc.ComponentName,
			FactoryCode:      acc.FactoryCode,
			LineID:         lineID,
			LineName:       acc.LineName,
			NormQuantity:   acc.NormQuantity,
			LineQuantity:   lineQty,
			WareQuantity:   wareStockMap[componentID],
			NeededQuantity: shortage,
		})
	}

	sortWareRequestRows(result)
	return result, nil
}

func balanceKey(lineID, componentID int) string {
	return fmt.Sprintf("%d:%d", lineID, componentID)
}

func (r *Repo) linesBalanceLookup(lineIDs, componentIDs map[int]struct{}) (map[string]float64, error) {
	result := make(map[string]float64)
	if len(lineIDs) == 0 || len(componentIDs) == 0 {
		return result, nil
	}

	lineIDList := make([]int, 0, len(lineIDs))
	for id := range lineIDs {
		if id > 0 {
			lineIDList = append(lineIDList, id)
		}
	}
	componentIDList := make([]int, 0, len(componentIDs))
	for id := range componentIDs {
		componentIDList = append(componentIDList, id)
	}
	if len(lineIDList) == 0 {
		return result, nil
	}

	rows, err := r.store.db.Query(`
		SELECT line_id, component_id, quantity
		FROM lines.balance
		WHERE line_id = ANY($1) AND component_id = ANY($2)`,
		pq.Array(lineIDList), pq.Array(componentIDList),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var lineID, componentID int
		var qty float64
		if err := rows.Scan(&lineID, &componentID, &qty); err != nil {
			return result, err
		}
		result[balanceKey(lineID, componentID)] = qty
	}
	return result, rows.Err()
}

func (r *Repo) wareStockLookup(componentIDs map[int]struct{}) (map[int]float64, error) {
	result := make(map[int]float64)
	if len(componentIDs) == 0 {
		return result, nil
	}

	componentIDList := make([]int, 0, len(componentIDs))
	for id := range componentIDs {
		componentIDList = append(componentIDList, id)
	}

	rows, err := r.store.db.Query(`
		SELECT component_id, quantity
		FROM ware.stock
		WHERE component_id = ANY($1)`,
		pq.Array(componentIDList),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var componentID int
		var qty float64
		if err := rows.Scan(&componentID, &qty); err != nil {
			return result, err
		}
		result[componentID] = qty
	}
	return result, rows.Err()
}

type WareDeliveryNoteItemInput struct {
	ComponentID      int
	LineID           int
	LineName         string
	ComponentName    string
	ManufacturerCode string
	FactoryCode      string
	Quantity         float64
}

type WareDeliveryNoteCreateInput struct {
	ModelID       int
	ModelName     string
	ModelQuantity float64
	UserID        int
	Items         []WareDeliveryNoteItemInput
}

type WareStockShortage struct {
	ComponentID       int     `json:"component_id"`
	ManufacturerCode  string  `json:"manufacturer_code"`
	ComponentName     string  `json:"component_name"`
	LineName          string  `json:"line_name,omitempty"`
	RequiredQuantity  float64 `json:"required_quantity"`
	AvailableQuantity float64 `json:"available_quantity"`
	MissingQuantity   float64 `json:"missing_quantity"`
}

type WareInsufficientStockError struct {
	Shortages []WareStockShortage
}

func (e WareInsufficientStockError) Error() string {
	return "omborda komponent yetarli emas"
}

type WareInvalidQuantityItem struct {
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code,omitempty"`
	ComponentName    string  `json:"component_name,omitempty"`
	LineName         string  `json:"line_name,omitempty"`
	Quantity         float64 `json:"quantity"`
}

type WareInvalidQuantityError struct {
	Items []WareInvalidQuantityItem
}

func (e WareInvalidQuantityError) Error() string {
	return "Detal soni 0 bo'lishi mumkin emas"
}

func ValidateWareDeliveryNoteItems(items []WareDeliveryNoteItemInput) ([]WareDeliveryNoteItemInput, error) {
	invalid := make([]WareInvalidQuantityItem, 0)
	valid := make([]WareDeliveryNoteItemInput, 0, len(items))

	for _, item := range items {
		qty, ok := normalizeWareQuantity(item.Quantity)
		if !ok {
			invalid = append(invalid, WareInvalidQuantityItem{
				ComponentID:      item.ComponentID,
				ManufacturerCode: item.ManufacturerCode,
				ComponentName:    item.ComponentName,
				LineName:         item.LineName,
				Quantity:         item.Quantity,
			})
			continue
		}
		item.Quantity = qty
		valid = append(valid, item)
	}

	if len(invalid) > 0 {
		return nil, WareInvalidQuantityError{Items: invalid}
	}
	if len(valid) == 0 {
		return nil, errors.New("kerakli miqdorli komponentlar yo'q")
	}
	return valid, nil
}

type WareDeliveryNoteResult struct {
	ID        int64 `json:"id"`
	ItemCount int   `json:"item_count"`
}

func (r *Repo) WareDeliveryNoteCreate(input WareDeliveryNoteCreateInput) (WareDeliveryNoteResult, error) {
	result := WareDeliveryNoteResult{}
	if input.UserID <= 0 {
		return result, errors.New("foydalanuvchi aniqlanmadi")
	}
	if len(input.Items) == 0 {
		return result, errors.New("nakladnoma uchun komponentlar yo'q")
	}

	input.ModelQuantity = roundWareQuantity(input.ModelQuantity)

	validatedItems, err := ValidateWareDeliveryNoteItems(input.Items)
	if err != nil {
		return result, err
	}
	input.Items = validatedItems

	tx, err := r.store.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	// Per row: Omborda >= Kerakli miqdor (line_id may be 0 when norm has no consume line).
	componentIDs := make([]int, 0, len(input.Items))
	seenComponent := make(map[int]struct{})

	for _, item := range input.Items {
		if item.ComponentID <= 0 {
			return result, errors.New("noto'g'ri komponent yoki miqdor")
		}
		if _, ok := seenComponent[item.ComponentID]; !ok {
			seenComponent[item.ComponentID] = struct{}{}
			componentIDs = append(componentIDs, item.ComponentID)
		}
	}

	availableByComponent := make(map[int]float64)
	if len(componentIDs) > 0 {
		rows, err := tx.Query(`
			SELECT component_id, quantity
			FROM ware.stock
			WHERE component_id = ANY($1)
			FOR UPDATE`,
			pq.Array(componentIDs),
		)
		if err != nil {
			return result, err
		}
		defer rows.Close()

		for rows.Next() {
			var componentID int
			var qty float64
			if err := rows.Scan(&componentID, &qty); err != nil {
				return result, err
			}
			availableByComponent[componentID] = qty
		}
		if err := rows.Err(); err != nil {
			return result, err
		}
	}

	shortages := make([]WareStockShortage, 0)
	for _, item := range input.Items {
		availableQty := availableByComponent[item.ComponentID]
		if item.Quantity > availableQty {
			shortages = append(shortages, WareStockShortage{
				ComponentID:       item.ComponentID,
				ManufacturerCode:  item.ManufacturerCode,
				ComponentName:     item.ComponentName,
				LineName:          item.LineName,
				RequiredQuantity:  item.Quantity,
				AvailableQuantity: availableQty,
				MissingQuantity:   item.Quantity - availableQty,
			})
		}
	}
	if len(shortages) > 0 {
		return result, WareInsufficientStockError{Shortages: shortages}
	}

	var noteID int64
	err = tx.QueryRow(`
		INSERT INTO ware.delivery_notes (model_id, model_name, model_quantity, user_id)
		VALUES (NULLIF($1, 0), $2, $3, $4)
		RETURNING id`,
		input.ModelID, input.ModelName, input.ModelQuantity, input.UserID,
	).Scan(&noteID)
	if err != nil {
		return result, err
	}

	snapshotID, err := r.wareDeliveryNoteSnapshotCreateTx(tx, noteID, input)
	if err != nil {
		return result, err
	}

	for _, item := range input.Items {
		var componentOK int
		err = tx.QueryRow(`
			SELECT 1 FROM production.components WHERE id = $1 AND status = true`,
			item.ComponentID,
		).Scan(&componentOK)
		if err != nil {
			if err == sql.ErrNoRows {
				return result, errors.New("komponent topilmadi")
			}
			return result, err
		}

		_, err = tx.Exec(`
			INSERT INTO ware.delivery_note_items (
				delivery_note_id, component_id, line_id, line_name,
				component_name, manufacturer_code, factory_code,
				quantity, quantity_actual, is_ready
			)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8, false)`,
			noteID,
			item.ComponentID,
			item.LineID,
			item.LineName,
			item.ComponentName,
			item.ManufacturerCode,
			item.FactoryCode,
			item.Quantity,
		)
		if err != nil {
			return result, err
		}

		if err = r.wareDeliveryNoteSnapshotItemCreateTx(tx, snapshotID, item); err != nil {
			return result, err
		}
		result.ItemCount++
	}

	if err = tx.Commit(); err != nil {
		return result, err
	}

	result.ID = noteID
	return result, nil
}

func sortWareRequestRows(rows []WareRequestRow) {
	for i := 0; i < len(rows); i++ {
		for j := i + 1; j < len(rows); j++ {
			left := rows[i].LineName + rows[i].ComponentName
			right := rows[j].LineName + rows[j].ComponentName
			if left > right {
				rows[i], rows[j] = rows[j], rows[i]
			}
		}
	}
}

func (r *Repo) WareIncomeHistoryGet(filter WareIncomeFilter) ([]WareIncomeRow, error) {
	query := `
		SELECT i.id, i.component_id, c.manufacturer_code, c.standard_name_uz, COALESCE(c.odoo_code, ''),
			i.quantity, i.quantity_after, i.user_id,
			COALESCE(NULLIF(u.name, ''), u.login, ''), COALESCE(u.login, ''),
			COALESCE(i.comment, ''),
			to_char(i.created_at, 'YYYY-MM-DD HH24:MI:SS')
		FROM ware.income i
		INNER JOIN production.components c ON c.id = i.component_id
		LEFT JOIN auth.users u ON u.id = i.user_id
		WHERE 1 = 1`

	args := []any{}
	argPos := 1

	if filter.ComponentID > 0 {
		query += fmt.Sprintf(" AND i.component_id = $%d", argPos)
		args = append(args, filter.ComponentID)
		argPos++
	}
	if filter.DateFrom != "" {
		query += fmt.Sprintf(" AND i.created_at >= $%d::date", argPos)
		args = append(args, filter.DateFrom)
		argPos++
	}
	if filter.DateTo != "" {
		query += fmt.Sprintf(" AND i.created_at < ($%d::date + INTERVAL '1 day')", argPos)
		args = append(args, filter.DateTo)
		argPos++
	}

	query += " ORDER BY i.created_at DESC, i.id DESC LIMIT 5000"

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareIncomeRow{}
	for rows.Next() {
		item := WareIncomeRow{}
		if err := rows.Scan(
			&item.ID, &item.ComponentID, &item.ManufacturerCode, &item.StandardNameUz, &item.OdooCode,
			&item.Quantity, &item.QuantityAfter, &item.UserID, &item.UserName, &item.UserLogin,
			&item.Comment, &item.CreatedAt,
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
