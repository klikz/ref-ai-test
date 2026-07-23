package store

import (
	"testing"
	"time"
)

// lib/pq returns `timestamp without time zone` as a UTC-labelled time.Time that
// preserves the stored wall clock. productionWallClockLocal must re-label it in
// time.Local so shift-window assignment matches the stored wall clock instead of
// being skewed by the local UTC offset.
func TestProductionWallClockLocalShiftAssignment(t *testing.T) {
	settings := ProductionShiftSettings{
		Shift1Start: "08:00",
		Shift1End:   "19:00",
		Shift2Start: "19:00",
		Shift2End:   "08:00",
	}

	cases := []struct {
		name      string
		wallClock time.Time
		wantShift int
		wantDate  string
	}{
		{"early morning belongs to previous shift2", time.Date(2026, 7, 21, 5, 0, 0, 0, time.UTC), 2, "2026-07-20"},
		{"daytime belongs to shift1", time.Date(2026, 7, 21, 9, 0, 0, 0, time.UTC), 1, "2026-07-21"},
		{"evening belongs to shift2", time.Date(2026, 7, 21, 21, 0, 0, 0, time.UTC), 2, "2026-07-21"},
	}
	for _, tc := range cases {
		local := productionWallClockLocal(tc.wallClock)
		shiftNo, planDate := settings.Assign(local)
		if shiftNo != tc.wantShift || planDate != tc.wantDate {
			t.Fatalf("%s: got shift %d date %s, want shift %d date %s", tc.name, shiftNo, planDate, tc.wantShift, tc.wantDate)
		}
	}
}

func TestProductionPlanActualKey(t *testing.T) {
	if got := productionPlanActualKey("2026-06-01", 4, 1, 123, 0); got != "2026-06-01|4|s:1|m:123" {
		t.Fatalf("model key = %q", got)
	}
	if got := productionPlanActualKey("2026-06-01", 8, 2, 0, 45); got != "2026-06-01|8|s:2|c:45" {
		t.Fatalf("component key = %q", got)
	}
}

func TestProductionPlanActualQtyFromMap(t *testing.T) {
	actuals := map[string]int{
		"2026-06-01|4|s:1|m:10": 7,
		"2026-06-01|9|s:2|c:3":  2,
	}
	if got := productionPlanActualQtyFromMap("2026-06-01", 4, 1, 10, 0, actuals); got != 7 {
		t.Fatalf("model actual = %d", got)
	}
	if got := productionPlanActualQtyFromMap("2026-06-01", 9, 2, 0, 3, actuals); got != 2 {
		t.Fatalf("component actual = %d", got)
	}
	if got := productionPlanActualQtyFromMap("2026-06-01", 4, 1, 99, 0, actuals); got != 0 {
		t.Fatalf("missing actual = %d", got)
	}
}

func TestApplyProductionPlanActuals(t *testing.T) {
	row := DailyPlanItemRow{PlanDate: "2026-06-01", LineID: 4, ShiftNo: 1, ModelID: 10, PlannedQty: 20}
	actuals := map[string]int{"2026-06-01|4|s:1|m:10": 15}
	applyProductionPlanActuals(&row, actuals)
	if row.ActualQty != 15 || row.RemainingQty != 5 || row.CompletionPct != 75 {
		t.Fatalf("unexpected row: %+v", row)
	}
}
