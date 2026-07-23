package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	T1LineID = 4
	T2LineID = 5
	T3LineID = 6

	ProductStatusActive      = "active"
	ProductStatusTransferred = "transferred"
	ProductStatusWrittenOff  = "written_off"
)

type ProductLineTransferResult struct {
	FromProductID   int  `json:"from_product_id"`
	ToProductID     int  `json:"to_product_id"`
	fromWasActive   bool `json:"-"`
	toCreatedNewRow bool `json:"-"`
}

type ProductBalanceItem struct {
	ID        int    `json:"id"`
	Serial    string `json:"serial"`
	ModelID   int    `json:"model_id"`
	ModelName string `json:"model_name"`
	Modeli    string `json:"modeli"`
	OdooCode  string `json:"odoo_code"`
	Time      string `json:"time"`
}

type ProductBalanceSummaryRow struct {
	ModelID   int    `json:"model_id"`
	ModelName string `json:"model_name"`
	Modeli    string `json:"modeli"`
	OdooCode  string `json:"odoo_code"`
	Count     int    `json:"count"`
}

type ProductTransferReportRow struct {
	ID           int    `json:"id"`
	Serial       string `json:"serial"`
	LineID       int    `json:"line_id"`
	LineName     string `json:"line_name"`
	ModelID      int    `json:"model_id"`
	ModelName    string `json:"model_name"`
	Modeli       string `json:"modeli"`
	OdooCode     string `json:"odoo_code"`
	TransferredAt string `json:"transferred_at"`
	UserName     string `json:"user_name"`
}

func (r *Repo) productOnLineTx(tx *sql.Tx, lineID int, serial string) (id int, status string, err error) {
	serial = strings.TrimSpace(serial)
	if lineID <= 0 || serial == "" {
		return 0, "", errors.New("serial yoki liniya noto'g'ri")
	}

	err = tx.QueryRow(`
		SELECT id, status
		FROM lines.products
		WHERE line_id = $1 AND serial = $2
		FOR UPDATE`,
		lineID, serial,
	).Scan(&id, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, "", fmt.Errorf("serial %s liniyada topilmadi (liniya %d)", serial, lineID)
		}
		return 0, "", err
	}
	return id, status, nil
}

