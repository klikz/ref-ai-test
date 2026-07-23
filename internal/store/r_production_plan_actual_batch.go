package store

import (
	"fmt"
	"strings"
	"time"
)

// productionWallClockLocal reinterprets a timestamp in time.Local.
//
// lib/pq returns `timestamp without time zone` columns as time.Time labelled
// UTC while preserving the stored wall clock. Shift windows are built in
// time.Local, so comparing the two as instants skews the result by the local
// UTC offset. Re-labelling the wall clock as Local keeps the comparison correct.
func productionWallClockLocal(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), time.Local)
}

func productionPlanActualKey(planDate string, lineID, shiftNo, modelID, componentID int) string {
	if modelID > 0 {
		return fmt.Sprintf("%s|%d|s:%d|m:%d", planDate, lineID, shiftNo, modelID)
	}
	return fmt.Sprintf("%s|%d|s:%d|c:%d", planDate, lineID, shiftNo, componentID)
}

func (r *Repo) productionPlanActualQtyForDay(planDate string, lineID int) (map[string]int, error) {
	return r.productionPlanActualQtyBatch(planDate, planDate, []int{lineID})
}

func (r *Repo) productionPlanActualQtyBatch(dateFrom, dateTo string, lineIDs []int) (map[string]int, error) {
	result := map[string]int{}
	if strings.TrimSpace(dateFrom) == "" || strings.TrimSpace(dateTo) == "" {
		return result, nil
	}

	settings, err := r.ProductionShiftSettingsGet()
	if err != nil {
		return result, err
	}

	fromDay, err := productionPlanParseDate(dateFrom)
	if err != nil {
		return result, err
	}
	toDay, err := productionPlanParseDate(dateTo)
	if err != nil {
		return result, err
	}

	type shiftWindow struct {
		planDate string
		shiftNo  int
		start    time.Time
		end      time.Time
	}
	windows := make([]shiftWindow, 0)
	for day := fromDay; !day.After(toDay); day = day.AddDate(0, 0, 1) {
		planDate := day.Format("2006-01-02")
		for _, shiftNo := range []int{1, 2} {
			start, end, err := settings.ShiftWindow(planDate, shiftNo)
			if err != nil {
				return result, err
			}
			windows = append(windows, shiftWindow{
				planDate: planDate,
				shiftNo:  shiftNo,
				start:    start,
				end:      end,
			})
		}
	}
	if len(windows) == 0 {
		return result, nil
	}

	rangeStart := windows[0].start
	rangeEnd := windows[0].end
	for _, w := range windows[1:] {
		if w.start.Before(rangeStart) {
			rangeStart = w.start
		}
		if w.end.After(rangeEnd) {
			rangeEnd = w.end
		}
	}

	assign := func(at time.Time) (string, int, bool) {
		for _, w := range windows {
			if !at.Before(w.start) && at.Before(w.end) {
				return w.planDate, w.shiftNo, true
			}
		}
		return "", 0, false
	}

	rows, err := r.store.db.Query(`
		SELECT p.time, p.line_id, p.model_id
		FROM lines.products p
		WHERE p.time >= $1
		  AND p.time < $2
		  AND (cardinality($3::int[]) = 0 OR p.line_id = ANY($3))`,
		rangeStart, rangeEnd, intSliceParam(lineIDs),
	)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var at time.Time
		var lineID, modelID int
		if err := rows.Scan(&at, &lineID, &modelID); err != nil {
			rows.Close()
			return result, err
		}
		at = productionWallClockLocal(at)
		planDate, shiftNo, ok := assign(at)
		if !ok {
			continue
		}
		key := productionPlanActualKey(planDate, lineID, shiftNo, modelID, 0)
		result[key]++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()

	finPressIncluded := len(lineIDs) == 0
	for _, lineID := range lineIDs {
		if lineID == FinPressLineID {
			finPressIncluded = true
			break
		}
	}
	if finPressIncluded {
		rows, err = r.store.db.Query(`
			SELECT s.c_time, s.component_id, COALESCE(s.count, 0)::int
			FROM production.fin_press_print_sessions s
			WHERE s.c_time >= $1
			  AND s.c_time < $2`,
			rangeStart, rangeEnd,
		)
		if err != nil {
			return result, err
		}
		for rows.Next() {
			var at time.Time
			var componentID, count int
			if err := rows.Scan(&at, &componentID, &count); err != nil {
				rows.Close()
				return result, err
			}
			at = productionWallClockLocal(at)
			planDate, shiftNo, ok := assign(at)
			if !ok {
				continue
			}
			key := productionPlanActualKey(planDate, FinPressLineID, shiftNo, 0, componentID)
			result[key] += count
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return result, err
		}
		rows.Close()
	}

	rows, err = r.store.db.Query(`
		SELECT ap.c_time, ap.line_id, ap.component_id
		FROM lines.auxiliary_products ap
		WHERE ap.c_time >= $1
		  AND ap.c_time < $2
		  AND (cardinality($3::int[]) = 0 OR ap.line_id = ANY($3))`,
		rangeStart, rangeEnd, intSliceParam(lineIDs),
	)
	if err != nil {
		return result, err
	}
	for rows.Next() {
		var at time.Time
		var lineID, componentID int
		if err := rows.Scan(&at, &lineID, &componentID); err != nil {
			rows.Close()
			return result, err
		}
		at = productionWallClockLocal(at)
		planDate, shiftNo, ok := assign(at)
		if !ok {
			continue
		}
		key := productionPlanActualKey(planDate, lineID, shiftNo, 0, componentID)
		result[key]++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return result, err
	}
	rows.Close()

	return result, nil
}

func applyProductionPlanActuals(row *DailyPlanItemRow, actuals map[string]int) {
	row.ActualQty = productionPlanActualQtyFromMap(row.PlanDate, row.LineID, row.ShiftNo, row.ModelID, row.ComponentID, actuals)
	row.RemainingQty = row.PlannedQty - row.ActualQty
	if row.PlannedQty > 0 {
		row.CompletionPct = row.ActualQty * 100 / row.PlannedQty
	}
}

func productionPlanActualQtyFromMap(planDate string, lineID, shiftNo, modelID, componentID int, actuals map[string]int) int {
	if actuals == nil {
		return 0
	}
	if shiftNo <= 0 {
		shiftNo = 1
	}
	return actuals[productionPlanActualKey(planDate, lineID, shiftNo, modelID, componentID)]
}
