package store

import (
	"database/sql"
	"errors"
	"fmt"
)

type BalanceChangeParams struct {
	LineID         int
	ComponentID    int
	QuantityChange float64
	UserID         int
	Source         string
	Comment        string
}

type LinesBalanceTransaction struct {
	ID               int     `json:"id"`
	LineID           int     `json:"line_id"`
	LineName         string  `json:"line_name"`
	ComponentID      int     `json:"component_id"`
	ManufacturerCode string  `json:"manufacturer_code"`
	StandardNameUz   string  `json:"standard_name_uz"`
	OdooCode         string  `json:"odoo_code"`
	QuantityChange   float64 `json:"quantity_change"`
	QuantityAfter    float64 `json:"quantity_after"`
	UserID           int     `json:"user_id"`
	UserName         string  `json:"user_name"`
	UserLogin        string  `json:"user_login"`
	Source           string  `json:"source"`
	Comment          string  `json:"comment"`
	CreatedAt        string  `json:"created_at"`
}

type BalanceTransactionFilter struct {
	LineID      int
	ComponentID int
	DateFrom    string
	DateTo      string
	Limit       int
}

func (r *Repo) LinesBalanceApplyChange(p BalanceChangeParams) (float64, error) {
	if p.LineID <= 0 || p.ComponentID <= 0 {
		return 0, errors.New("liniya yoki komponent tanlanmagan")
	}
	p.QuantityChange = roundWareQuantity(p.QuantityChange)
	if p.QuantityChange == 0 {
		return 0, errors.New("miqdor 0 bo'lishi mumkin emas")
	}
	if p.UserID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}
	if p.Source == "" {
		p.Source = "manual"
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	quantityAfter, err := r.linesBalanceApplyChangeTx(tx, p)
	if err != nil {
		return 0, err
	}

	if err = tx.Commit(); err != nil {
		return 0, err
	}
	return quantityAfter, nil
}

func (r *Repo) linesBalanceApplyChangeTx(tx *sql.Tx, p BalanceChangeParams) (float64, error) {
	current := 0.0
	err := tx.QueryRow(`
		SELECT quantity
		FROM lines.balance
		WHERE line_id = $1 AND component_id = $2`,
		p.LineID, p.ComponentID,
	).Scan(&current)
	if err != nil {
		if err == sql.ErrNoRows {
			current = 0
		} else {
			return 0, err
		}
	}
	current = roundWareQuantity(current)
	p.QuantityChange = roundWareQuantity(p.QuantityChange)

	quantityAfter := roundWareQuantity(current + p.QuantityChange)
	if quantityAfter < 0 {
		return 0, fmt.Errorf("balans yetarli emas (joriy: %.4f)", current)
	}

	_, err = tx.Exec(`
		INSERT INTO lines.balance (line_id, component_id, quantity)
		VALUES ($1, $2, $3)
		ON CONFLICT (line_id, component_id)
		DO UPDATE SET quantity = EXCLUDED.quantity`,
		p.LineID, p.ComponentID, quantityAfter,
	)
	if err != nil {
		return 0, err
	}

	_, err = tx.Exec(`
		INSERT INTO lines.balance_transactions
			(line_id, component_id, quantity_change, quantity_after, user_id, source, comment)
		VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		p.LineID, p.ComponentID, p.QuantityChange, quantityAfter, p.UserID, p.Source, p.Comment,
	)
	if err != nil {
		return 0, err
	}

	return quantityAfter, nil
}

func (r *Repo) LinesBalanceQuantity(lineID, componentID int) (float64, error) {
	if lineID <= 0 || componentID <= 0 {
		return 0, errors.New("liniya yoki komponent tanlanmagan")
	}

	var quantity sql.NullFloat64
	err := r.store.db.QueryRow(`
		SELECT quantity
		FROM lines.balance
		WHERE line_id = $1 AND component_id = $2`,
		lineID, componentID,
	).Scan(&quantity)
	if err != nil {
		if err == sql.ErrNoRows {
			return 0, nil
		}
		return 0, err
	}
	if !quantity.Valid {
		return 0, nil
	}
	return roundWareQuantity(quantity.Float64), nil
}

func (r *Repo) LinesBalanceTransactionsGet(filter BalanceTransactionFilter) ([]LinesBalanceTransaction, error) {
	query := `
		SELECT bt.id, bt.line_id, ll.name, bt.component_id,
			COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''), c.standard_name_uz,
			COALESCE(c.odoo_code, ''),
			bt.quantity_change, bt.quantity_after, bt.user_id, COALESCE(u.name, ''), COALESCE(u.login, ''),
			bt.source, bt.comment, to_char(bt.created_at, 'YYYY-MM-DD HH24:MI:SS')
		FROM lines.balance_transactions bt
		INNER JOIN lines.lines_list ll ON ll.line_id = bt.line_id
		INNER JOIN production.components c ON c.id = bt.component_id
		LEFT JOIN auth.users u ON u.id = bt.user_id
		WHERE 1 = 1`

	args := []any{}
	argPos := 1

	if filter.LineID > 0 {
		query += fmt.Sprintf(" AND bt.line_id = $%d", argPos)
		args = append(args, filter.LineID)
		argPos++
	}
	if filter.ComponentID > 0 {
		query += fmt.Sprintf(" AND bt.component_id = $%d", argPos)
		args = append(args, filter.ComponentID)
		argPos++
	}
	if filter.DateFrom != "" {
		query += fmt.Sprintf(" AND bt.created_at >= $%d::date", argPos)
		args = append(args, filter.DateFrom)
		argPos++
	}
	if filter.DateTo != "" {
		query += fmt.Sprintf(" AND bt.created_at < ($%d::date + INTERVAL '1 day')", argPos)
		args = append(args, filter.DateTo)
		argPos++
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 5000
	}
	query += fmt.Sprintf(" ORDER BY bt.created_at DESC, bt.id DESC LIMIT $%d", argPos)
	args = append(args, limit)

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []LinesBalanceTransaction{}
	for rows.Next() {
		item := LinesBalanceTransaction{}
		if err := rows.Scan(
			&item.ID,
			&item.LineID,
			&item.LineName,
			&item.ComponentID,
			&item.ManufacturerCode,
			&item.StandardNameUz,
			&item.OdooCode,
			&item.QuantityChange,
			&item.QuantityAfter,
			&item.UserID,
			&item.UserName,
			&item.UserLogin,
			&item.Source,
			&item.Comment,
			&item.CreatedAt,
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
