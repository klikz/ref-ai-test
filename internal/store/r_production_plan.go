package store

import (
	"bytes"
	"database/sql"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

const (
	PlanStatusDraft  = "draft"
	PlanStatusLocked = "locked"

	IchkiLineID = 7
)

var ProductionPlanProductLineIDs = []int{YigishLineID, QadoqlashLineID} // Yi'g'ish, Qadoqlash
var ProductionPlanAuxiliaryLineIDs = []int{EshikLineID}                 // Eshik

func IsProductionPlanProductLine(lineID int) bool {
	for _, id := range ProductionPlanProductLineIDs {
		if lineID == id {
			return true
		}
	}
	return false
}

func IsProductionPlanAuxiliaryLine(lineID int) bool {
	for _, id := range ProductionPlanAuxiliaryLineIDs {
		if lineID == id {
			return true
		}
	}
	return false
}

type DailyPlanDay struct {
	PlanDate string `json:"plan_date"`
	LineID   int    `json:"line_id"`
	LineName string `json:"line_name"`
	Status   string `json:"status"`
	LockedAt string `json:"locked_at"`
}

type DailyPlanItemRow struct {
	ID            int64  `json:"id"`
	PlanDate      string `json:"plan_date"`
	LineID        int    `json:"line_id"`
	LineName      string `json:"line_name"`
	ModelID       int    `json:"model_id"`
	ComponentID   int    `json:"component_id"`
	ItemKey       int    `json:"item_key"`
	Label         string `json:"label"`
	ArtikulRaqami string `json:"artikul_raqami"`
	OdooCode      string `json:"odoo_code"`
	Brend         string `json:"brend"`
	SeriyaRaqami  string `json:"seriya_raqami"`
	Modeli        string `json:"modeli"`
	Rangi         string `json:"rangi"`
	GsCodeCount   int    `json:"gscode_count"`
	ShiftNo       int    `json:"shift_no"`
	PlannedQty    int    `json:"planned_qty"`
	AllowOverplan bool   `json:"allow_overplan"`
	ActualQty     int    `json:"actual_qty"`
	RemainingQty  int    `json:"remaining_qty"`
	CompletionPct int    `json:"completion_pct"`
}

type DailyPlanImportError struct {
	Sheet   string `json:"sheet"`
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

type DailyPlanImportResult struct {
	ImportedRows int                    `json:"imported_rows"`
	SkippedRows  int                    `json:"skipped_rows"`
	Errors       []DailyPlanImportError `json:"errors"`
}

type PlanDashboardLine struct {
	LineID         int    `json:"line_id"`
	LineName       string `json:"line_name"`
	PlannedTotal   int    `json:"planned_total"`
	ActualTotal    int    `json:"actual_total"`
	CompletionPct  int    `json:"completion_pct"`
	ModelsInPlan   int    `json:"models_in_plan"`
	ModelsComplete int    `json:"models_complete"`
	Shift1Planned  int    `json:"shift1_planned"`
	Shift1Actual   int    `json:"shift1_actual"`
	Shift2Planned  int    `json:"shift2_planned"`
	Shift2Actual   int    `json:"shift2_actual"`
	CurrentShift   int    `json:"current_shift"`
}

type PlanDashboardResponse struct {
	PlanDate        string                   `json:"plan_date"`
	CurrentShiftNo  int                      `json:"current_shift_no"`
	CurrentPlanDate string                   `json:"current_plan_date"`
	Lines           []PlanDashboardLine      `json:"lines"`
	Items           []DailyPlanItemRow       `json:"items"`
}

func planDateOrToday(planDate string) string {
	planDate = strings.TrimSpace(planDate)
	if planDate == "" {
		return time.Now().Format("2006-01-02")
	}
	return planDate
}

func productionPlanToday() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
}

func productionPlanParseDate(planDate string) (time.Time, error) {
	t, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(planDate), time.Local)
	if err != nil {
		return time.Time{}, err
	}
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.Local), nil
}

func productionPlanIsPastDate(planDate string) bool {
	d, err := productionPlanParseDate(planDate)
	if err != nil {
		return false
	}
	return d.Before(productionPlanToday())
}

func (r *Repo) productionPlanEffectiveDayStatus(planDate string, lineID int) (string, error) {
	if productionPlanIsPastDate(planDate) {
		return PlanStatusLocked, nil
	}
	status, err := r.productionPlanDayStatus(planDate, lineID)
	if err != nil {
		return "", err
	}
	if status == "" {
		return PlanStatusDraft, nil
	}
	return status, nil
}

func (r *Repo) ProductionPlanDayStatus(planDate string, lineID int) (string, error) {
	return r.productionPlanDayStatus(planDate, lineID)
}

func (r *Repo) productionPlanDayStatus(planDate string, lineID int) (string, error) {
	status := PlanStatusDraft
	err := r.store.db.QueryRow(`
		SELECT status
		FROM production.daily_plan_days
		WHERE plan_date = $1::date AND line_id = $2`,
		planDate, lineID,
	).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return status, nil
}

func (r *Repo) productionPlanEnsureDayDraft(tx *sql.Tx, planDate string, lineID int) error {
	if productionPlanIsPastDate(planDate) {
		return fmt.Errorf("%s — o'tgan kun, o'zgartirish mumkin emas", planDate)
	}
	status, err := r.productionPlanDayStatusTx(tx, planDate, lineID)
	if err != nil {
		return err
	}
	if status == PlanStatusLocked {
		return fmt.Errorf("%s sanasi uchun reja tasdiqlangan — o'zgartirish mumkin emas", planDate)
	}
	if status == "" {
		_, err = tx.Exec(`
			INSERT INTO production.daily_plan_days (plan_date, line_id, status)
			VALUES ($1::date, $2, $3)
			ON CONFLICT (plan_date, line_id) DO NOTHING`,
			planDate, lineID, PlanStatusDraft,
		)
	}
	return err
}

func (r *Repo) productionPlanDayStatusTx(tx *sql.Tx, planDate string, lineID int) (string, error) {
	status := PlanStatusDraft
	err := tx.QueryRow(`
		SELECT status
		FROM production.daily_plan_days
		WHERE plan_date = $1::date AND line_id = $2`,
		planDate, lineID,
	).Scan(&status)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", nil
		}
		return "", err
	}
	return status, nil
}

func (r *Repo) ProductionPlanActualQty(planDate string, lineID, shiftNo, modelID, componentID int) (int, error) {
	planDate = planDateOrToday(planDate)
	if shiftNo != 2 {
		shiftNo = 1
	}
	start, end, err := r.ProductionShiftWindow(planDate, shiftNo)
	if err != nil {
		return 0, err
	}
	if modelID > 0 {
		var count int
		err := r.store.db.QueryRow(`
			SELECT COUNT(*)::int
			FROM lines.products
			WHERE line_id = $1 AND model_id = $2
			  AND time >= $3
			  AND time < $4`,
			lineID, modelID, start, end,
		).Scan(&count)
		return count, err
	}
	if componentID > 0 && lineID == FinPressLineID {
		var count int
		err := r.store.db.QueryRow(`
			SELECT COALESCE(SUM(count), 0)::int
			FROM production.fin_press_print_sessions
			WHERE component_id = $1
			  AND c_time >= $2
			  AND c_time < $3`,
			componentID, start, end,
		).Scan(&count)
		return count, err
	}
	if componentID > 0 && (lineID == RadiatorLineID || lineID == KlapanLineID || lineID == EshikLineID) {
		var count int
		err := r.store.db.QueryRow(`
			SELECT COUNT(*)::int
			FROM lines.auxiliary_products
			WHERE line_id = $1 AND component_id = $2
			  AND c_time >= $3
			  AND c_time < $4`,
			lineID, componentID, start, end,
		).Scan(&count)
		return count, err
	}
	return 0, nil
}

// ProductionPlanEshikPairActualQty returns complete freeze+ref pair count for an eshik model.
func (r *Repo) ProductionPlanEshikPairActualQty(planDate string, shiftNo, eshikModelID int) (int, error) {
	freezeID, refID, err := r.EshikComponentIDsByModelID(eshikModelID)
	if err != nil {
		return 0, err
	}
	freezeActual, err := r.ProductionPlanActualQty(planDate, EshikLineID, shiftNo, 0, freezeID)
	if err != nil {
		return 0, err
	}
	refActual, err := r.ProductionPlanActualQty(planDate, EshikLineID, shiftNo, 0, refID)
	if err != nil {
		return 0, err
	}
	return eshikPairMin(freezeActual, refActual), nil
}

func (r *Repo) ProductionPlanAssertCanProduce(lineID, modelID, componentID, qty int) error {
	_, err := r.productionPlanCheckCanProduce(lineID, modelID, componentID, qty, false)
	return err
}

func (r *Repo) ProductionPlanCheckCanProduceAllowMissing(lineID, modelID, componentID, qty int) (string, error) {
	return r.productionPlanCheckCanProduce(lineID, modelID, componentID, qty, true)
}

