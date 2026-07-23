package store

import (
	"database/sql"
	"fmt"
	"strings"
)

func productionPlanUpsertCellsBatchTx(tx *sql.Tx, cells []PlanUpsertCell) error {
	if len(cells) == 0 {
		return nil
	}
	for start := 0; start < len(cells); start += bulkBatchSize {
		end := start + bulkBatchSize
		if end > len(cells) {
			end = len(cells)
		}
		batch := cells[start:end]
		if err := productionPlanUpsertModelCellsBatchTx(tx, batch); err != nil {
			return err
		}
		if err := productionPlanUpsertComponentCellsBatchTx(tx, batch); err != nil {
			return err
		}
	}
	return nil
}

func normalizePlanUpsertShiftNo(cell *PlanUpsertCell) {
	if cell.ShiftNo != 2 {
		cell.ShiftNo = 1
	}
}

func productionPlanUpsertModelCellsBatchTx(tx *sql.Tx, cells []PlanUpsertCell) error {
	modelCells := make([]PlanUpsertCell, 0, len(cells))
	for _, cell := range cells {
		if cell.ModelID > 0 {
			normalizePlanUpsertShiftNo(&cell)
			modelCells = append(modelCells, cell)
		}
	}
	if len(modelCells) == 0 {
		return nil
	}

	var sb strings.Builder
	args := make([]any, 0, len(modelCells)*6)
	sb.WriteString(`INSERT INTO production.daily_plan_items
		(plan_date, line_id, model_id, shift_no, planned_qty, allow_overplan) VALUES `)
	for i, cell := range modelCells {
		if i > 0 {
			sb.WriteString(",")
		}
		n := i*6 + 1
		sb.WriteString(fmt.Sprintf("($%d::date,$%d,$%d,$%d,$%d,$%d)", n, n+1, n+2, n+3, n+4, n+5))
		args = append(args, cell.PlanDate, cell.LineID, cell.ModelID, cell.ShiftNo, cell.PlannedQty, cell.AllowOverplan)
	}
	sb.WriteString(`
		ON CONFLICT (plan_date, line_id, model_id, shift_no) WHERE model_id IS NOT NULL
		DO UPDATE SET planned_qty = EXCLUDED.planned_qty, allow_overplan = EXCLUDED.allow_overplan`)
	_, err := tx.Exec(sb.String(), args...)
	return err
}

func productionPlanUpsertComponentCellsBatchTx(tx *sql.Tx, cells []PlanUpsertCell) error {
	componentCells := make([]PlanUpsertCell, 0, len(cells))
	for _, cell := range cells {
		if cell.ComponentID > 0 {
			normalizePlanUpsertShiftNo(&cell)
			componentCells = append(componentCells, cell)
		}
	}
	if len(componentCells) == 0 {
		return nil
	}

	var sb strings.Builder
	args := make([]any, 0, len(componentCells)*6)
	sb.WriteString(`INSERT INTO production.daily_plan_items
		(plan_date, line_id, component_id, shift_no, planned_qty, allow_overplan) VALUES `)
	for i, cell := range componentCells {
		if i > 0 {
			sb.WriteString(",")
		}
		n := i*6 + 1
		sb.WriteString(fmt.Sprintf("($%d::date,$%d,$%d,$%d,$%d,$%d)", n, n+1, n+2, n+3, n+4, n+5))
		args = append(args, cell.PlanDate, cell.LineID, cell.ComponentID, cell.ShiftNo, cell.PlannedQty, cell.AllowOverplan)
	}
	sb.WriteString(`
		ON CONFLICT (plan_date, line_id, component_id, shift_no) WHERE component_id IS NOT NULL
		DO UPDATE SET planned_qty = EXCLUDED.planned_qty, allow_overplan = EXCLUDED.allow_overplan`)
	_, err := tx.Exec(sb.String(), args...)
	return err
}
