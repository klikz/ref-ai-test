package store

import "fmt"

type AuxiliaryReportSummaryRow struct {
	LineID        int     `json:"line_id"`
	LineName      string  `json:"line_name"`
	ComponentID   int     `json:"component_id"`
	FactoryCode   string  `json:"factory_code"`
	ComponentName string  `json:"component_name"`
	OdooCode      string  `json:"odoo_code"`
	Received      float64 `json:"received"`
	Expended      float64 `json:"expended"`
	BalanceEnd    float64 `json:"balance_end"`
}

type AuxiliaryReportDetailRow struct {
	ID             int     `json:"id"`
	LineID         int     `json:"line_id"`
	LineName       string  `json:"line_name"`
	FactoryCode    string  `json:"factory_code"`
	ComponentName  string  `json:"component_name"`
	OdooCode       string  `json:"odoo_code"`
	QuantityChange float64 `json:"quantity_change"`
	QuantityAfter  float64 `json:"quantity_after"`
	Time           string  `json:"time"`
	UserName       string  `json:"user_name"`
	Source         string  `json:"source"`
	Comment        string  `json:"comment"`
}

var auxiliaryReportLineIDs = []int{EshikLineID}

func effectiveAuxiliaryLineIDs(lineIDs []int) []int {
	if len(lineIDs) == 0 {
		return append([]int(nil), auxiliaryReportLineIDs...)
	}
	return lineIDs
}

const auxiliaryReportSummaryQuery = `
	WITH period_movements AS (
		SELECT bt.line_id,
			bt.component_id,
			COALESCE(SUM(CASE WHEN bt.quantity_change > 0 THEN bt.quantity_change ELSE 0 END), 0) AS received,
			COALESCE(SUM(CASE WHEN bt.quantity_change < 0 THEN -bt.quantity_change ELSE 0 END), 0) AS expended
		FROM lines.balance_transactions bt
		WHERE bt.created_at >= $1::timestamp
		  AND bt.created_at < $2::timestamp
		  AND bt.line_id = ANY($3)
		GROUP BY bt.line_id, bt.component_id
	),
	closing_balance AS (
		SELECT DISTINCT ON (bt.line_id, bt.component_id)
			bt.line_id,
			bt.component_id,
			bt.quantity_after AS balance_end
		FROM lines.balance_transactions bt
		WHERE bt.created_at < $2::timestamp
		  AND bt.line_id = ANY($3)
		ORDER BY bt.line_id, bt.component_id, bt.created_at DESC, bt.id DESC
	),
	all_keys AS (
		SELECT line_id, component_id FROM period_movements
		UNION
		SELECT line_id, component_id FROM closing_balance WHERE balance_end <> 0
	)
	SELECT k.line_id,
		ll.name,
		k.component_id,
		COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''),
		COALESCE(c.full_name_uz, c.standard_name_uz, ''),
		COALESCE(c.odoo_code, ''),
		COALESCE(pm.received, 0),
		COALESCE(pm.expended, 0),
		COALESCE(cb.balance_end, 0)
	FROM all_keys k
	INNER JOIN lines.lines_list ll ON ll.line_id = k.line_id
	INNER JOIN production.components c ON c.id = k.component_id
	LEFT JOIN period_movements pm ON pm.line_id = k.line_id AND pm.component_id = k.component_id
	LEFT JOIN closing_balance cb ON cb.line_id = k.line_id AND cb.component_id = k.component_id
	WHERE COALESCE(pm.received, 0) <> 0
	   OR COALESCE(pm.expended, 0) <> 0
	   OR COALESCE(cb.balance_end, 0) <> 0
	ORDER BY ll.name,
		COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''),
		COALESCE(c.full_name_uz, c.standard_name_uz, '')`