func (r *Repo) productionPlanCheckCanProduce(lineID, modelID, componentID, qty int, lenient bool) (string, error) {
	if qty <= 0 {
		qty = 1
	}
	if !IsProductionPlanProductLine(lineID) && !IsProductionPlanAuxiliaryLine(lineID) {
		return "", nil
	}
	if modelID > 0 && componentID > 0 {
		return "", errors.New("reja tekshiruvi: model yoki komponent")
	}
	if modelID <= 0 && componentID <= 0 {
		return "", errors.New("reja tekshiruvi: model yoki komponent tanlanmagan")
	}

	current, err := r.ProductionCurrentShift(time.Time{})
	if err != nil {
		return "", err
	}
	if current.ShiftNo == 0 || current.PlanDate == "" {
		if lenient {
			return "Hozir smena vaqti emas", nil
		}
		return "", errors.New("hozir smena vaqti emas — ishlab chiqarish mumkin emas")
	}
	planDate := current.PlanDate
	shiftNo := current.ShiftNo

	status, err := r.productionPlanDayStatus(planDate, lineID)
	if err != nil {
		return "", err
	}
	if status != PlanStatusLocked {
		if lenient {
			return "Bugungi reja tasdiqlanmagan", nil
		}
		return "", errors.New("bugungi reja tasdiqlanmagan — ishlab chiqarish mumkin emas")
	}

	var plannedQty int
	var allowOverplan bool
	if modelID > 0 {
		err = r.store.db.QueryRow(`
			SELECT planned_qty, allow_overplan
			FROM production.daily_plan_items
			WHERE plan_date = $1::date AND line_id = $2 AND model_id = $3 AND shift_no = $4`,
			planDate, lineID, modelID, shiftNo,
		).Scan(&plannedQty, &allowOverplan)
	} else {
		err = r.store.db.QueryRow(`
			SELECT planned_qty, allow_overplan
			FROM production.daily_plan_items
			WHERE plan_date = $1::date AND line_id = $2 AND component_id = $3 AND shift_no = $4`,
			planDate, lineID, componentID, shiftNo,
		).Scan(&plannedQty, &allowOverplan)
	}
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			if lenient {
				return "Bu mahsulot joriy smena rejasida yo'q", nil
			}
			return "", errors.New("bu mahsulot joriy smena rejasida yo'q — ishlab chiqarish mumkin emas")
		}
		return "", err
	}

	actual, err := r.ProductionPlanActualQty(planDate, lineID, shiftNo, modelID, componentID)
	if err != nil {
		return "", err
	}
	if !allowOverplan && actual+qty > plannedQty {
		return "", fmt.Errorf("smena rejasi bajarildi (reja: %d, fakt: %d) — qo'shimcha ishlab chiqarish mumkin emas", plannedQty, actual)
	}
	return "", nil
}

func (r *Repo) ProductionPlanLockDay(planDate string, lineID, userID int) error {
	planDate = planDateOrToday(planDate)
	if productionPlanIsPastDate(planDate) {
		return errors.New("o'tgan kunlarni tasdiqlash yoki o'zgartirish mumkin emas")
	}
	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var itemCount int
	err = tx.QueryRow(`
		SELECT COUNT(*)::int
		FROM production.daily_plan_items
		WHERE plan_date = $1::date AND line_id = $2`,
		planDate, lineID,
	).Scan(&itemCount)
	if err != nil {
		return err
	}
	if itemCount == 0 {
		return errors.New("reja bo'sh — avval Saqlash tugmasini bosing")
	}

	_, err = tx.Exec(`
		INSERT INTO production.daily_plan_days (plan_date, line_id, status, locked_at, locked_by)
		VALUES ($1::date, $2, $3, NOW(), $4)
		ON CONFLICT (plan_date, line_id) DO UPDATE
		SET status = EXCLUDED.status,
		    locked_at = NOW(),
		    locked_by = EXCLUDED.locked_by`,
		planDate, lineID, PlanStatusLocked, nullInt(userID),
	)
	if err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repo) ProductionPlanLockMonth(yearMonth string, lineID, userID int) (int, error) {
	yearMonth = strings.TrimSpace(yearMonth)
	if len(yearMonth) != 7 {
		return 0, errors.New("oy formati: YYYY-MM")
	}
	start := yearMonth + "-01"
	tx, err := r.store.db.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()

	today := productionPlanToday().Format("2006-01-02")
	result, err := tx.Exec(`
		INSERT INTO production.daily_plan_days (plan_date, line_id, status, locked_at, locked_by)
		SELECT DISTINCT i.plan_date, i.line_id, $3, NOW(), $4
		FROM production.daily_plan_items i
		LEFT JOIN production.daily_plan_days d
			ON d.plan_date = i.plan_date AND d.line_id = i.line_id
		WHERE i.line_id = $1
		  AND i.plan_date >= $2::date
		  AND i.plan_date < ($2::date + INTERVAL '1 month')
		  AND i.plan_date >= $5::date
		  AND COALESCE(d.status, $6) <> $3
		ON CONFLICT (plan_date, line_id) DO UPDATE
		SET status = EXCLUDED.status,
		    locked_at = NOW(),
		    locked_by = EXCLUDED.locked_by`,
		lineID, start, PlanStatusLocked, nullInt(userID), today, PlanStatusDraft,
	)
	if err != nil {
		return 0, err
	}
	locked64, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}
	locked := int(locked64)
	if locked == 0 {
		return 0, errors.New("tasdiqlash uchun reja topilmadi — avval Saqlash tugmasini bosing")
	}
	return locked, tx.Commit()
}

func (r *Repo) ProductionPlanUnlockDay(planDate string, lineID int) error {
	planDate = planDateOrToday(planDate)
	if productionPlanIsPastDate(planDate) {
		return errors.New("o'tgan kunlarni qayta ochish mumkin emas")
	}
	status, err := r.productionPlanDayStatus(planDate, lineID)
	if err != nil {
		return err
	}
	if status != PlanStatusLocked {
		return errors.New("kun tasdiqlanmagan — ochish mumkin emas")
	}
	res, err := r.store.db.Exec(`
		UPDATE production.daily_plan_days
		SET status = $3, locked_at = NULL, locked_by = NULL
		WHERE plan_date = $1::date AND line_id = $2 AND status = $4`,
		planDate, lineID, PlanStatusDraft, PlanStatusLocked,
	)
	if err != nil {
		return err
	}
	affected, _ := res.RowsAffected()
	if affected == 0 {
		return errors.New("kun tasdiqlanmagan — ochish mumkin emas")
	}
	return nil
}

func (r *Repo) ProductionPlanUnlockMonth(yearMonth string, lineID int) (int, error) {
	yearMonth = strings.TrimSpace(yearMonth)
	if len(yearMonth) != 7 {
		return 0, errors.New("oy formati: YYYY-MM")
	}
	start := yearMonth + "-01"
	today := productionPlanToday().Format("2006-01-02")
	res, err := r.store.db.Exec(`
		UPDATE production.daily_plan_days
		SET status = $3, locked_at = NULL, locked_by = NULL
		WHERE line_id = $1
		  AND plan_date >= $2::date
		  AND plan_date < ($2::date + INTERVAL '1 month')
		  AND plan_date >= $5::date
		  AND status = $4`,
		lineID, start, PlanStatusDraft, PlanStatusLocked, today,
	)
	if err != nil {
		return 0, err
	}
	unlocked, _ := res.RowsAffected()
	if unlocked == 0 {
		return 0, errors.New("tasdiqlangan kunlar topilmadi")
	}
	return int(unlocked), nil
}

func nullInt(v int) any {
	if v <= 0 {
		return nil
	}
	return v
}

type PlanUpsertCell struct {
	PlanDate      string
	LineID        int
	ModelID       int
	ComponentID   int
	ShiftNo       int
	PlannedQty    int
	AllowOverplan bool
}

func (r *Repo) ProductionPlanUpsertCells(cells []PlanUpsertCell) error {
	expanded, err := r.expandEshikPlanCells(cells)
	if err != nil {
		return err
	}
	return r.productionPlanSaveCells(expanded, false)
}

func (r *Repo) ProductionPlanSaveCells(cells []PlanUpsertCell) error {
	expanded, err := r.expandEshikPlanCells(cells)
	if err != nil {
		return err
	}
	return r.productionPlanSaveCells(expanded, true)
}

// expandEshikPlanCells turns eshik_model_id (sent as component_id from UI) into freeze+ref component plan rows.
func (r *Repo) expandEshikPlanCells(cells []PlanUpsertCell) ([]PlanUpsertCell, error) {
	if len(cells) == 0 {
		return cells, nil
	}
	out := make([]PlanUpsertCell, 0, len(cells)*2)
	for _, cell := range cells {
		if cell.LineID != EshikLineID {
			out = append(out, cell)
			continue
		}
		modelID := cell.ComponentID
		if modelID <= 0 {
			modelID = cell.ModelID
		}
		if modelID <= 0 {
			return nil, errors.New("eshik modeli tanlanmagan")
		}
		freezeID, refID, err := r.EshikComponentIDsByModelID(modelID)
		if err != nil {
			return nil, err
		}
		freezeCell := cell
		freezeCell.ModelID = 0
		freezeCell.ComponentID = freezeID
		refCell := cell
		refCell.ModelID = 0
		refCell.ComponentID = refID
		out = append(out, freezeCell, refCell)
	}
	return out, nil
}