func (r *Repo) productLineHasSerialTx(tx *sql.Tx, lineID int, serial string) (bool, error) {
	var id int
	err := tx.QueryRow(`
		SELECT id
		FROM lines.products
		WHERE line_id = $1 AND serial = $2
		LIMIT 1`,
		lineID, strings.TrimSpace(serial),
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

type ProductActiveInfo struct {
	ID        int
	LineID    int
	ModelID   int
	Serial    string
	AccSerial string
}

func (r *Repo) ProductFindActiveBySerial(serial string) (ProductActiveInfo, error) {
	out := ProductActiveInfo{}
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return out, errors.New("serial bo'sh")
	}
	err := r.store.db.QueryRow(`
		SELECT id, line_id, COALESCE(model_id, 0), serial, COALESCE(acc_serial, '')
		FROM lines.products
		WHERE serial = $1 AND status = $2
		ORDER BY id DESC
		LIMIT 1`,
		serial, ProductStatusActive,
	).Scan(&out.ID, &out.LineID, &out.ModelID, &out.Serial, &out.AccSerial)
	if errors.Is(err, sql.ErrNoRows) {
		return out, fmt.Errorf("serial %s topilmadi (faol mahsulot yo'q)", serial)
	}
	return out, err
}

func (r *Repo) ProductUpdateAccSerial(productID int, accSerial string) error {
	if productID <= 0 {
		return errors.New("mahsulot id noto'g'ri")
	}
	_, err := r.store.db.Exec(`
		UPDATE lines.products
		SET acc_serial = $2
		WHERE id = $1`,
		productID, strings.TrimSpace(accSerial),
	)
	return err
}

func (r *Repo) ProductLineTransfer(
	fromLineID, toLineID, componentID, userID, modelID int,
	serial, accSerial string,
) (ProductLineTransferResult, error) {
	result := ProductLineTransferResult{}
	serial = strings.TrimSpace(serial)
	accSerial = strings.TrimSpace(accSerial)

	if fromLineID <= 0 || toLineID <= 0 {
		return result, errors.New("liniya noto'g'ri")
	}
	if userID <= 0 {
		return result, errors.New("foydalanuvchi aniqlanmadi")
	}
	if modelID <= 0 {
		return result, errors.New("model noto'g'ri")
	}
	if serial == "" {
		return result, errors.New("serial bo'sh")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return result, err
	}
	defer tx.Rollback()

	fromID, fromStatus, err := r.productOnLineTx(tx, fromLineID, serial)
	if err != nil {
		return result, err
	}
	fromWasActive := fromStatus == ProductStatusActive

	var toID int
	toCreatedNewRow := false

	toExists, err := r.productLineHasSerialTx(tx, toLineID, serial)
	if err != nil {
		return result, err
	}
	if toExists {
		var toStatus string
		toID, toStatus, err = r.productOnLineTx(tx, toLineID, serial)
		if err != nil {
			return result, err
		}
		if toStatus == ProductStatusActive {
			return result, fmt.Errorf("serial %s allaqachon %d-liniyada mavjud", serial, toLineID)
		}
	}

	if _, err = tx.Exec(`
		UPDATE lines.products
		SET status = $2, transferred_at = COALESCE(transferred_at, NOW())
		WHERE id = $1`,
		fromID, ProductStatusTransferred,
	); err != nil {
		return result, err
	}

	if toExists {
		var res sql.Result
		if accSerial != "" {
			res, err = tx.Exec(`
				UPDATE lines.products
				SET status = $1, transferred_at = NULL, acc_serial = $3
				WHERE id = $2`,
				ProductStatusActive, toID, accSerial,
			)
		} else {
			res, err = tx.Exec(`
				UPDATE lines.products
				SET status = $1, transferred_at = NULL
				WHERE id = $2`,
				ProductStatusActive, toID,
			)
		}
		if err != nil {
			return result, err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return result, err
		}
		if affected == 0 {
			return result, errors.New("mahsulot holati yangilanmadi")
		}
	} else {
		toID, err = r.linesAddProductTx(tx, toLineID, componentID, userID, modelID, serial, accSerial)
		if err != nil {
			return result, err
		}
		toCreatedNewRow = true
	}

	if err = tx.Commit(); err != nil {
		return result, err
	}

	result.FromProductID = fromID
	result.ToProductID = toID
	result.fromWasActive = fromWasActive
	result.toCreatedNewRow = toCreatedNewRow
	return result, nil
}

func (r *Repo) ProductLineTransferRollback(transfer ProductLineTransferResult) error {
	if transfer.FromProductID <= 0 || transfer.ToProductID <= 0 {
		return errors.New("mahsulot id noto'g'ri")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if transfer.toCreatedNewRow {
		if _, err = tx.Exec(`DELETE FROM lines.products WHERE id = $1`, transfer.ToProductID); err != nil {
			return err
		}
	} else {
		if _, err = tx.Exec(`
			UPDATE lines.products
			SET status = $1, transferred_at = COALESCE(transferred_at, NOW())
			WHERE id = $2`,
			ProductStatusTransferred, transfer.ToProductID,
		); err != nil {
			return err
		}
	}

	if transfer.fromWasActive {
		res, err := tx.Exec(`
			UPDATE lines.products
			SET status = $2, transferred_at = NULL
			WHERE id = $1 AND status = $3`,
			transfer.FromProductID, ProductStatusActive, ProductStatusTransferred,
		)
		if err != nil {
			return err
		}
		affected, err := res.RowsAffected()
		if err != nil {
			return err
		}
		if affected == 0 {
			return errors.New("oldingi liniya yozuvi tiklanmadi")
		}
	}

	return tx.Commit()
}

func (r *Repo) ProductBalanceByLine(lineID int) ([]ProductBalanceItem, error) {
	if lineID <= 0 {
		return nil, errors.New("liniya tanlanmagan")
	}

	rows, err := r.store.db.Query(`
		SELECT p.id,
			p.serial,
			p.model_id,
			COALESCE(m.qisqa_nomi, ''),
			COALESCE(m.modeli, ''),
			COALESCE(m.odoo_code, ''),
			to_char(p."time", 'YYYY-MM-DD HH24:MI')
		FROM lines.products p
		LEFT JOIN production.models m ON m.id = p.model_id
		WHERE p.line_id = $1 AND p.status = $2
		ORDER BY p."time" DESC, p.id DESC`,
		lineID, ProductStatusActive,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ProductBalanceItem{}
	for rows.Next() {
		item := ProductBalanceItem{}
		if err := rows.Scan(&item.ID, &item.Serial, &item.ModelID, &item.ModelName, &item.Modeli, &item.OdooCode, &item.Time); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) ProductBalanceSummary(lineID int) ([]ProductBalanceSummaryRow, error) {
	if lineID <= 0 {
		return nil, errors.New("liniya tanlanmagan")
	}

	rows, err := r.store.db.Query(`
		SELECT p.model_id,
			COALESCE(m.qisqa_nomi, ''),
			COALESCE(m.modeli, ''),
			COALESCE(m.odoo_code, ''),
			COUNT(*)::int
		FROM lines.products p
		LEFT JOIN production.models m ON m.id = p.model_id
		WHERE p.line_id = $1 AND p.status = $2
		GROUP BY p.model_id, m.qisqa_nomi, m.modeli, m.odoo_code
		ORDER BY m.modeli, m.qisqa_nomi`,
		lineID, ProductStatusActive,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ProductBalanceSummaryRow{}
	for rows.Next() {
		item := ProductBalanceSummaryRow{}
		if err := rows.Scan(&item.ModelID, &item.ModelName, &item.Modeli, &item.OdooCode, &item.Count); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) ProductTransferReport(lineID int, dateFrom, dateTo string) ([]ProductTransferReportRow, error) {
	query := `
		SELECT p.id,
			p.serial,
			p.line_id,
			COALESCE(ll.name, ''),
			p.model_id,
			COALESCE(m.qisqa_nomi, ''),
			COALESCE(m.modeli, ''),
			COALESCE(m.odoo_code, ''),
			to_char(p.transferred_at, 'YYYY-MM-DD HH24:MI'),
			COALESCE(NULLIF(u.name, ''), u.login, '')
		FROM lines.products p
		LEFT JOIN lines.lines_list ll ON ll.line_id = p.line_id
		LEFT JOIN production.models m ON m.id = p.model_id
		LEFT JOIN auth.users u ON u.id = p.user_id
		WHERE p.status = $1
		  AND p.transferred_at IS NOT NULL`

	args := []any{ProductStatusTransferred}
	argN := 2

	if lineID > 0 {
		query += fmt.Sprintf(" AND p.line_id = $%d", argN)
		args = append(args, lineID)
		argN++
	}
	if strings.TrimSpace(dateFrom) != "" {
		query += fmt.Sprintf(" AND p.transferred_at >= COALESCE(NULLIF($%d, '')::date, CURRENT_DATE)", argN)
		args = append(args, dateFrom)
		argN++
	}
	if strings.TrimSpace(dateTo) != "" {
		query += fmt.Sprintf(" AND p.transferred_at < COALESCE(NULLIF($%d, '')::date, CURRENT_DATE) + INTERVAL '1 day'", argN)
		args = append(args, dateTo)
		argN++
	}

	query += ` ORDER BY p.transferred_at DESC, p.id DESC`

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []ProductTransferReportRow{}
	for rows.Next() {
		item := ProductTransferReportRow{}
		if err := rows.Scan(
			&item.ID, &item.Serial, &item.LineID, &item.LineName,
			&item.ModelID, &item.ModelName, &item.Modeli, &item.OdooCode,
			&item.TransferredAt, &item.UserName,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// BackfillProductLineStatus — bir martalik: mavjud seriallar bo'yicha T1↔T2, T2↔T3 balansni moslashtirish.
func (r *Repo) BackfillProductLineStatus() (t1Updated, t2Updated int64, err error) {
	res, err := r.store.db.Exec(`
		UPDATE lines.products p1
		SET status = $1, transferred_at = COALESCE(p1.transferred_at, NOW())
		WHERE p1.line_id = $2
		  AND p1.status = $3
		  AND EXISTS (
		      SELECT 1 FROM lines.products p2
		      WHERE p2.line_id = $4 AND p2.serial = p1.serial
		  )`,
		ProductStatusTransferred, T1LineID, ProductStatusActive, T2LineID,
	)
	if err != nil {
		return 0, 0, err
	}
	t1Updated, _ = res.RowsAffected()

	res, err = r.store.db.Exec(`
		UPDATE lines.products p2
		SET status = $1, transferred_at = COALESCE(p2.transferred_at, NOW())
		WHERE p2.line_id = $2
		  AND p2.status = $3
		  AND EXISTS (
		      SELECT 1 FROM lines.products p3
		      WHERE p3.line_id = $4 AND p3.serial = p2.serial
		  )`,
		ProductStatusTransferred, T2LineID, ProductStatusActive, T3LineID,
	)
	if err != nil {
		return t1Updated, 0, err
	}
	t2Updated, _ = res.RowsAffected()
	return t1Updated, t2Updated, nil
}
