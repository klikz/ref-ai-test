package store

import (
	"database/sql"
	"errors"
	"fmt"

	"github.com/lib/pq"
)

type BrigadirDeliveryItemRow struct {
	ID               int64   `json:"id"`
	DeliveryNoteID   int64   `json:"delivery_note_id"`
	ModelName        string  `json:"model_name"`
	NoteCreatedAt    string  `json:"note_created_at"`
	ComponentID      int     `json:"component_id"`
	ComponentName    string  `json:"component_name"`
	ManufacturerCode string  `json:"manufacturer_code"`
	FactoryCode      string  `json:"factory_code"`
	LineID           int     `json:"line_id"`
	LineName         string  `json:"line_name"`
	QuantityActual   float64 `json:"quantity_actual"`
	IsReceived       bool    `json:"is_received"`
}

type brigadirPendingItem struct {
	ID               int64
	DeliveryNoteID   int64
	ComponentID      int
	LineID           int
	QuantityActual   float64
	ComponentName    string
	ManufacturerCode string
	LineName         string
}

func (r *Repo) BrigadirDeliveryItemsList(userID int) ([]BrigadirDeliveryItemRow, error) {
	if userID <= 0 {
		return nil, errors.New("foydalanuvchi aniqlanmadi")
	}

	rows, err := r.store.db.Query(`
		SELECT dni.id, dni.delivery_note_id,
			COALESCE(dn.model_name, ''),
			to_char(dn.created_at, 'YYYY-MM-DD HH24:MI:SS'),
			dni.component_id,
			COALESCE(dni.component_name, ''),
			COALESCE(dni.manufacturer_code, ''),
			COALESCE(dni.factory_code, ''),
			dni.line_id,
			COALESCE(dni.line_name, ''),
			COALESCE(dni.quantity_actual, dni.quantity, 0),
			COALESCE(dni.is_received, false)
		FROM ware.delivery_note_items dni
		INNER JOIN ware.delivery_notes dn ON dn.id = dni.delivery_note_id
		INNER JOIN lines.line_responsibles lr ON lr.line_id = dni.line_id
		WHERE lr.user_id = $1
		  AND dni.is_ready = true
		  AND COALESCE(dni.is_received, false) = false
		ORDER BY dn.created_at DESC, dni.id DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []BrigadirDeliveryItemRow{}
	for rows.Next() {
		item := BrigadirDeliveryItemRow{}
		if err := rows.Scan(
			&item.ID,
			&item.DeliveryNoteID,
			&item.ModelName,
			&item.NoteCreatedAt,
			&item.ComponentID,
			&item.ComponentName,
			&item.ManufacturerCode,
			&item.FactoryCode,
			&item.LineID,
			&item.LineName,
			&item.QuantityActual,
			&item.IsReceived,
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

func (r *Repo) brigadirPendingItemsForUserTx(tx *sql.Tx, userID int64, itemID int64) ([]brigadirPendingItem, error) {
	query := `
		SELECT dni.id, dni.delivery_note_id, dni.component_id, dni.line_id,
			COALESCE(dni.quantity_actual, dni.quantity, 0),
			COALESCE(dni.component_name, ''),
			COALESCE(dni.manufacturer_code, ''),
			COALESCE(dni.line_name, '')
		FROM ware.delivery_note_items dni
		INNER JOIN lines.line_responsibles lr ON lr.line_id = dni.line_id AND lr.user_id = $1
		WHERE dni.is_ready = true
		  AND COALESCE(dni.is_received, false) = false`
	args := []any{userID}
	if itemID > 0 {
		query += ` AND dni.id = $2`
		args = append(args, itemID)
	}
	query += ` ORDER BY dni.id FOR UPDATE OF dni`

	rows, err := tx.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]brigadirPendingItem, 0)
	for rows.Next() {
		item := brigadirPendingItem{}
		if err := rows.Scan(
			&item.ID,
			&item.DeliveryNoteID,
			&item.ComponentID,
			&item.LineID,
			&item.QuantityActual,
			&item.ComponentName,
			&item.ManufacturerCode,
			&item.LineName,
		); err != nil {
			return items, err
		}
		var normErr error
		item.QuantityActual, normErr = requireNormalizedWareQuantity(item.QuantityActual)
		if normErr != nil {
			return nil, errors.New("miqdor 0 dan katta bo'lishi kerak")
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) brigadirCheckStockForItems(tx *sql.Tx, items []brigadirPendingItem) error {
	if len(items) == 0 {
		return nil
	}

	componentIDs := make([]int, 0)
	seen := make(map[int]struct{})
	for _, item := range items {
		if _, ok := seen[item.ComponentID]; !ok {
			seen[item.ComponentID] = struct{}{}
			componentIDs = append(componentIDs, item.ComponentID)
		}
	}

	stockByComponent := make(map[int]float64)
	rows, err := tx.Query(`
		SELECT component_id, quantity
		FROM ware.stock
		WHERE component_id = ANY($1)
		FOR UPDATE`,
		pq.Array(componentIDs),
	)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var componentID int
		var qty float64
		if err := rows.Scan(&componentID, &qty); err != nil {
			return err
		}
		stockByComponent[componentID] = qty
	}
	if err := rows.Err(); err != nil {
		return err
	}

	shortages := make([]WareStockShortage, 0)
	availableSim := make(map[int]float64, len(stockByComponent))
	for id, qty := range stockByComponent {
		availableSim[id] = qty
	}

	for _, item := range items {
		availableQty := availableSim[item.ComponentID]
		if item.QuantityActual > availableQty {
			shortages = append(shortages, WareStockShortage{
				ComponentID:       item.ComponentID,
				ManufacturerCode:  item.ManufacturerCode,
				ComponentName:     item.ComponentName,
				LineName:          item.LineName,
				RequiredQuantity:  item.QuantityActual,
				AvailableQuantity: availableQty,
				MissingQuantity:   item.QuantityActual - availableQty,
			})
			continue
		}
		availableSim[item.ComponentID] -= item.QuantityActual
	}

	if len(shortages) > 0 {
		return WareInsufficientStockError{Shortages: shortages}
	}
	return nil
}

func (r *Repo) brigadirDeliveryItemConfirmTx(tx *sql.Tx, item brigadirPendingItem, userID int) error {
	comment := fmt.Sprintf("Nakladnoma #%d, pozitsiya #%d", item.DeliveryNoteID, item.ID)

	_, err := r.wareOutcomeApplyTx(tx, WareOutcomeParams{
		ComponentID:        item.ComponentID,
		Quantity:           item.QuantityActual,
		UserID:             userID,
		DeliveryNoteID:     item.DeliveryNoteID,
		DeliveryNoteItemID: item.ID,
		LineID:             item.LineID,
		Comment:            comment,
	})
	if err != nil {
		return err
	}

	_, err = r.linesBalanceApplyChangeTx(tx, BalanceChangeParams{
		LineID:         item.LineID,
		ComponentID:    item.ComponentID,
		QuantityChange: item.QuantityActual,
		UserID:         userID,
		Source:         "brigadir_receive",
		Comment:        comment,
	})
	if err != nil {
		return err
	}

	result, err := tx.Exec(`
		UPDATE ware.delivery_note_items
		SET is_received = true,
			received_at = now(),
			received_by = $2
		WHERE id = $1 AND is_ready = true AND COALESCE(is_received, false) = false`,
		item.ID, userID,
	)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}
	if rowsAffected == 0 {
		return errors.New("pozitsiya yangilanmadi")
	}
	return nil
}

func (r *Repo) BrigadirDeliveryItemConfirm(itemID int64, userID int) error {
	if itemID <= 0 {
		return errors.New("pozitsiya tanlanmagan")
	}
	if userID <= 0 {
		return errors.New("foydalanuvchi aniqlanmadi")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	items, err := r.brigadirPendingItemsForUserTx(tx, int64(userID), itemID)
	if err != nil {
		return err
	}
	if len(items) == 0 {
		return errors.New("pozitsiya topilmadi yoki siz bu liniya mas'uli emassiz")
	}

	if err := r.brigadirCheckStockForItems(tx, items); err != nil {
		return err
	}

	if err := r.brigadirDeliveryItemConfirmTx(tx, items[0], userID); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repo) BrigadirDeliveryItemsConfirmAll(userID int) (int, error) {
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	items, err := r.brigadirPendingItemsForUserTx(tx, int64(userID), 0)
	if err != nil {
		return 0, err
	}
	if len(items) == 0 {
		return 0, errors.New("tasdiqlash uchun pozitsiyalar yo'q")
	}

	if err := r.brigadirCheckStockForItems(tx, items); err != nil {
		return 0, err
	}

	confirmedCount := 0
	for _, item := range items {
		if err := r.brigadirDeliveryItemConfirmTx(tx, item, userID); err != nil {
			return 0, err
		}
		confirmedCount++
	}

	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return confirmedCount, nil
}
