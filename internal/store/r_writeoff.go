package store

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
)

const (
	WriteoffStatusDraft    = "draft"
	WriteoffStatusPending  = "pending"
	WriteoffStatusApproved = "approved"
	WriteoffStatusRejected = "rejected"

	WriteoffItemComponent = "component"
	WriteoffItemProduct   = "product"
)

var WriteoffProductLineIDs = []int{1}
var WriteoffAuxLineIDs = []int{11}

func IsWriteoffProductLine(lineID int) bool {
	for _, id := range WriteoffProductLineIDs {
		if lineID == id {
			return true
		}
	}
	return false
}

func IsWriteoffAuxLine(lineID int) bool {
	for _, id := range WriteoffAuxLineIDs {
		if lineID == id {
			return true
		}
	}
	return false
}

type WriteoffResponsible struct {
	ID        int64  `json:"id"`
	UserID    int    `json:"user_id"`
	UserName  string `json:"user_name"`
	UserLogin string `json:"user_login"`
	CTime     string `json:"c_time"`
}

type WriteoffDocument struct {
	ID            int64  `json:"id"`
	Status        string `json:"status"`
	CreatedBy     int    `json:"created_by"`
	CreatedByName string `json:"created_by_name"`
	SubmittedAt   string `json:"submitted_at"`
	ApprovedBy    int    `json:"approved_by"`
	ApprovedByName string `json:"approved_by_name"`
	ApprovedAt    string `json:"approved_at"`
	RejectedBy    int    `json:"rejected_by"`
	RejectedByName string `json:"rejected_by_name"`
	RejectedAt    string `json:"rejected_at"`
	RejectComment string `json:"reject_comment"`
	CTime         string `json:"c_time"`
	ItemCount     int    `json:"item_count"`
	ApprovalRequired int `json:"approval_required"`
	ApprovalDone     int `json:"approval_done"`
}

type WriteoffDocumentApproval struct {
	ID         int64  `json:"id"`
	DocumentID int64  `json:"document_id"`
	UserID     int    `json:"user_id"`
	UserName   string `json:"user_name"`
	UserLogin  string `json:"user_login"`
	ApprovedAt string `json:"approved_at"`
	IsApproved bool   `json:"is_approved"`
}

type WriteoffApproveResult struct {
	AllApproved      bool `json:"all_approved"`
	ApprovalDone     int  `json:"approval_done"`
	ApprovalRequired int  `json:"approval_required"`
}

type WriteoffDocumentItem struct {
	ID          int64   `json:"id"`
	DocumentID  int64   `json:"document_id"`
	LineID      int     `json:"line_id"`
	LineName    string  `json:"line_name"`
	ItemType    string  `json:"item_type"`
	ModelID     int     `json:"model_id"`
	ComponentID int     `json:"component_id"`
	ProductID   int     `json:"product_id"`
	Serial      string  `json:"serial"`
	ItemLabel   string  `json:"item_label"`
	Quantity    float64 `json:"quantity"`
	Comment     string  `json:"comment"`
	SortOrder   int     `json:"sort_order"`
}

type WriteoffRecord struct {
	ID            int64   `json:"id"`
	DocumentID    int64   `json:"document_id"`
	LineID        int     `json:"line_id"`
	LineName      string  `json:"line_name"`
	ItemType      string  `json:"item_type"`
	ModelID       int     `json:"model_id"`
	ComponentID   int     `json:"component_id"`
	ProductID     int     `json:"product_id"`
	Serial        string  `json:"serial"`
	ItemLabel     string  `json:"item_label"`
	Quantity      float64 `json:"quantity"`
	Comment       string  `json:"comment"`
	BalanceBefore float64 `json:"balance_before"`
	BalanceAfter  float64 `json:"balance_after"`
	WrittenOffBy  int     `json:"written_off_by"`
	WrittenOffByName string `json:"written_off_by_name"`
	WrittenOffAt  string  `json:"written_off_at"`
}

type WriteoffCatalogItem struct {
	ItemID    int    `json:"item_id"`
	ItemType  string `json:"item_type"`
	Label     string `json:"label"`
	LineID    int    `json:"line_id"`
	LineName  string `json:"line_name"`
}

type WriteoffImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

type WriteoffSaveItemInput struct {
	ID          int64
	LineID      int
	ItemType    string
	ModelID     int
	ComponentID int
	Serial      string
	Quantity    float64
	Comment     string
	SortOrder   int
}