func (r *Repo) productionPlanSaveCells(cells []PlanUpsertCell, rejectLocked bool) error {
	if len(cells) == 0 {
		return nil
	}
	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	upsertCells := make([]PlanUpsertCell, 0, len(cells))
	ensuredDays := map[string]struct{}{}

	for _, cell := range cells {
		if productionPlanIsPastDate(cell.PlanDate) {
			if rejectLocked {
				return fmt.Errorf("%s — o'tgan kun, o'zgartirish mumkin emas", cell.PlanDate)
			}
			continue
		}
		if rejectLocked {
			status, err := r.productionPlanDayStatusTx(tx, cell.PlanDate, cell.LineID)
			if err != nil {
				return err
			}
			if status == PlanStatusLocked {
				return fmt.Errorf("%s sanasi uchun reja tasdiqlangan — o'zgartirish mumkin emas", cell.PlanDate)
			}
		} else {
			status, err := r.productionPlanDayStatusTx(tx, cell.PlanDate, cell.LineID)
			if err != nil {
				return err
			}
			if status == PlanStatusLocked {
				continue
			}
		}

		if cell.PlannedQty <= 0 {
			if err := r.productionPlanDeleteCellTx(tx, cell); err != nil {
				return err
			}
			continue
		}

		if cell.ModelID <= 0 && cell.ComponentID <= 0 {
			return errors.New("model yoki komponent tanlanmagan")
		}

		dayKey := cell.PlanDate + "|" + fmt.Sprint(cell.LineID)
		if _, ok := ensuredDays[dayKey]; !ok {
			if err := r.productionPlanEnsureDayDraft(tx, cell.PlanDate, cell.LineID); err != nil {
				return err
			}
			ensuredDays[dayKey] = struct{}{}
		}
		upsertCells = append(upsertCells, cell)
	}
	if err := productionPlanUpsertCellsBatchTx(tx, upsertCells); err != nil {
		return err
	}
	return tx.Commit()
}

func (r *Repo) productionPlanDeleteCellTx(tx *sql.Tx, cell PlanUpsertCell) error {
	shiftNo := cell.ShiftNo
	if shiftNo != 2 {
		shiftNo = 1
	}
	if cell.ModelID > 0 {
		_, err := tx.Exec(`
			DELETE FROM production.daily_plan_items
			WHERE plan_date = $1::date AND line_id = $2 AND model_id = $3 AND shift_no = $4`,
			cell.PlanDate, cell.LineID, cell.ModelID, shiftNo,
		)
		return err
	}
	if cell.ComponentID > 0 {
		_, err := tx.Exec(`
			DELETE FROM production.daily_plan_items
			WHERE plan_date = $1::date AND line_id = $2 AND component_id = $3 AND shift_no = $4`,
			cell.PlanDate, cell.LineID, cell.ComponentID, shiftNo,
		)
		return err
	}
	return nil
}

// func (r *Repo) productionPlanUpsertCellTx(tx *sql.Tx, cell PlanUpsertCell) error {
// 	if cell.ModelID > 0 {
// 		res, err := tx.Exec(`
// 			UPDATE production.daily_plan_items
// 			SET planned_qty = $4, allow_overplan = $5
// 			WHERE plan_date = $1::date AND line_id = $2 AND model_id = $3`,
// 			cell.PlanDate, cell.LineID, cell.ModelID, cell.PlannedQty, cell.AllowOverplan,
// 		)
// 		if err != nil {
// 			return err
// 		}
// 		affected, _ := res.RowsAffected()
// 		if affected == 0 {
// 			_, err = tx.Exec(`
// 				INSERT INTO production.daily_plan_items
// 					(plan_date, line_id, model_id, component_id, planned_qty, allow_overplan)
// 				VALUES ($1::date, $2, $3, NULL, $4, $5)`,
// 				cell.PlanDate, cell.LineID, cell.ModelID, cell.PlannedQty, cell.AllowOverplan,
// 			)
// 		}
// 		return err
// 	}
// 	if cell.ComponentID > 0 {
// 		res, err := tx.Exec(`
// 			UPDATE production.daily_plan_items
// 			SET planned_qty = $4, allow_overplan = $5
// 			WHERE plan_date = $1::date AND line_id = $2 AND component_id = $3`,
// 			cell.PlanDate, cell.LineID, cell.ComponentID, cell.PlannedQty, cell.AllowOverplan,
// 		)
// 		if err != nil {
// 			return err
// 		}
// 		affected, _ := res.RowsAffected()
// 		if affected == 0 {
// 			_, err = tx.Exec(`
// 				INSERT INTO production.daily_plan_items
// 					(plan_date, line_id, model_id, component_id, planned_qty, allow_overplan)
// 				VALUES ($1::date, $2, NULL, $3, $4, $5)`,
// 				cell.PlanDate, cell.LineID, cell.ComponentID, cell.PlannedQty, cell.AllowOverplan,
// 			)
// 		}
// 		return err
// 	}
// 	return nil
// }

func (r *Repo) ProductionPlanApplyOverplanSettings(lineID int, yearMonth string, settings map[int]bool, useModel bool) error {
	start := yearMonth + "-01"
	today := productionPlanToday().Format("2006-01-02")
	tx, err := r.store.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	for itemKey, allow := range settings {
		if useModel {
			_, err = tx.Exec(`
				UPDATE production.daily_plan_items i
				SET allow_overplan = $4
				FROM production.daily_plan_days d
				WHERE i.plan_date = d.plan_date AND i.line_id = d.line_id
				  AND i.line_id = $1
				  AND i.model_id = $2
				  AND i.plan_date >= $3::date
				  AND i.plan_date < ($3::date + INTERVAL '1 month')
				  AND i.plan_date >= $6::date
				  AND d.status = $5`,
				lineID, itemKey, start, allow, PlanStatusDraft, today,
			)
		} else {
			_, err = tx.Exec(`
				UPDATE production.daily_plan_items i
				SET allow_overplan = $4
				FROM production.daily_plan_days d
				WHERE i.plan_date = d.plan_date AND i.line_id = d.line_id
				  AND i.line_id = $1
				  AND i.component_id = $2
				  AND i.plan_date >= $3::date
				  AND i.plan_date < ($3::date + INTERVAL '1 month')
				  AND i.plan_date >= $6::date
				  AND d.status = $5`,
				lineID, itemKey, start, allow, PlanStatusDraft, today,
			)
		}
		if err != nil {
			return err
		}
	}
	return tx.Commit()
}

func (r *Repo) ProductionPlanGetDay(planDate string, lineID int, shiftNo int) ([]DailyPlanItemRow, string, error) {
	planDate = planDateOrToday(planDate)
	status, err := r.productionPlanEffectiveDayStatus(planDate, lineID)
	if err != nil {
		return nil, "", err
	}

	query := `
		SELECT i.id, i.plan_date::text, i.line_id,
			COALESCE(i.model_id, 0), COALESCE(i.component_id, 0),
			COALESCE(NULLIF(m.modeli, ''), NULLIF(c.factory_code, ''), ''),
			COALESCE(m.qisqa_nomi, ''),
			COALESCE(NULLIF(m.odoo_code, ''), NULLIF(c.odoo_code, ''), ''),
			COALESCE(m.brend, ''),
			COALESCE(m.seriya_raqami, ''),
			COALESCE(m.modeli, ''),
			COALESCE(m.rangi, ''),
			COALESCE((
				SELECT COUNT(*)::int
				FROM production.gscodes gs
				WHERE gs.model_id = m.id AND gs.status = true
			), 0),
			i.shift_no, i.planned_qty, i.allow_overplan
		FROM production.daily_plan_items i
		LEFT JOIN production.models m ON m.id = i.model_id
		LEFT JOIN production.components c ON c.id = i.component_id
		WHERE i.plan_date = $1::date AND i.line_id = $2
		  AND ($3::int = 0 OR i.shift_no = $3)
		ORDER BY i.shift_no, COALESCE(m.modeli, c.factory_code, '')`

	rows, err := r.store.db.Query(query, planDate, lineID, shiftNo)
	if err != nil {
		return nil, status, err
	}
	defer rows.Close()

	actuals, err := r.productionPlanActualQtyForDay(planDate, lineID)
	if err != nil {
		return nil, status, err
	}

	items := []DailyPlanItemRow{}
	for rows.Next() {
		row := DailyPlanItemRow{PlanDate: planDate, LineID: lineID}
		if err := rows.Scan(
			&row.ID, &row.PlanDate, &row.LineID,
			&row.ModelID, &row.ComponentID,
			&row.Label, &row.ArtikulRaqami, &row.OdooCode,
			&row.Brend, &row.SeriyaRaqami, &row.Modeli, &row.Rangi, &row.GsCodeCount,
			&row.ShiftNo, &row.PlannedQty, &row.AllowOverplan,
		); err != nil {
			return items, status, err
		}
		if row.ModelID > 0 {
			row.ItemKey = row.ModelID
		} else {
			row.ItemKey = row.ComponentID
		}
		applyProductionPlanActuals(&row, actuals)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return items, status, err
	}
	if lineID == EshikLineID {
		items, err = r.groupEshikDailyPlanItems(items)
		if err != nil {
			return nil, status, err
		}
	}
	return items, status, nil
}

