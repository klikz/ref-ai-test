package store

import (
	"strings"
)

type WareGPProductBalanceRow struct {
	ID         int64  `json:"id"`
	ProductID  int    `json:"product_id"`
	Serial     string `json:"serial"`
	ModelID    int    `json:"model_id"`
	Modeli     string `json:"modeli"`
	ModelName  string `json:"model_name"`
	LineID     int    `json:"line_id"`
	LineName   string `json:"line_name"`
	UserID     int    `json:"user_id"`
	UserName   string `json:"user_name"`
	ReceivedAt string `json:"received_at"`
}

type WareGPProductBalanceSummaryRow struct {
	ModelID   int    `json:"model_id"`
	Modeli    string `json:"modeli"`
	ModelName string `json:"model_name"`
	Count     int    `json:"count"`
}

func (r *Repo) WareGPProductBalanceLast(limit int) ([]WareGPProductBalanceRow, error) {
	if limit <= 0 {
		limit = 10
	}
	if limit > 200 {
		limit = 200
	}

	rows, err := r.store.db.Query(`
		SELECT b.id, b.product_id, b.serial, b.model_id,
			COALESCE(m.modeli, ''),
			COALESCE(m.qisqa_nomi, ''),
			b.line_id,
			COALESCE(ll.name, ''),
			COALESCE(b.user_id, 0),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(b.received_at, 'YYYY-MM-DD HH24:MI:SS')
		FROM ware.gp_product_balance b
		LEFT JOIN production.models m ON m.id = b.model_id
		LEFT JOIN lines.lines_list ll ON ll.line_id = b.line_id
		LEFT JOIN auth.users u ON u.id = b.user_id
		ORDER BY b.received_at DESC, b.id DESC
		LIMIT $1`,
		limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareGPProductBalanceRow{}
	for rows.Next() {
		row := WareGPProductBalanceRow{}
		if err := rows.Scan(
			&row.ID,
			&row.ProductID,
			&row.Serial,
			&row.ModelID,
			&row.Modeli,
			&row.ModelName,
			&row.LineID,
			&row.LineName,
			&row.UserID,
			&row.UserName,
			&row.ReceivedAt,
		); err != nil {
			return items, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func (r *Repo) WareGPProductBalanceHistory(limit int) ([]WareGPProductBalanceRow, error) {
	if limit <= 0 {
		limit = 100
	}
	if limit > 1000 {
		limit = 1000
	}
	return r.WareGPProductBalanceLast(limit)
}

func (r *Repo) WareGPProductBalanceSummary() ([]WareGPProductBalanceSummaryRow, error) {
	rows, err := r.store.db.Query(`
		SELECT b.model_id,
			COALESCE(m.modeli, ''),
			COALESCE(m.qisqa_nomi, ''),
			COUNT(*)::int
		FROM ware.gp_product_balance b
		LEFT JOIN production.models m ON m.id = b.model_id
		GROUP BY b.model_id, m.modeli, m.qisqa_nomi
		ORDER BY m.modeli, m.qisqa_nomi`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareGPProductBalanceSummaryRow{}
	for rows.Next() {
		row := WareGPProductBalanceSummaryRow{}
		if err := rows.Scan(&row.ModelID, &row.Modeli, &row.ModelName, &row.Count); err != nil {
			return items, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}

func (r *Repo) WareGPProductTransactions(dateFrom, dateTo string, limit int) ([]WareGPProductBalanceRow, error) {
	if limit <= 0 {
		limit = 500
	}
	if limit > 2000 {
		limit = 2000
	}

	rows, err := r.store.db.Query(`
		SELECT b.id, b.product_id, b.serial, b.model_id,
			COALESCE(m.modeli, ''),
			COALESCE(m.qisqa_nomi, ''),
			b.line_id,
			COALESCE(ll.name, ''),
			COALESCE(b.user_id, 0),
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(b.received_at, 'YYYY-MM-DD HH24:MI:SS')
		FROM ware.gp_product_balance b
		LEFT JOIN production.models m ON m.id = b.model_id
		LEFT JOIN lines.lines_list ll ON ll.line_id = b.line_id
		LEFT JOIN auth.users u ON u.id = b.user_id
		WHERE b.received_at >= COALESCE(NULLIF($1, '')::date, CURRENT_DATE)
		  AND b.received_at < COALESCE(NULLIF($2, '')::date, CURRENT_DATE) + INTERVAL '1 day'
		ORDER BY b.received_at DESC, b.id DESC
		LIMIT $3`,
		strings.TrimSpace(dateFrom), strings.TrimSpace(dateTo), limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WareGPProductBalanceRow{}
	for rows.Next() {
		row := WareGPProductBalanceRow{}
		if err := rows.Scan(
			&row.ID,
			&row.ProductID,
			&row.Serial,
			&row.ModelID,
			&row.Modeli,
			&row.ModelName,
			&row.LineID,
			&row.LineName,
			&row.UserID,
			&row.UserName,
			&row.ReceivedAt,
		); err != nil {
			return items, err
		}
		items = append(items, row)
	}
	return items, rows.Err()
}