func (r *Repo) WriteoffIsResponsible(userID int) (bool, error) {
	if userID <= 0 {
		return false, nil
	}
	var ok int
	err := r.store.db.QueryRow(`
		SELECT 1 FROM writeoff.responsibles WHERE user_id = $1 LIMIT 1`, userID,
	).Scan(&ok)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}

func (r *Repo) WriteoffResponsiblesGetAll() ([]WriteoffResponsible, error) {
	rows, err := r.store.db.Query(`
		SELECT wr.id, wr.user_id,
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			COALESCE(u.login, ''),
			to_char(wr.c_time, 'YYYY-MM-DD HH24:MI:SS')
		FROM writeoff.responsibles wr
		INNER JOIN auth.users u ON u.id = wr.user_id
		WHERE u.status = true
		ORDER BY u.login`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffResponsible{}
	for rows.Next() {
		item := WriteoffResponsible{}
		if err := rows.Scan(&item.ID, &item.UserID, &item.UserName, &item.UserLogin, &item.CTime); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) WriteoffResponsibleAdd(userID, cUserID int) error {
	if userID <= 0 {
		return errors.New("foydalanuvchi tanlanmagan")
	}
	if cUserID <= 0 {
		return errors.New("foydalanuvchi aniqlanmadi")
	}
	var userOK int
	err := r.store.db.QueryRow(`SELECT 1 FROM auth.users WHERE id = $1 AND status = true`, userID).Scan(&userOK)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("foydalanuvchi topilmadi")
		}
		return err
	}
	_, err = r.store.db.Exec(`
		INSERT INTO writeoff.responsibles (user_id, c_user_id) VALUES ($1, $2)`,
		userID, cUserID,
	)
	if err != nil {
		if strings.Contains(err.Error(), "writeoff_responsibles_user_un") ||
			strings.Contains(err.Error(), "duplicate key") {
			return errors.New("bu tayinlash allaqachon mavjud")
		}
		return err
	}
	return nil
}

func (r *Repo) WriteoffResponsibleDelete(id int64) error {
	if id <= 0 {
		return errors.New("yozuv topilmadi")
	}
	res, err := r.store.db.Exec(`DELETE FROM writeoff.responsibles WHERE id = $1`, id)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("yozuv topilmadi")
	}
	return nil
}

func (r *Repo) WriteoffDocumentsList(status string, limit int) ([]WriteoffDocument, error) {
	if limit <= 0 || limit > 500 {
		limit = 200
	}
	status = strings.TrimSpace(status)

	query := `
		SELECT d.id, d.status, d.created_by,
			COALESCE(NULLIF(uc.name, ''), uc.login, ''),
			COALESCE(to_char(d.submitted_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(d.approved_by, 0),
			COALESCE(NULLIF(ua.name, ''), ua.login, ''),
			COALESCE(to_char(d.approved_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(d.rejected_by, 0),
			COALESCE(NULLIF(ur.name, ''), ur.login, ''),
			COALESCE(to_char(d.rejected_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(d.reject_comment, ''),
			to_char(d.c_time, 'YYYY-MM-DD HH24:MI:SS'),
			(SELECT COUNT(*)::int FROM writeoff.document_items di WHERE di.document_id = d.id),
			(SELECT COUNT(*)::int FROM writeoff.document_approvals da WHERE da.document_id = d.id),
			(SELECT COUNT(*)::int FROM writeoff.document_approvals da WHERE da.document_id = d.id AND da.approved_at IS NOT NULL)
		FROM writeoff.documents d
		INNER JOIN auth.users uc ON uc.id = d.created_by
		LEFT JOIN auth.users ua ON ua.id = d.approved_by
		LEFT JOIN auth.users ur ON ur.id = d.rejected_by
		WHERE 1=1`
	args := []any{}
	if status != "" {
		query += ` AND d.status = $1`
		args = append(args, status)
	}
	query += ` ORDER BY d.c_time DESC LIMIT ` + fmt.Sprintf("%d", limit)

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffDocument{}
	for rows.Next() {
		item := WriteoffDocument{}
		if err := rows.Scan(
			&item.ID, &item.Status, &item.CreatedBy, &item.CreatedByName,
			&item.SubmittedAt, &item.ApprovedBy, &item.ApprovedByName, &item.ApprovedAt,
			&item.RejectedBy, &item.RejectedByName, &item.RejectedAt, &item.RejectComment,
			&item.CTime, &item.ItemCount, &item.ApprovalRequired, &item.ApprovalDone,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) WriteoffDocumentCreate(userID int) (int64, error) {
	if userID <= 0 {
		return 0, errors.New("foydalanuvchi aniqlanmadi")
	}
	var id int64
	err := r.store.db.QueryRow(`
		INSERT INTO writeoff.documents (status, created_by)
		VALUES ($1, $2) RETURNING id`,
		WriteoffStatusDraft, userID,
	).Scan(&id)
	return id, err
}

func (r *Repo) WriteoffDocumentDelete(documentID int64) error {
	res, err := r.store.db.Exec(`
		DELETE FROM writeoff.documents WHERE id = $1 AND status IN ('draft', 'rejected')`,
		documentID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("hujjat topilmadi yoki o'chirish mumkin emas")
	}
	return nil
}

func (r *Repo) writeoffDocumentStatus(documentID int64) (string, error) {
	var status string
	err := r.store.db.QueryRow(`SELECT status FROM writeoff.documents WHERE id = $1`, documentID).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", errors.New("hujjat topilmadi")
		}
		return "", err
	}
	return status, nil
}