func (r *Repo) ProductionPlanReport(dateFrom, dateTo string, lineIDs []int) ([]DailyPlanItemRow, error) {
	if strings.TrimSpace(dateFrom) == "" {
		dateFrom = time.Now().Format("2006-01-02")
	}
	if strings.TrimSpace(dateTo) == "" {
		dateTo = dateFrom
	}
	if len(lineIDs) == 0 {
		lineIDs = append(append([]int{}, ProductionPlanProductLineIDs...), ProductionPlanAuxiliaryLineIDs...)
	}

	query := `
		SELECT i.id, i.plan_date::text, i.line_id, ll.name,
			COALESCE(i.model_id, 0), COALESCE(i.component_id, 0),
			COALESCE(NULLIF(m.modeli, ''), NULLIF(c.factory_code, ''), NULLIF(c.manufacturer_code, ''), ''),
			COALESCE(m.qisqa_nomi, ''),
			COALESCE(NULLIF(m.odoo_code, ''), NULLIF(c.odoo_code, ''), ''),
			COALESCE(m.brend, ''),
			COALESCE(m.seriya_raqami, ''),
			COALESCE(m.modeli, ''),
			COALESCE(m.rangi, ''),
			COALESCE((
				SELECT COUNT(*)::int
				FROM production.gscodes gs
				WHERE gs.model_id = m.id AND gs.status = true
			), 0),
			i.shift_no, i.planned_qty, i.allow_overplan, COALESCE(d.status, 'draft')
		FROM production.daily_plan_items i
		INNER JOIN lines.lines_list ll ON ll.line_id = i.line_id
		LEFT JOIN production.daily_plan_days d
			ON d.plan_date = i.plan_date AND d.line_id = i.line_id
		LEFT JOIN production.models m ON m.id = i.model_id
		LEFT JOIN production.components c ON c.id = i.component_id
		WHERE i.plan_date >= $1::date
		  AND i.plan_date <= $2::date
		  AND i.line_id = ANY($3)
		ORDER BY i.plan_date, ll.name, i.shift_no, COALESCE(m.modeli, c.factory_code, '')`

	rows, err := r.store.db.Query(query, dateFrom, dateTo, intSliceParam(lineIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	actuals, err := r.productionPlanActualQtyBatch(dateFrom, dateTo, lineIDs)
	if err != nil {
		return nil, err
	}

	items := []DailyPlanItemRow{}
	for rows.Next() {
		row := DailyPlanItemRow{}
		var lineName, dayStatus string
		if err := rows.Scan(
			&row.ID, &row.PlanDate, &row.LineID, &lineName,
			&row.ModelID, &row.ComponentID, &row.Label,
			&row.ArtikulRaqami, &row.OdooCode,
			&row.Brend, &row.SeriyaRaqami, &row.Modeli, &row.Rangi, &row.GsCodeCount,
			&row.ShiftNo, &row.PlannedQty, &row.AllowOverplan, &dayStatus,
		); err != nil {
			return items, err
		}
		row.LineName = lineName
		if row.ModelID > 0 {
			row.ItemKey = row.ModelID
		} else {
			row.ItemKey = row.ComponentID
		}
		applyProductionPlanActuals(&row, actuals)
		items = append(items, row)
	}
	if err := rows.Err(); err != nil {
		return items, err
	}

	eshikItems := make([]DailyPlanItemRow, 0)
	otherItems := make([]DailyPlanItemRow, 0, len(items))
	for _, item := range items {
		if item.LineID == EshikLineID {
			eshikItems = append(eshikItems, item)
			continue
		}
		otherItems = append(otherItems, item)
	}
	if len(eshikItems) > 0 {
		grouped, err := r.groupEshikDailyPlanItems(eshikItems)
		if err != nil {
			return nil, err
		}
		otherItems = append(otherItems, grouped...)
	}
	return otherItems, nil
}

func (r *Repo) ProductionPlanDashboard(planDate string) (*PlanDashboardResponse, error) {
	planDate = planDateOrToday(planDate)
	current, err := r.ProductionCurrentShift(time.Time{})
	if err != nil {
		return nil, err
	}
	resp := &PlanDashboardResponse{
		PlanDate:        planDate,
		CurrentShiftNo:  current.ShiftNo,
		CurrentPlanDate: current.PlanDate,
		Lines:           []PlanDashboardLine{},
		Items:           []DailyPlanItemRow{},
	}

	lineRows, err := r.store.db.Query(`
		SELECT line_id, name
		FROM lines.lines_list
		WHERE line_id = ANY($1)
		ORDER BY line_id`, intSliceParam(append(append([]int{}, ProductionPlanProductLineIDs...), ProductionPlanAuxiliaryLineIDs...)))
	if err != nil {
		return resp, err
	}
	defer lineRows.Close()

	for lineRows.Next() {
		var line PlanDashboardLine
		if err := lineRows.Scan(&line.LineID, &line.LineName); err != nil {
			return resp, err
		}
		line.CurrentShift = current.ShiftNo
		items, _, err := r.ProductionPlanGetDay(planDate, line.LineID, 0)
		if err != nil {
			return resp, err
		}
		for _, item := range items {
			line.PlannedTotal += item.PlannedQty
			line.ActualTotal += item.ActualQty
			line.ModelsInPlan++
			if item.ActualQty >= item.PlannedQty {
				line.ModelsComplete++
			}
			if item.ShiftNo == 2 {
				line.Shift2Planned += item.PlannedQty
				line.Shift2Actual += item.ActualQty
			} else {
				line.Shift1Planned += item.PlannedQty
				line.Shift1Actual += item.ActualQty
			}
		}
		if line.PlannedTotal > 0 {
			line.CompletionPct = line.ActualTotal * 100 / line.PlannedTotal
		}
		resp.Lines = append(resp.Lines, line)
		resp.Items = append(resp.Items, items...)
	}
	return resp, lineRows.Err()
}

func (r *Repo) ProductionPlanLinePlannedTotal(planDate string, lineID int) (int, error) {
	planDate = planDateOrToday(planDate)
	if lineID == EshikLineID {
		var total int
		err := r.store.db.QueryRow(`
			SELECT COALESCE(SUM(max_qty), 0)::int
			FROM (
				SELECT MAX(i.planned_qty) AS max_qty
				FROM production.daily_plan_items i
				INNER JOIN production.eshik_model_parts p ON p.component_id = i.component_id
				WHERE i.plan_date = $1::date AND i.line_id = $2
				GROUP BY p.eshik_model_id, i.shift_no
			) t`,
			planDate, lineID).Scan(&total)
		return total, err
	}
	var total int
	err := r.store.db.QueryRow(`
		SELECT COALESCE(SUM(planned_qty), 0)
		FROM production.daily_plan_items
		WHERE plan_date = $1::date AND line_id = $2`,
		planDate, lineID).Scan(&total)
	return total, err
}

func (r *Repo) ProductionPlanLineShiftPlannedTotal(planDate string, lineID, shiftNo int) (int, error) {
	planDate = planDateOrToday(planDate)
	if shiftNo != 1 && shiftNo != 2 {
		return 0, nil
	}
	if lineID == EshikLineID {
		var total int
		err := r.store.db.QueryRow(`
			SELECT COALESCE(SUM(max_qty), 0)::int
			FROM (
				SELECT MAX(i.planned_qty) AS max_qty
				FROM production.daily_plan_items i
				INNER JOIN production.eshik_model_parts p ON p.component_id = i.component_id
				WHERE i.plan_date = $1::date AND i.line_id = $2 AND i.shift_no = $3
				GROUP BY p.eshik_model_id
			) t`,
			planDate, lineID, shiftNo).Scan(&total)
		return total, err
	}
	var total int
	err := r.store.db.QueryRow(`
		SELECT COALESCE(SUM(planned_qty), 0)
		FROM production.daily_plan_items
		WHERE plan_date = $1::date AND line_id = $2 AND shift_no = $3`,
		planDate, lineID, shiftNo).Scan(&total)
	return total, err
}

type planTemplateItem struct {
	ItemKey       int
	Label         string
	OdooCode      string
	SeriyaRaqami  string
	FullNameUz    string
	Rangi         string
	AllowOverplan bool
}

func productionPlanModelSeriyaSuffix(lineID int) (string, bool) {
	switch lineID {
	case T1LineID, T2LineID, T3LineID:
		return "T", true
	case IchkiLineID:
		return "I", true
	default:
		return "", false
	}
}

func productionPlanSeriyaEndsWithClause(paramIndex int) string {
	return fmt.Sprintf(` AND UPPER(RIGHT(TRIM(COALESCE(seriya_raqami, '')), 1)) = UPPER($%d)`, paramIndex)
}

func (r *Repo) productionPlanTemplateItems(lineID int, modelIDs, componentIDs []int) ([]planTemplateItem, error) {
	switch {
	case lineID == YigishLineID, lineID == QadoqlashLineID, IsProductionPlanProductLine(lineID):
		return r.productionPlanTemplateModels(lineID, modelIDs)
	case lineID == FinPressLineID:
		return r.productionPlanTemplateFinPress(componentIDs)
	case lineID == RadiatorLineID:
		return r.productionPlanTemplateRadiator(componentIDs)
	case lineID == KlapanLineID:
		return r.productionPlanTemplateKlapan(componentIDs)
	case lineID == EshikLineID:
		return r.productionPlanTemplateEshik(componentIDs)
	default:
		return nil, fmt.Errorf("noto'g'ri line_id: %d", lineID)
	}
}

func (r *Repo) productionPlanTemplateModels(lineID int, modelIDs []int) ([]planTemplateItem, error) {
	items := []planTemplateItem{}
	char, hasChar := productionPlanModelSeriyaSuffix(lineID)
	query := `
		SELECT id, COALESCE(modeli, ''), COALESCE(odoo_code, ''), COALESCE(seriya_raqami, ''), '', COALESCE(rangi, '')
		FROM production.models
		WHERE status = true AND deleted = false`
	args := []any{}
	argN := 1
	if hasChar {
		query += productionPlanSeriyaEndsWithClause(argN)
		args = append(args, char)
		argN++
	}
	if len(modelIDs) > 0 {
		query += fmt.Sprintf(` AND id = ANY($%d)`, argN)
		args = append(args, intSliceParam(modelIDs))
	}
	query += ` ORDER BY COALESCE(modeli, ''), id`

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var item planTemplateItem
		if err := rows.Scan(&item.ItemKey, &item.Label, &item.OdooCode, &item.SeriyaRaqami, &item.FullNameUz, &item.Rangi); err != nil {
			return items, err
		}
		item.AllowOverplan = true
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) productionPlanTemplateFinPress(componentIDs []int) ([]planTemplateItem, error) {
	query := `
		SELECT c.id, COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''), COALESCE(c.odoo_code, ''), '', COALESCE(c.full_name_uz, ''), ''
		FROM production.fin_press_components fpc
		INNER JOIN production.components c ON c.id = fpc.component_id
		WHERE c.status = true`
	args := []any{}
	if len(componentIDs) > 0 {
		query += ` AND c.id = ANY($1)`
		args = append(args, intSliceParam(componentIDs))
	}
	query += ` ORDER BY COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''), c.id`

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlanTemplateItems(rows)
}

func (r *Repo) productionPlanTemplateRadiator(componentIDs []int) ([]planTemplateItem, error) {
	query := `
		SELECT cr.id, COALESCE(NULLIF(cr.factory_code, ''), cr.manufacturer_code, ''), COALESCE(cr.odoo_code, ''), COALESCE(rc.seriya_raqami, ''), COALESCE(cr.full_name_uz, ''), ''
		FROM production.radiator_components rc
		INNER JOIN production.components cr ON cr.id = rc.component_id
		WHERE cr.status = true`
	args := []any{}
	if len(componentIDs) > 0 {
		query += ` AND cr.id = ANY($1)`
		args = append(args, intSliceParam(componentIDs))
	}
	query += ` ORDER BY COALESCE(NULLIF(cr.factory_code, ''), cr.manufacturer_code, ''), cr.id`

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlanTemplateItems(rows)
}

func (r *Repo) productionPlanTemplateKlapan(componentIDs []int) ([]planTemplateItem, error) {
	query := `
		SELECT c.id, COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''), COALESCE(c.odoo_code, ''), COALESCE(kc.seriya_raqami, ''), COALESCE(c.full_name_uz, ''), ''
		FROM production.klapan_components kc
		INNER JOIN production.components c ON c.id = kc.component_id
		WHERE c.status = true`
	args := []any{}
	if len(componentIDs) > 0 {
		query += ` AND c.id = ANY($1)`
		args = append(args, intSliceParam(componentIDs))
	}
	query += ` ORDER BY COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''), c.id`

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlanTemplateItems(rows)
}

func (r *Repo) productionPlanTemplateEshik(modelIDs []int) ([]planTemplateItem, error) {
	query := `
		SELECT m.id, m.model_name, '', '', '', ''
		FROM production.eshik_models m`
	args := []any{}
	if len(modelIDs) > 0 {
		query += ` WHERE m.id = ANY($1)`
		args = append(args, intSliceParam(modelIDs))
	}
	query += ` ORDER BY m.model_name, m.id`

	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanPlanTemplateItems(rows)
}

func scanPlanTemplateItems(rows *sql.Rows) ([]planTemplateItem, error) {
	items := []planTemplateItem{}
	for rows.Next() {
		var item planTemplateItem
		if err := rows.Scan(&item.ItemKey, &item.Label, &item.OdooCode, &item.SeriyaRaqami, &item.FullNameUz, &item.Rangi); err != nil {
			return items, err
		}
		item.AllowOverplan = true
		items = append(items, item)
	}
	return items, rows.Err()
}

func (r *Repo) ProductionPlanExportWorkbook(yearMonth string, lineIDs []int, includeActual bool) ([]byte, error) {
	yearMonth = strings.TrimSpace(yearMonth)
	if len(yearMonth) != 7 {
		return nil, errors.New("oy formati: YYYY-MM")
	}
	year, _ := strconv.Atoi(yearMonth[:4])
	month, _ := strconv.Atoi(yearMonth[5:7])
	if month < 1 || month > 12 {
		return nil, errors.New("noto'g'ri oy")
	}
	daysInMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local).Day()

	if len(lineIDs) == 0 {
		lineIDs = append(append([]int{}, ProductionPlanProductLineIDs...), ProductionPlanAuxiliaryLineIDs...)
	}

	f := excelize.NewFile()
	defer f.Close()
	_ = f.SetSheetName("Sheet1", "Sozlamalar")

	styles, err := newPlanExcelStyles(f)
	if err != nil {
		return nil, err
	}

	overplanSettings := map[string]map[int]bool{}
	allTemplateItems := map[int][]planTemplateItem{}

	for _, lineID := range lineIDs {
		lineName, err := r.lineNameByID(lineID)
		if err != nil {
			return nil, err
		}
		templateItems, err := r.productionPlanTemplateItems(lineID, nil, nil)
		if err != nil {
			return nil, err
		}
		allTemplateItems[lineID] = templateItems

		planMap, err := r.productionPlanMonthMap(yearMonth, lineID)
		if err != nil {
			return nil, err
		}
		sheet := sanitizeSheetName(lineName)
		overplanSettings[sheet] = map[int]bool{}
		for key, cells := range planMap {
			for _, cell := range cells {
				overplanSettings[sheet][key] = cell.AllowOverplan
			}
		}

		if err := r.writePlanLineSheet(f, styles, lineID, lineName, yearMonth, daysInMonth, templateItems, planMap, includeActual); err != nil {
			return nil, err
		}
	}

	r.writePlanSozlamalarSheetStyled(f, styles, lineIDs, allTemplateItems, overplanSettings)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type planMonthCell struct {
	Shift1Planned int
	Shift2Planned int
	AllowOverplan bool
}

func (r *Repo) productionPlanMonthMap(yearMonth string, lineID int) (map[int]map[string]planMonthCell, error) {
	start := yearMonth + "-01"
	rows, err := r.store.db.Query(`
		SELECT plan_date::text,
			COALESCE(model_id, component_id),
			shift_no, planned_qty, allow_overplan
		FROM production.daily_plan_items
		WHERE line_id = $1
		  AND plan_date >= $2::date
		  AND plan_date < ($2::date + INTERVAL '1 month')`,
		lineID, start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := map[int]map[string]planMonthCell{}
	componentKeys := []int{}
	type rawCell struct {
		date    string
		key     int
		shiftNo int
		qty     int
		allow   bool
	}
	raw := []rawCell{}
	for rows.Next() {
		var dateStr string
		var key, shiftNo, qty int
		var allow bool
		if err := rows.Scan(&dateStr, &key, &shiftNo, &qty, &allow); err != nil {
			return result, err
		}
		raw = append(raw, rawCell{date: dateStr, key: key, shiftNo: shiftNo, qty: qty, allow: allow})
		if lineID == EshikLineID && key > 0 {
			componentKeys = append(componentKeys, key)
		}
	}
	if err := rows.Err(); err != nil {
		return result, err
	}

	modelByComponent := map[int]int{}
	if lineID == EshikLineID && len(componentKeys) > 0 {
		modelByComponent, err = r.EshikModelIDsByComponentIDs(componentKeys)
		if err != nil {
			return nil, err
		}
	}

	for _, cell := range raw {
		key := cell.key
		if lineID == EshikLineID {
			if modelID := modelByComponent[cell.key]; modelID > 0 {
				key = modelID
			}
		}
		if _, ok := result[key]; !ok {
			result[key] = map[string]planMonthCell{}
		}
		existing := result[key][cell.date]
		existing.AllowOverplan = existing.AllowOverplan || cell.allow
		if cell.shiftNo == 2 {
			if cell.qty > existing.Shift2Planned {
				existing.Shift2Planned = cell.qty
			}
		} else {
			if cell.qty > existing.Shift1Planned {
				existing.Shift1Planned = cell.qty
			}
		}
		result[key][cell.date] = existing
	}
	return result, nil
}

func (r *Repo) lineNameByID(lineID int) (string, error) {
	var name string
	err := r.store.db.QueryRow(`SELECT name FROM lines.lines_list WHERE line_id = $1`, lineID).Scan(&name)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("liniya topilmadi (id=%d)", lineID)
		}
		return "", err
	}
	return name, nil
}

func (r *Repo) productionPlanItemLabel(lineID, itemKey int) (string, error) {
	if IsProductionPlanProductLine(lineID) {
		var label string
		err := r.store.db.QueryRow(`SELECT COALESCE(modeli, '') FROM production.models WHERE id = $1`, itemKey).Scan(&label)
		return label, err
	}
	var label string
	err := r.store.db.QueryRow(`
		SELECT COALESCE(NULLIF(factory_code, ''), manufacturer_code, '')
		FROM production.components WHERE id = $1`, itemKey).Scan(&label)
	return label, err
}

func sanitizeSheetName(name string) string {
	replacer := strings.NewReplacer("/", "-", "\\", "-", ":", "-", "?", "", "*", "", "[", "(", "]", ")")
	out := replacer.Replace(strings.TrimSpace(name))
	if out == "" {
		return "Liniya"
	}
	if len(out) > 31 {
		return out[:31]
	}
	return out
}

func (r *Repo) writePlanSozlamalarSheetStyled(
	f *excelize.File,
	styles *planExcelStyles,
	lineIDs []int,
	allItems map[int][]planTemplateItem,
	settings map[string]map[int]bool,
) {
	headers := []string{"line_id", "item_id", "label", "Qo'shimcha (1/0)"}
	for i, h := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue("Sozlamalar", cell, h)
		_ = f.SetCellStyle("Sozlamalar", cell, cell, styles.sozHdr)
	}
	row := 2
	for _, lineID := range lineIDs {
		lineName, _ := r.lineNameByID(lineID)
		sheet := sanitizeSheetName(lineName)
		items := allItems[lineID]
		for _, item := range items {
			allow := true
			if sheetSettings, ok := settings[sheet]; ok {
				if v, ok := sheetSettings[item.ItemKey]; ok {
					allow = v
				}
			}
			val := 0
			if allow {
				val = 1
			}
			_ = f.SetCellValue("Sozlamalar", fmt.Sprintf("A%d", row), lineID)
			_ = f.SetCellValue("Sozlamalar", fmt.Sprintf("B%d", row), item.ItemKey)
			_ = f.SetCellValue("Sozlamalar", fmt.Sprintf("C%d", row), item.Label)
			_ = f.SetCellValue("Sozlamalar", fmt.Sprintf("D%d", row), val)
			row++
		}
	}
}