func (r *Repo) AuxiliaryReportSummary(dateFrom, dateTo string, lineIDs []int) ([]AuxiliaryReportSummaryRow, error) {
	from, to, err := NormalizeReportTimeRange(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	rows, err := r.store.db.Query(auxiliaryReportSummaryQuery, from, to, intSliceParam(effectiveAuxiliaryLineIDs(lineIDs)))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []AuxiliaryReportSummaryRow{}
	for rows.Next() {
		item := AuxiliaryReportSummaryRow{}
		if err := rows.Scan(
			&item.LineID, &item.LineName, &item.ComponentID,
			&item.FactoryCode, &item.ComponentName, &item.OdooCode,
			&item.Received, &item.Expended, &item.BalanceEnd,
		); err != nil {
			return items, err
		}
		item.Received = roundWareQuantity(item.Received)
		item.Expended = roundWareQuantity(item.Expended)
		item.BalanceEnd = roundWareQuantity(item.BalanceEnd)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}

func (r *Repo) AuxiliaryReportCount(dateFrom, dateTo string, lineIDs []int) (int, error) {
	from, to, err := NormalizeReportTimeRange(dateFrom, dateTo)
	if err != nil {
		return 0, err
	}
	ids := effectiveAuxiliaryLineIDs(lineIDs)
	// Eshik: one produced unit = freeze+ref pair. Count freeze print sessions.
	hasEshik := false
	otherIDs := make([]int, 0, len(ids))
	for _, id := range ids {
		if id == EshikLineID {
			hasEshik = true
			continue
		}
		otherIDs = append(otherIDs, id)
	}

	total := 0
	if hasEshik {
		var pairCount int
		err = r.store.db.QueryRow(`
			SELECT COUNT(*)::int
			FROM production.eshik_print_sessions s
			INNER JOIN production.eshik_model_parts p ON p.id = s.eshik_model_part_id
			WHERE s.c_time >= $1::timestamp
			  AND s.c_time < $2::timestamp
			  AND p.door_code = $3`,
			from, to, EshikDoorFreeze,
		).Scan(&pairCount)
		if err != nil {
			return 0, err
		}
		total += pairCount
	}
	if len(otherIDs) > 0 {
		var count int
		err = r.store.db.QueryRow(`
			SELECT COUNT(*)::int
			FROM lines.balance_transactions bt
			WHERE bt.created_at >= $1::timestamp
			  AND bt.created_at < $2::timestamp
			  AND bt.line_id = ANY($3)`,
			from, to, intSliceParam(otherIDs),
		).Scan(&count)
		if err != nil {
			return 0, err
		}
		total += count
	}
	return total, nil
}

const auxiliaryReportDetailQuery = `
	SELECT bt.id,
		bt.line_id,
		ll.name,
		COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''),
		COALESCE(c.full_name_uz, c.standard_name_uz, ''),
		COALESCE(c.odoo_code, ''),
		bt.quantity_change,
		bt.quantity_after,
		to_char(bt.created_at, 'YYYY-MM-DD HH24:MI'),
		COALESCE(NULLIF(u.name, ''), u.login, ''),
		bt.source,
		COALESCE(bt.comment, '')
	FROM lines.balance_transactions bt
	INNER JOIN lines.lines_list ll ON ll.line_id = bt.line_id
	INNER JOIN production.components c ON c.id = bt.component_id
	LEFT JOIN auth.users u ON u.id = bt.user_id
	WHERE bt.created_at >= $1::timestamp
	  AND bt.created_at < $2::timestamp
	  AND bt.line_id = ANY($3)
	ORDER BY bt.created_at DESC, bt.id DESC`

func (r *Repo) AuxiliaryReportDetail(dateFrom, dateTo string, lineIDs []int, limit, offset int) ([]AuxiliaryReportDetailRow, error) {
	from, to, err := NormalizeReportTimeRange(dateFrom, dateTo)
	if err != nil {
		return nil, err
	}
	query := auxiliaryReportDetailQuery
	args := []any{from, to, intSliceParam(effectiveAuxiliaryLineIDs(lineIDs))}
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)+1, len(args)+2)
		args = append(args, limit, offset)
	}

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []AuxiliaryReportDetailRow{}
	for rows.Next() {
		item := AuxiliaryReportDetailRow{}
		if err := rows.Scan(
			&item.ID, &item.LineID, &item.LineName,
			&item.FactoryCode, &item.ComponentName, &item.OdooCode,
			&item.QuantityChange, &item.QuantityAfter,
			&item.Time, &item.UserName, &item.Source, &item.Comment,
		); err != nil {
			return items, err
		}
		item.QuantityChange = roundWareQuantity(item.QuantityChange)
		item.QuantityAfter = roundWareQuantity(item.QuantityAfter)
		items = append(items, item)
	}
	if err = rows.Err(); err != nil {
		return items, err
	}
	return items, nil
}