func (r *Repo) writeoffDocumentApprovalsGet(documentID int64) ([]WriteoffDocumentApproval, error) {
	rows, err := r.store.db.Query(`
		SELECT da.id, da.document_id, da.user_id,
			COALESCE(NULLIF(u.name, ''), u.login, ''),
			COALESCE(u.login, ''),
			COALESCE(to_char(da.approved_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			(da.approved_at IS NOT NULL)
		FROM writeoff.document_approvals da
		INNER JOIN auth.users u ON u.id = da.user_id
		WHERE da.document_id = $1
		ORDER BY u.login, da.id`, documentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffDocumentApproval{}
	for rows.Next() {
		item := WriteoffDocumentApproval{}
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.UserID, &item.UserName, &item.UserLogin,
			&item.ApprovedAt, &item.IsApproved,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) WriteoffDocumentGet(documentID int64) (*WriteoffDocument, []WriteoffDocumentItem, []WriteoffDocumentApproval, error) {
	docs, err := r.WriteoffDocumentsList("", 1)
	_ = docs
	row := WriteoffDocument{}
	err = r.store.db.QueryRow(`
		SELECT d.id, d.status, d.created_by,
			COALESCE(NULLIF(uc.name, ''), uc.login, ''),
			COALESCE(to_char(d.submitted_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(d.approved_by, 0),
			COALESCE(NULLIF(ua.name, ''), ua.login, ''),
			COALESCE(to_char(d.approved_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(d.rejected_by, 0),
			COALESCE(NULLIF(ur.name, ''), ur.login, ''),
			COALESCE(to_char(d.rejected_at, 'YYYY-MM-DD HH24:MI:SS'), ''),
			COALESCE(d.reject_comment, ''),
			to_char(d.c_time, 'YYYY-MM-DD HH24:MI:SS'),
			(SELECT COUNT(*)::int FROM writeoff.document_items di WHERE di.document_id = d.id),
			(SELECT COUNT(*)::int FROM writeoff.document_approvals da WHERE da.document_id = d.id),
			(SELECT COUNT(*)::int FROM writeoff.document_approvals da WHERE da.document_id = d.id AND da.approved_at IS NOT NULL)
		FROM writeoff.documents d
		INNER JOIN auth.users uc ON uc.id = d.created_by
		LEFT JOIN auth.users ua ON ua.id = d.approved_by
		LEFT JOIN auth.users ur ON ur.id = d.rejected_by
		WHERE d.id = $1`, documentID,
	).Scan(
		&row.ID, &row.Status, &row.CreatedBy, &row.CreatedByName,
		&row.SubmittedAt, &row.ApprovedBy, &row.ApprovedByName, &row.ApprovedAt,
		&row.RejectedBy, &row.RejectedByName, &row.RejectedAt, &row.RejectComment,
		&row.CTime, &row.ItemCount, &row.ApprovalRequired, &row.ApprovalDone,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil, nil, errors.New("hujjat topilmadi")
		}
		return nil, nil, nil, err
	}

	items, err := r.writeoffDocumentItemsGet(documentID)
	if err != nil {
		return nil, nil, nil, err
	}
	approvals, err := r.writeoffDocumentApprovalsGet(documentID)
	if err != nil {
		return nil, nil, nil, err
	}
	return &row, items, approvals, nil
}

func (r *Repo) writeoffDocumentItemsGet(documentID int64) ([]WriteoffDocumentItem, error) {
	rows, err := r.store.db.Query(`
		SELECT di.id, di.document_id, di.line_id, COALESCE(ll.name, ''),
			di.item_type,
			COALESCE(di.model_id, 0), COALESCE(di.component_id, 0), COALESCE(di.product_id, 0),
			COALESCE(di.serial, ''),
			COALESCE(
				NULLIF(TRIM(COALESCE(m.modeli, '') || CASE WHEN COALESCE(m.qisqa_nomi, '') <> '' THEN ' — ' || m.qisqa_nomi ELSE '' END), ''),
				NULLIF(TRIM(COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, '') || CASE WHEN COALESCE(c.full_name_uz, '') <> '' THEN ' — ' || c.full_name_uz ELSE '' END), ''),
				''
			),
			di.quantity, COALESCE(di.comment, ''), di.sort_order
		FROM writeoff.document_items di
		INNER JOIN lines.lines_list ll ON ll.line_id = di.line_id
		LEFT JOIN production.models m ON m.id = di.model_id
		LEFT JOIN production.components c ON c.id = di.component_id
		WHERE di.document_id = $1
		ORDER BY di.sort_order, di.id`, documentID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffDocumentItem{}
	for rows.Next() {
		item := WriteoffDocumentItem{}
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.LineID, &item.LineName, &item.ItemType,
			&item.ModelID, &item.ComponentID, &item.ProductID, &item.Serial, &item.ItemLabel,
			&item.Quantity, &item.Comment, &item.SortOrder,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) writeoffModelAllowedOnLine(lineID, modelID int) error {
	if modelID <= 0 {
		return errors.New("model tanlanmagan")
	}
	char, ok := productionPlanModelSeriyaSuffix(lineID)
	if !ok {
		return errors.New("bu liniya uchun model tanlash mumkin emas")
	}
	var seriya string
	err := r.store.db.QueryRow(`
		SELECT COALESCE(seriya_raqami, '') FROM production.models
		WHERE id = $1 AND status = true AND deleted = false`, modelID,
	).Scan(&seriya)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return errors.New("model topilmadi")
		}
		return err
	}
	seriya = strings.TrimSpace(seriya)
	if seriya == "" || !strings.EqualFold(string(seriya[len(seriya)-1]), char) {
		return fmt.Errorf("model ushbu liniyaga mos emas (seriya %s)", char)
	}
	return nil
}

func (r *Repo) writeoffResolveProductBySerial(lineID int, serial string) (productID, modelID int, err error) {
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return 0, 0, errors.New("serial kiritilishi shart")
	}
	var status string
	err = r.store.db.QueryRow(`
		SELECT id, model_id, status FROM lines.products
		WHERE line_id = $1 AND serial = $2`, lineID, serial,
	).Scan(&productID, &modelID, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, 0, fmt.Errorf("serial %s liniyada topilmadi", serial)
		}
		return 0, 0, err
	}
	if status != ProductStatusActive {
		return 0, 0, fmt.Errorf("serial %s active emas", serial)
	}
	if err := r.writeoffModelAllowedOnLine(lineID, modelID); err != nil {
		return 0, 0, err
	}
	return productID, modelID, nil
}