func (r *Repo) ProductionPlanImportWorkbook(data []byte, yearMonth string) (*DailyPlanImportResult, error) {
	yearMonth = strings.TrimSpace(yearMonth)
	if len(yearMonth) != 7 {
		return nil, errors.New("oy formati: YYYY-MM")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	result := &DailyPlanImportResult{Errors: []DailyPlanImportError{}}
	overplanByLine := r.readPlanSozlamalarSheet(f)

	cells := []PlanUpsertCell{}
	for _, sheet := range f.GetSheetList() {
		if sheet == "Sozlamalar" || strings.HasPrefix(sheet, "Ref_") {
			continue
		}
		lineID, err := f.GetCellValue(sheet, "B2")
		if err != nil {
			continue
		}
		lineIDInt, _ := strconv.Atoi(strings.TrimSpace(lineID))
		if lineIDInt <= 0 {
			result.Errors = append(result.Errors, DailyPlanImportError{Sheet: sheet, Message: "line_id topilmadi"})
			continue
		}

		rows, err := f.GetRows(sheet)
		if err != nil || len(rows) < 4 {
			continue
		}
		dateColumns := map[int]string{}
		for colNum := 3; ; colNum += 4 {
			planCol, _ := excelize.ColumnNumberToName(colNum)
			dateRaw, err := f.GetCellValue(sheet, planCol+"2")
			if err != nil || strings.TrimSpace(dateRaw) == "" {
				break
			}
			dateRaw = strings.TrimSpace(dateRaw)
			if strings.HasPrefix(dateRaw, yearMonth) {
				dateColumns[colNum] = dateRaw
			}
		}

		for rowNum := 4; ; rowNum++ {
			label, err := f.GetCellValue(sheet, fmt.Sprintf("B%d", rowNum))
			if err != nil {
				break
			}
			label = strings.TrimSpace(label)
			itemKeyRaw, _ := f.GetCellValue(sheet, fmt.Sprintf("A%d", rowNum))
			itemKey, _ := strconv.Atoi(strings.TrimSpace(itemKeyRaw))
			if label == "" && itemKey <= 0 {
				if rowNum > 4+planTemplateDataRows+50 {
					break
				}
				continue
			}
			if itemKey <= 0 {
				itemKey, err = r.productionPlanResolveLabel(lineIDInt, label)
				if err != nil {
					result.Errors = append(result.Errors, DailyPlanImportError{
						Sheet: sheet, Row: rowNum, Message: err.Error(),
					})
					continue
				}
			}
			if err := r.productionPlanValidateItemKey(lineIDInt, itemKey, label); err != nil {
				result.Errors = append(result.Errors, DailyPlanImportError{
					Sheet: sheet, Row: rowNum, Message: err.Error(),
				})
				continue
			}
			allow := true
			if lineSettings, ok := overplanByLine[lineIDInt]; ok {
				if v, ok := lineSettings[itemKey]; ok {
					allow = v
				}
			}
			hasQty := false
			for colNum, dateStr := range dateColumns {
				if productionPlanIsPastDate(dateStr) {
					result.SkippedRows++
					continue
				}
				status, _ := r.productionPlanDayStatus(dateStr, lineIDInt)
				if status == PlanStatusLocked {
					result.SkippedRows++
					continue
				}
				for _, shiftSpec := range []struct {
					offset  int
					shiftNo int
				}{
					{0, 1},
					{2, 2},
				} {
					planCol, _ := excelize.ColumnNumberToName(colNum + shiftSpec.offset)
					qtyRaw, _ := f.GetCellValue(sheet, fmt.Sprintf("%s%d", planCol, rowNum))
					qty, _ := strconv.Atoi(strings.TrimSpace(qtyRaw))
					if qty <= 0 {
						continue
					}
					hasQty = true
					cell := PlanUpsertCell{
						PlanDate: dateStr, LineID: lineIDInt, ShiftNo: shiftSpec.shiftNo,
						PlannedQty: qty, AllowOverplan: allow,
					}
					if IsProductionPlanProductLine(lineIDInt) {
						cell.ModelID = itemKey
					} else {
						cell.ComponentID = itemKey
					}
					cells = append(cells, cell)
					result.ImportedRows++
				}
			}
			if label != "" && !hasQty {
				continue
			}
			if rowNum > 4+planTemplateDataRows+100 {
				break
			}
		}
	}

	if len(result.Errors) > 0 {
		return result, nil
	}
	if err := r.ProductionPlanUpsertCells(cells); err != nil {
		return nil, err
	}
	return result, nil
}

func (r *Repo) productionPlanValidateItemKey(lineID, itemKey int, label string) error {
	if IsProductionPlanProductLine(lineID) {
		char, hasChar := productionPlanModelSeriyaSuffix(lineID)
		query := `
			SELECT COALESCE(modeli, '') FROM production.models
			WHERE id = $1 AND status = true AND deleted = false`
		args := []any{itemKey}
		if hasChar {
			query += productionPlanSeriyaEndsWithClause(2)
			args = append(args, char)
		}
		var dbLabel string
		err := r.store.db.QueryRow(query, args...).Scan(&dbLabel)
		if err != nil {
			return errors.New("model topilmadi yoki bu liniya uchun mos emas")
		}
		if label != "" && !strings.EqualFold(strings.TrimSpace(dbLabel), strings.TrimSpace(label)) {
			return fmt.Errorf("model nomi mos emas: %s", dbLabel)
		}
		return nil
	}
	var dbLabel string
	var err error
	switch lineID {
	case FinPressLineID:
		err = r.store.db.QueryRow(`
			SELECT COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, '')
			FROM production.fin_press_components fpc
			INNER JOIN production.components c ON c.id = fpc.component_id
			WHERE c.id = $1 AND c.status = true`, itemKey).Scan(&dbLabel)
	case RadiatorLineID:
		err = r.store.db.QueryRow(`
			SELECT COALESCE(NULLIF(cr.factory_code, ''), cr.manufacturer_code, '')
			FROM production.radiator_components rc
			INNER JOIN production.components cr ON cr.id = rc.component_id
			WHERE cr.id = $1 AND cr.status = true`, itemKey).Scan(&dbLabel)
	case KlapanLineID:
		err = r.store.db.QueryRow(`
			SELECT COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, '')
			FROM production.klapan_components kc
			INNER JOIN production.components c ON c.id = kc.component_id
			WHERE c.id = $1 AND c.status = true`, itemKey).Scan(&dbLabel)
	case EshikLineID:
		err = r.store.db.QueryRow(`
			SELECT model_name
			FROM production.eshik_models
			WHERE id = $1`, itemKey).Scan(&dbLabel)
		if err != nil {
			return errors.New("eshik modeli topilmadi")
		}
		if label != "" && !strings.EqualFold(strings.TrimSpace(dbLabel), strings.TrimSpace(label)) {
			return fmt.Errorf("model nomi mos emas: %s", dbLabel)
		}
		return nil
	default:
		return fmt.Errorf("noto'g'ri line_id: %d", lineID)
	}
	if err != nil {
		return errors.New("komponent topilmadi yoki bu liniya ro'yxatida yo'q")
	}
	if label != "" && !strings.EqualFold(strings.TrimSpace(dbLabel), strings.TrimSpace(label)) {
		return fmt.Errorf("factory_code mos emas: %s", dbLabel)
	}
	return nil
}

func (r *Repo) readPlanSozlamalarSheet(f *excelize.File) map[int]map[int]bool {
	result := map[int]map[int]bool{}
	rows, err := f.GetRows("Sozlamalar")
	if err != nil || len(rows) < 2 {
		return result
	}
	for i := 1; i < len(rows); i++ {
		row := rows[i]
		if len(row) < 4 {
			continue
		}
		lineID, _ := strconv.Atoi(strings.TrimSpace(row[0]))
		itemKey, _ := strconv.Atoi(strings.TrimSpace(row[1]))
		allowRaw := strings.TrimSpace(row[3])
		if lineID <= 0 || itemKey <= 0 {
			continue
		}
		allow := allowRaw == "1" || strings.EqualFold(allowRaw, "ha") || strings.EqualFold(allowRaw, "yes")
		if _, ok := result[lineID]; !ok {
			result[lineID] = map[int]bool{}
		}
		result[lineID][itemKey] = allow
	}
	return result
}

func (r *Repo) ProductionPlanAllowedModelIDs(planDate string, lineID int) ([]int, error) {
	planDate = planDateOrToday(planDate)
	current, err := r.ProductionCurrentShift(time.Time{})
	if err != nil {
		return nil, err
	}
	shiftNo := 0
	if current.PlanDate == planDate && current.ShiftNo > 0 {
		shiftNo = current.ShiftNo
	}
	rows, err := r.store.db.Query(`
		SELECT DISTINCT model_id
		FROM production.daily_plan_items
		WHERE plan_date = $1::date AND line_id = $2 AND model_id IS NOT NULL
		  AND ($3::int = 0 OR shift_no = $3)
		ORDER BY model_id`, planDate, lineID, shiftNo)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type PlanCatalogItem struct {
	ItemKey       int    `json:"item_key"`
	Label         string `json:"label"`
	OdooCode      string `json:"odoo_code"`
	SeriyaRaqami  string `json:"seriya_raqami"`
	FullNameUz    string `json:"full_name_uz"`
	Rangi         string `json:"rangi"`
	AllowOverplan bool   `json:"allow_overplan"`
}

type PlanMonthShiftCell struct {
	PlannedQty int `json:"planned_qty"`
	ActualQty  int `json:"actual_qty"`
}

type PlanMonthDayCell struct {
	Shift1 PlanMonthShiftCell `json:"shift1"`
	Shift2 PlanMonthShiftCell `json:"shift2"`
}

type PlanMonthGridRow struct {
	ItemKey       int                         `json:"item_key"`
	ModelID       int                         `json:"model_id"`
	ComponentID   int                         `json:"component_id"`
	Label         string                      `json:"label"`
	Rangi         string                      `json:"rangi"`
	AllowOverplan bool                        `json:"allow_overplan"`
	Days          map[string]PlanMonthDayCell `json:"days"`
}

type PlanMonthResponse struct {
	YearMonth   string             `json:"year_month"`
	LineID      int                `json:"line_id"`
	LineName    string             `json:"line_name"`
	Dates       []string           `json:"dates"`
	DayStatuses map[string]string  `json:"day_statuses"`
	Rows        []PlanMonthGridRow `json:"rows"`
	Catalog     []PlanCatalogItem  `json:"catalog"`
}

func (r *Repo) ProductionPlanGetMonth(yearMonth string, lineID int, includeActual bool) (*PlanMonthResponse, error) {
	yearMonth = strings.TrimSpace(yearMonth)
	if len(yearMonth) != 7 {
		return nil, errors.New("oy formati: YYYY-MM")
	}
	year, _ := strconv.Atoi(yearMonth[:4])
	month, _ := strconv.Atoi(yearMonth[5:7])
	if month < 1 || month > 12 {
		return nil, errors.New("noto'g'ri oy")
	}
	daysInMonth := time.Date(year, time.Month(month+1), 0, 0, 0, 0, 0, time.Local).Day()

	lineName, err := r.lineNameByID(lineID)
	if err != nil {
		return nil, err
	}

	resp := &PlanMonthResponse{
		YearMonth:   yearMonth,
		LineID:      lineID,
		LineName:    lineName,
		Dates:       make([]string, 0, daysInMonth),
		DayStatuses: map[string]string{},
		Rows:        []PlanMonthGridRow{},
		Catalog:     []PlanCatalogItem{},
	}

	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%s-%02d", yearMonth, day)
		resp.Dates = append(resp.Dates, dateStr)
		status, err := r.productionPlanEffectiveDayStatus(dateStr, lineID)
		if err != nil {
			return nil, err
		}
		resp.DayStatuses[dateStr] = status
	}

	catalogItems, err := r.productionPlanTemplateItems(lineID, nil, nil)
	if err != nil {
		return nil, err
	}
	for _, item := range catalogItems {
		resp.Catalog = append(resp.Catalog, PlanCatalogItem{
			ItemKey: item.ItemKey, Label: item.Label, OdooCode: item.OdooCode,
			SeriyaRaqami: item.SeriyaRaqami, FullNameUz: item.FullNameUz, Rangi: item.Rangi, AllowOverplan: item.AllowOverplan,
		})
	}

	start := yearMonth + "-01"
	endDate := fmt.Sprintf("%s-%02d", yearMonth, daysInMonth)
	var actuals map[string]int
	if includeActual {
		actuals, err = r.productionPlanActualQtyBatch(start, endDate, []int{lineID})
		if err != nil {
			return nil, err
		}
	}
	rows, err := r.store.db.Query(`
		SELECT i.plan_date::text,
			COALESCE(i.model_id, 0), COALESCE(i.component_id, 0),
			COALESCE(NULLIF(m.modeli, ''), NULLIF(c.factory_code, ''), ''),
			COALESCE(m.rangi, ''),
			i.shift_no, i.planned_qty, i.allow_overplan
		FROM production.daily_plan_items i
		LEFT JOIN production.models m ON m.id = i.model_id
		LEFT JOIN production.components c ON c.id = i.component_id
		WHERE i.line_id = $1
		  AND i.plan_date >= $2::date
		  AND i.plan_date < ($2::date + INTERVAL '1 month')
		ORDER BY COALESCE(m.modeli, c.factory_code, ''), COALESCE(i.model_id, i.component_id), i.plan_date, i.shift_no`,
		lineID, start,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	rowMap := map[int]*PlanMonthGridRow{}
	rowOrder := []int{}
	for rows.Next() {
		var dateStr, label, rangi string
		var modelID, componentID, shiftNo, plannedQty int
		var allow bool
		if err := rows.Scan(&dateStr, &modelID, &componentID, &label, &rangi, &shiftNo, &plannedQty, &allow); err != nil {
			return nil, err
		}
		itemKey := modelID
		if itemKey <= 0 {
			itemKey = componentID
		}
		if itemKey <= 0 {
			continue
		}
		if shiftNo != 2 {
			shiftNo = 1
		}
		gridRow, ok := rowMap[itemKey]
		if !ok {
			gridRow = &PlanMonthGridRow{
				ItemKey: itemKey, ModelID: modelID, ComponentID: componentID,
				Label: label, Rangi: rangi, AllowOverplan: allow, Days: map[string]PlanMonthDayCell{},
			}
			rowMap[itemKey] = gridRow
			rowOrder = append(rowOrder, itemKey)
		}
		cell := gridRow.Days[dateStr]
		shiftCell := PlanMonthShiftCell{PlannedQty: plannedQty}
		if includeActual {
			shiftCell.ActualQty = productionPlanActualQtyFromMap(dateStr, lineID, shiftNo, modelID, componentID, actuals)
		}
		if shiftNo == 2 {
			cell.Shift2 = shiftCell
		} else {
			cell.Shift1 = shiftCell
		}
		gridRow.Days[dateStr] = cell
		gridRow.AllowOverplan = allow
		if gridRow.Label == "" {
			gridRow.Label = label
		}
		if gridRow.Rangi == "" {
			gridRow.Rangi = rangi
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	for _, key := range rowOrder {
		resp.Rows = append(resp.Rows, *rowMap[key])
	}
	if lineID == EshikLineID {
		resp.Rows, err = r.groupEshikPlanMonthRows(resp.Rows)
		if err != nil {
			return nil, err
		}
	}
	return resp, nil
}

func eshikPairMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func eshikPairMax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// groupEshikDailyPlanItems merges freeze+ref into one model row.
// Pair unit: planned = max(parts), actual = min(parts).
func (r *Repo) groupEshikDailyPlanItems(items []DailyPlanItemRow) ([]DailyPlanItemRow, error) {
	if len(items) == 0 {
		return items, nil
	}
	componentIDs := make([]int, 0, len(items))
	for _, item := range items {
		if item.ComponentID > 0 {
			componentIDs = append(componentIDs, item.ComponentID)
		}
	}
	modelByComponent, err := r.EshikModelIDsByComponentIDs(componentIDs)
	if err != nil {
		return nil, err
	}
	modelIDs := make([]int, 0, len(modelByComponent))
	seenModel := map[int]struct{}{}
	for _, modelID := range modelByComponent {
		if _, ok := seenModel[modelID]; ok {
			continue
		}
		seenModel[modelID] = struct{}{}
		modelIDs = append(modelIDs, modelID)
	}
	names, err := r.EshikModelNamesByIDs(modelIDs)
	if err != nil {
		return nil, err
	}

	type groupKey struct {
		PlanDate string
		ShiftNo  int
		ModelID  int
	}
	type groupState struct {
		row      DailyPlanItemRow
		partSeen int
	}
	grouped := map[groupKey]*groupState{}
	order := []groupKey{}
	for _, item := range items {
		modelID := modelByComponent[item.ComponentID]
		if modelID <= 0 {
			// Orphan component plan row — keep as-is
			key := groupKey{PlanDate: item.PlanDate, ShiftNo: item.ShiftNo, ModelID: -item.ComponentID}
			copyItem := item
			grouped[key] = &groupState{row: copyItem, partSeen: 1}
			order = append(order, key)
			continue
		}
		key := groupKey{PlanDate: item.PlanDate, ShiftNo: item.ShiftNo, ModelID: modelID}
		if existing, ok := grouped[key]; ok {
			existing.row.PlannedQty = eshikPairMax(existing.row.PlannedQty, item.PlannedQty)
			existing.row.ActualQty = eshikPairMin(existing.row.ActualQty, item.ActualQty)
			existing.row.AllowOverplan = existing.row.AllowOverplan || item.AllowOverplan
			existing.partSeen++
			continue
		}
		copyItem := item
		copyItem.ModelID = 0
		copyItem.ComponentID = 0
		copyItem.ItemKey = modelID
		copyItem.Label = names[modelID]
		if copyItem.Label == "" {
			copyItem.Label = item.Label
		}
		copyItem.Modeli = copyItem.Label
		grouped[key] = &groupState{row: copyItem, partSeen: 1}
		order = append(order, key)
	}

	out := make([]DailyPlanItemRow, 0, len(order))
	for _, key := range order {
		state := grouped[key]
		row := state.row
		// Single mapped part without pair: actual cannot form a complete unit.
		if key.ModelID > 0 && state.partSeen < 2 {
			row.ActualQty = 0
		}
		row.RemainingQty = row.PlannedQty - row.ActualQty
		row.CompletionPct = 0
		if row.PlannedQty > 0 {
			row.CompletionPct = row.ActualQty * 100 / row.PlannedQty
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *Repo) groupEshikPlanMonthRows(rows []PlanMonthGridRow) ([]PlanMonthGridRow, error) {
	if len(rows) == 0 {
		return rows, nil
	}
	componentIDs := make([]int, 0, len(rows))
	for _, row := range rows {
		if row.ComponentID > 0 {
			componentIDs = append(componentIDs, row.ComponentID)
		}
	}
	modelByComponent, err := r.EshikModelIDsByComponentIDs(componentIDs)
	if err != nil {
		return nil, err
	}
	modelIDs := make([]int, 0, len(modelByComponent))
	seenModel := map[int]struct{}{}
	for _, modelID := range modelByComponent {
		if _, ok := seenModel[modelID]; ok {
			continue
		}
		seenModel[modelID] = struct{}{}
		modelIDs = append(modelIDs, modelID)
	}
	names, err := r.EshikModelNamesByIDs(modelIDs)
	if err != nil {
		return nil, err
	}

	type dayAccum struct {
		shift1Planned, shift2Planned int
		shift1Actual, shift2Actual   int
		shift1ActualSet, shift2ActualSet bool
		partsSeen                        int
	}
	// Collect parts per model
	partsByModel := map[int][]int{} // modelID -> componentIDs
	allowByModel := map[int]bool{}
	for _, row := range rows {
		modelID := modelByComponent[row.ComponentID]
		if modelID <= 0 {
			continue
		}
		partsByModel[modelID] = append(partsByModel[modelID], row.ComponentID)
		allowByModel[modelID] = allowByModel[modelID] || row.AllowOverplan
	}

	// Pair unit: planned = max(parts), actual = min(parts).
	accum := map[int]map[string]*dayAccum{} // modelID -> date -> accum

	for _, row := range rows {
		modelID := modelByComponent[row.ComponentID]
		if modelID <= 0 {
			continue
		}
		if _, ok := accum[modelID]; !ok {
			accum[modelID] = map[string]*dayAccum{}
		}
		for date, cell := range row.Days {
			day := accum[modelID][date]
			if day == nil {
				day = &dayAccum{}
				accum[modelID][date] = day
			}
			day.partsSeen++
			if cell.Shift1.PlannedQty > day.shift1Planned {
				day.shift1Planned = cell.Shift1.PlannedQty
			}
			if cell.Shift2.PlannedQty > day.shift2Planned {
				day.shift2Planned = cell.Shift2.PlannedQty
			}
			if !day.shift1ActualSet {
				day.shift1Actual = cell.Shift1.ActualQty
				day.shift1ActualSet = true
			} else {
				day.shift1Actual = eshikPairMin(day.shift1Actual, cell.Shift1.ActualQty)
			}
			if !day.shift2ActualSet {
				day.shift2Actual = cell.Shift2.ActualQty
				day.shift2ActualSet = true
			} else {
				day.shift2Actual = eshikPairMin(day.shift2Actual, cell.Shift2.ActualQty)
			}
		}
	}

	out := make([]PlanMonthGridRow, 0, len(partsByModel))
	modelOrder := make([]int, 0, len(partsByModel))
	for modelID := range partsByModel {
		modelOrder = append(modelOrder, modelID)
	}
	// Stable-ish: by name
	for i := 0; i < len(modelOrder); i++ {
		for j := i + 1; j < len(modelOrder); j++ {
			if names[modelOrder[j]] < names[modelOrder[i]] {
				modelOrder[i], modelOrder[j] = modelOrder[j], modelOrder[i]
			}
		}
	}

	for _, modelID := range modelOrder {
		gridRow := PlanMonthGridRow{
			ItemKey:       modelID,
			ModelID:       0,
			ComponentID:   modelID, // UI aux lines use component_id/item_key as catalog key
			Label:         names[modelID],
			AllowOverplan: allowByModel[modelID],
			Days:          map[string]PlanMonthDayCell{},
		}
		for date, day := range accum[modelID] {
			shift1Actual := day.shift1Actual
			shift2Actual := day.shift2Actual
			if day.partsSeen < 2 {
				shift1Actual = 0
				shift2Actual = 0
			}
			gridRow.Days[date] = PlanMonthDayCell{
				Shift1: PlanMonthShiftCell{PlannedQty: day.shift1Planned, ActualQty: shift1Actual},
				Shift2: PlanMonthShiftCell{PlannedQty: day.shift2Planned, ActualQty: shift2Actual},
			}
		}
		out = append(out, gridRow)
	}

	// Keep orphan component rows that could not be mapped
	for _, row := range rows {
		if modelByComponent[row.ComponentID] > 0 {
			continue
		}
		out = append(out, row)
	}
	return out, nil
}

func (r *Repo) EshikModelNamesByIDs(ids []int) (map[int]string, error) {
	result := map[int]string{}
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := r.store.db.Query(`
		SELECT id, model_name
		FROM production.eshik_models
		WHERE id = ANY($1)`, intSliceParam(ids))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return result, err
		}
		result[id] = name
	}
	return result, rows.Err()
}