func (r *Repo) writeoffValidateItemInput(
	item *WriteoffSaveItemInput,
	serialsInDoc map[string]struct{},
	productModels map[string]int,
	componentBalances map[string]float64,
) error {
	if item.LineID <= 0 {
		return errors.New("liniya tanlanmagan")
	}
	qty, err := requireNormalizedWareQuantity(item.Quantity)
	if err != nil {
		return err
	}
	item.Quantity = qty

	switch {
	case IsWriteoffProductLine(item.LineID):
		if item.ItemType != WriteoffItemProduct {
			return errors.New("mahsulot liniyasi uchun faqat product")
		}
		serial := strings.TrimSpace(item.Serial)
		if serial == "" {
			return errors.New("serial kiritilishi shart")
		}
		if item.Quantity != 1 {
			return errors.New("mahsulot uchun miqdor 1 bo'lishi kerak")
		}
		key := fmt.Sprintf("%d:%s", item.LineID, serial)
		if _, dup := serialsInDoc[key]; dup {
			return fmt.Errorf("serial %s takrorlanmoqda", serial)
		}
		serialsInDoc[key] = struct{}{}

		if productModels != nil {
			modelID, ok := productModels[key]
			if !ok {
				return fmt.Errorf("serial %s liniyada topilmadi", serial)
			}
			item.ModelID = modelID
		} else {
			_, modelID, err := r.writeoffResolveProductBySerial(item.LineID, serial)
			if err != nil {
				return err
			}
			item.ModelID = modelID
		}
		item.Serial = serial

	case IsWriteoffAuxLine(item.LineID):
		if item.ItemType != WriteoffItemComponent {
			return errors.New("komponent liniyasi uchun faqat component")
		}
		if item.ComponentID <= 0 {
			return errors.New("komponent tanlanmagan")
		}
		var bal float64
		if componentBalances != nil {
			bal = componentBalances[writeoffComponentKey(item.LineID, item.ComponentID)]
		} else {
			var err error
			bal, err = r.LinesBalanceQuantity(item.LineID, item.ComponentID)
			if err != nil {
				return err
			}
		}
		if item.Quantity > bal {
			return fmt.Errorf("balans yetarli emas (joriy: %.4f)", bal)
		}
	default:
		return errors.New("bu liniya uchun hisobdan chiqarish qo'llab-quvvatlanmaydi")
	}
	return nil
}

func (r *Repo) WriteoffDocumentItemsSave(documentID int64, items []WriteoffSaveItemInput) error {
	status, err := r.writeoffDocumentStatus(documentID)
	if err != nil {
		return err
	}
	if status != WriteoffStatusDraft && status != WriteoffStatusRejected {
		return errors.New("faqat qoralama yoki rad etilgan hujjat tahrirlanadi")
	}

	serialsInDoc := map[string]struct{}{}
	productModels, err := r.writeoffPrefetchProductModels(items)
	if err != nil {
		return err
	}
	componentBalances, err := r.writeoffPrefetchComponentBalances(items)
	if err != nil {
		return err
	}
	for i, item := range items {
		item.SortOrder = i
		if IsWriteoffProductLine(item.LineID) {
			item.ItemType = WriteoffItemProduct
		} else if IsWriteoffAuxLine(item.LineID) {
			item.ItemType = WriteoffItemComponent
		}
		if err := r.writeoffValidateItemInput(&item, serialsInDoc, productModels, componentBalances); err != nil {
			return fmt.Errorf("qator %d: %w", i+1, err)
		}
		items[i] = item
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err = tx.Exec(`DELETE FROM writeoff.document_items WHERE document_id = $1`, documentID); err != nil {
		return err
	}

	if err := writeoffDocumentItemsInsertBatch(tx, documentID, items); err != nil {
		return err
	}

	if status == WriteoffStatusRejected {
		_, err = tx.Exec(`
			UPDATE writeoff.documents
			SET status = $2, rejected_by = NULL, rejected_at = NULL, reject_comment = ''
			WHERE id = $1`, documentID, WriteoffStatusDraft,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (r *Repo) WriteoffDocumentSubmit(documentID int64, userID int) error {
	status, err := r.writeoffDocumentStatus(documentID)
	if err != nil {
		return err
	}
	if status != WriteoffStatusDraft && status != WriteoffStatusRejected {
		return errors.New("hujjat allaqachon yuborilgan")
	}
	var count int
	err = r.store.db.QueryRow(`SELECT COUNT(*)::int FROM writeoff.document_items WHERE document_id = $1`, documentID).Scan(&count)
	if err != nil {
		return err
	}
	if count == 0 {
		return errors.New("kamida bitta qator kerak")
	}

	var responsibleCount int
	err = r.store.db.QueryRow(`SELECT COUNT(*)::int FROM writeoff.responsibles`).Scan(&responsibleCount)
	if err != nil {
		return err
	}
	if responsibleCount == 0 {
		return errors.New("hisobdan chiqarish mas'ullari tayinlanmagan")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE writeoff.documents
		SET status = $2, submitted_at = NOW(), rejected_by = NULL, rejected_at = NULL, reject_comment = '',
		    approved_by = NULL, approved_at = NULL
		WHERE id = $1 AND created_by = $3`,
		documentID, WriteoffStatusPending, userID,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("hujjat topilmadi yoki yuborish mumkin emas")
	}

	if _, err = tx.Exec(`DELETE FROM writeoff.document_approvals WHERE document_id = $1`, documentID); err != nil {
		return err
	}
	if _, err = tx.Exec(`
		INSERT INTO writeoff.document_approvals (document_id, user_id)
		SELECT $1, user_id FROM writeoff.responsibles`,
		documentID,
	); err != nil {
		return err
	}

	return tx.Commit()
}

func (r *Repo) WriteoffDocumentReject(documentID int64, userID int, comment string) error {
	ok, err := r.WriteoffIsResponsible(userID)
	if err != nil {
		return err
	}
	if !ok {
		return errors.New("siz tasdiqlash mas'uli emassiz")
	}
	res, err := r.store.db.Exec(`
		UPDATE writeoff.documents
		SET status = $2, rejected_by = $3, rejected_at = NOW(), reject_comment = $4
		WHERE id = $1 AND status = $5`,
		documentID, WriteoffStatusRejected, userID, strings.TrimSpace(comment), WriteoffStatusPending,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("hujjat topilmadi yoki tasdiqlash kutilmayapti")
	}
	_, err = r.store.db.Exec(`DELETE FROM writeoff.document_approvals WHERE document_id = $1`, documentID)
	return err
}

func (r *Repo) writeoffApproveProductTx(tx *sql.Tx, item WriteoffDocumentItem, userID int, docID int64) error {
	serial := strings.TrimSpace(item.Serial)
	var productID int
	var modelID int
	var status string
	err := tx.QueryRow(`
		SELECT id, model_id, status FROM lines.products
		WHERE line_id = $1 AND serial = $2 FOR UPDATE`,
		item.LineID, serial,
	).Scan(&productID, &modelID, &status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("serial %s topilmadi", serial)
		}
		return err
	}
	if status != ProductStatusActive {
		return fmt.Errorf("serial %s active emas", serial)
	}
	if modelID != item.ModelID {
		return errors.New("serial modeli mos kelmaydi")
	}

	_, err = tx.Exec(`
		UPDATE lines.products SET status = $2, transferred_at = COALESCE(transferred_at, NOW())
		WHERE id = $1`, productID, ProductStatusWrittenOff,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO writeoff.records (
			document_id, document_item_id, line_id, line_name, item_type,
			model_id, product_id, serial, item_label, quantity, comment,
			balance_before, balance_after, written_off_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,1,0,$12)`,
		docID, item.ID, item.LineID, item.LineName, WriteoffItemProduct,
		item.ModelID, productID, serial, item.ItemLabel, 1, item.Comment, userID,
	)
	return err
}

func (r *Repo) writeoffApproveComponentTx(tx *sql.Tx, repo *Repo, item WriteoffDocumentItem, userID int, docID int64) error {
	before, err := repo.LinesBalanceQuantity(item.LineID, item.ComponentID)
	if err != nil {
		return err
	}
	if item.Quantity > before {
		return fmt.Errorf("balans yetarli emas (joriy: %.4f)", before)
	}

	p := BalanceChangeParams{
		LineID:         item.LineID,
		ComponentID:    item.ComponentID,
		QuantityChange: -item.Quantity,
		UserID:         userID,
		Source:         "writeoff",
		Comment:        item.Comment,
	}
	after, err := repo.linesBalanceApplyChangeTx(tx, p)
	if err != nil {
		return err
	}

	_, err = tx.Exec(`
		INSERT INTO writeoff.records (
			document_id, document_item_id, line_id, line_name, item_type,
			component_id, item_label, quantity, comment,
			balance_before, balance_after, written_off_by
		) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)`,
		docID, item.ID, item.LineID, item.LineName, WriteoffItemComponent,
		item.ComponentID, item.ItemLabel, item.Quantity, item.Comment,
		before, after, userID,
	)
	return err
}

func (r *Repo) WriteoffDocumentApprove(documentID int64, userID int) (*WriteoffApproveResult, error) {
	ok, err := r.WriteoffIsResponsible(userID)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, errors.New("siz tasdiqlash mas'uli emassiz")
	}

	status, err := r.writeoffDocumentStatus(documentID)
	if err != nil {
		return nil, err
	}
	if status != WriteoffStatusPending {
		return nil, errors.New("hujjat tasdiqlash kutilmayapti")
	}

	var approvalID int64
	var approvedAt sql.NullTime
	err = r.store.db.QueryRow(`
		SELECT id, approved_at FROM writeoff.document_approvals
		WHERE document_id = $1 AND user_id = $2`, documentID, userID,
	).Scan(&approvalID, &approvedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, errors.New("siz ushbu hujjat uchun tasdiqlash mas'uli emassiz")
		}
		return nil, err
	}
	if approvedAt.Valid {
		return nil, errors.New("siz allaqachon tasdiqlagansiz")
	}

	items, err := r.writeoffDocumentItemsGet(documentID)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, errors.New("hujjat bo'sh")
	}

	tx, err := r.store.db.Begin()
	if err != nil {
		return nil, err
	}
	defer tx.Rollback()

	res, err := tx.Exec(`
		UPDATE writeoff.document_approvals
		SET approved_at = NOW()
		WHERE id = $1 AND approved_at IS NULL`, approvalID,
	)
	if err != nil {
		return nil, err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return nil, errors.New("tasdiqlash amalga oshmadi")
	}

	var pending int
	err = tx.QueryRow(`
		SELECT COUNT(*)::int FROM writeoff.document_approvals
		WHERE document_id = $1 AND approved_at IS NULL`, documentID,
	).Scan(&pending)
	if err != nil {
		return nil, err
	}

	var required int
	err = tx.QueryRow(`
		SELECT COUNT(*)::int FROM writeoff.document_approvals
		WHERE document_id = $1`, documentID,
	).Scan(&required)
	if err != nil {
		return nil, err
	}

	result := &WriteoffApproveResult{
		AllApproved:      pending == 0,
		ApprovalDone:     required - pending,
		ApprovalRequired: required,
	}

	if pending > 0 {
		if err = tx.Commit(); err != nil {
			return nil, err
		}
		return result, nil
	}

	for _, item := range items {
		switch item.ItemType {
		case WriteoffItemProduct:
			if err := r.writeoffApproveProductTx(tx, item, userID, documentID); err != nil {
				return nil, err
			}
		case WriteoffItemComponent:
			if err := r.writeoffApproveComponentTx(tx, r, item, userID, documentID); err != nil {
				return nil, err
			}
		default:
			return nil, errors.New("noto'g'ri item_type")
		}
	}

	_, err = tx.Exec(`
		UPDATE writeoff.documents
		SET status = $2, approved_by = $3, approved_at = NOW()
		WHERE id = $1 AND status = $4`,
		documentID, WriteoffStatusApproved, userID, WriteoffStatusPending,
	)
	if err != nil {
		return nil, err
	}
	if err = tx.Commit(); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) WriteoffRecordsList(documentID int64, limit int) ([]WriteoffRecord, error) {
	if limit <= 0 || limit > 1000 {
		limit = 500
	}
	query := `
		SELECT wr.id, wr.document_id, wr.line_id, wr.line_name, wr.item_type,
			COALESCE(wr.model_id, 0), COALESCE(wr.component_id, 0), COALESCE(wr.product_id, 0),
			COALESCE(wr.serial, ''), wr.item_label, wr.quantity, COALESCE(wr.comment, ''),
			COALESCE(wr.balance_before, 0), COALESCE(wr.balance_after, 0),
			wr.written_off_by, COALESCE(NULLIF(u.name, ''), u.login, ''),
			to_char(wr.written_off_at, 'YYYY-MM-DD HH24:MI:SS')
		FROM writeoff.records wr
		LEFT JOIN auth.users u ON u.id = wr.written_off_by
		WHERE 1=1`
	args := []any{}
	argN := 1
	if documentID > 0 {
		query += fmt.Sprintf(` AND wr.document_id = $%d`, argN)
		args = append(args, documentID)
		argN++
	}
	query += ` ORDER BY wr.written_off_at DESC, wr.id DESC`
	query += fmt.Sprintf(` LIMIT %d`, limit)

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffRecord{}
	for rows.Next() {
		item := WriteoffRecord{}
		if err := rows.Scan(
			&item.ID, &item.DocumentID, &item.LineID, &item.LineName, &item.ItemType,
			&item.ModelID, &item.ComponentID, &item.ProductID, &item.Serial, &item.ItemLabel,
			&item.Quantity, &item.Comment, &item.BalanceBefore, &item.BalanceAfter,
			&item.WrittenOffBy, &item.WrittenOffByName, &item.WrittenOffAt,
		); err != nil {
			return items, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) WriteoffCatalogLines() ([]WriteoffCatalogItem, error) {
	rows, err := r.store.db.Query(`
		SELECT line_id, name FROM lines.lines_list
		WHERE status = true AND line_id = ANY($1)
		ORDER BY line_id`, intSliceParam(append(append([]int{}, WriteoffProductLineIDs...), WriteoffAuxLineIDs...)),
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffCatalogItem{}
	for rows.Next() {
		item := WriteoffCatalogItem{ItemType: "line"}
		if err := rows.Scan(&item.LineID, &item.LineName); err != nil {
			return items, err
		}
		item.Label = item.LineName
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) WriteoffCatalogItems(lineID int) ([]WriteoffCatalogItem, error) {
	if IsWriteoffProductLine(lineID) {
		return r.writeoffCatalogModels(lineID)
	}
	if IsWriteoffAuxLine(lineID) {
		return r.writeoffCatalogComponents(lineID)
	}
	return nil, errors.New("noto'g'ri line_id")
}

func (r *Repo) writeoffCatalogModels(lineID int) ([]WriteoffCatalogItem, error) {
	char, ok := productionPlanModelSeriyaSuffix(lineID)
	if !ok {
		return nil, errors.New("noto'g'ri liniya")
	}
	lineName, _ := r.lineNameByID(lineID)
	query := `
		SELECT id, COALESCE(modeli, ''), COALESCE(qisqa_nomi, '')
		FROM production.models
		WHERE status = true AND deleted = false`
	args := []any{}
	if char != "" {
		query += productionPlanSeriyaEndsWithClause(1)
		args = append(args, char)
	}
	query += ` ORDER BY COALESCE(modeli, ''), id`

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffCatalogItem{}
	for rows.Next() {
		var id int
		var modeli, modelNomi string
		if err := rows.Scan(&id, &modeli, &modelNomi); err != nil {
			return items, err
		}
		label := strings.TrimSpace(modeli)
		if modelNomi != "" {
			if label != "" {
				label += " — "
			}
			label += modelNomi
		}
		items = append(items, WriteoffCatalogItem{
			ItemID: id, ItemType: WriteoffItemProduct, Label: label, LineID: lineID, LineName: lineName,
		})
	}
	return items, rows.Err()
}

func (r *Repo) writeoffCatalogComponents(lineID int) ([]WriteoffCatalogItem, error) {
	lineName, _ := r.lineNameByID(lineID)
	rows, err := r.store.db.Query(`
		SELECT c.id,
			COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''),
			COALESCE(c.full_name_uz, '')
		FROM lines.balance b
		INNER JOIN production.components c ON c.id = b.component_id
		WHERE b.line_id = $1 AND b.quantity > 0
		ORDER BY COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''), c.id`,
		lineID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := []WriteoffCatalogItem{}
	for rows.Next() {
		var id int
		var code, name string
		if err := rows.Scan(&id, &code, &name); err != nil {
			return items, err
		}
		label := strings.TrimSpace(code)
		if name != "" {
			if label != "" {
				label += " — "
			}
			label += name
		}
		items = append(items, WriteoffCatalogItem{
			ItemID: id, ItemType: WriteoffItemComponent, Label: label, LineID: lineID, LineName: lineName,
		})
	}
	return items, rows.Err()
}

func (r *Repo) WriteoffResolveLineID(lineName string) (int, error) {
	lineName = strings.TrimSpace(lineName)
	var id int
	err := r.store.db.QueryRow(`
		SELECT line_id FROM lines.lines_list
		WHERE status = true AND LOWER(name) = LOWER($1)`, lineName,
	).Scan(&id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return 0, fmt.Errorf("liniya topilmadi: %s", lineName)
		}
		return 0, err
	}
	return id, nil
}

func (r *Repo) WriteoffResolveModelID(lineID int, label string) (int, error) {
	items, err := r.writeoffCatalogModels(lineID)
	if err != nil {
		return 0, err
	}
	label = strings.TrimSpace(label)
	for _, item := range items {
		if strings.EqualFold(item.Label, label) {
			return item.ItemID, nil
		}
	}
	return 0, fmt.Errorf("model topilmadi: %s", label)
}

func (r *Repo) WriteoffResolveComponentID(lineID int, label string) (int, error) {
	items, err := r.writeoffCatalogComponents(lineID)
	if err != nil {
		return 0, err
	}
	label = strings.TrimSpace(label)
	for _, item := range items {
		if strings.EqualFold(item.Label, label) {
			return item.ItemID, nil
		}
	}
	return 0, fmt.Errorf("komponent topilmadi: %s", label)
}
