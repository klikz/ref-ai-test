package store

import (
	"testing"
	"time"
)

func TestProductionShiftWindowDaytime(t *testing.T) {
	settings := ProductionShiftSettings{
		Shift1Start: "08:00",
		Shift1End:   "20:00",
		Shift2Start: "20:00",
		Shift2End:   "08:00",
	}
	start, end, err := settings.ShiftWindow("2026-07-20", 1)
	if err != nil {
		t.Fatal(err)
	}
	if start.Format("2006-01-02 15:04") != "2026-07-20 08:00" {
		t.Fatalf("shift1 start = %s", start)
	}
	if end.Format("2006-01-02 15:04") != "2026-07-20 20:00" {
		t.Fatalf("shift1 end = %s", end)
	}
}

func TestProductionShiftWindowOvernight(t *testing.T) {
	settings := ProductionShiftSettings{
		Shift1Start: "08:00",
		Shift1End:   "20:00",
		Shift2Start: "20:00",
		Shift2End:   "08:00",
	}
	start, end, err := settings.ShiftWindow("2026-07-20", 2)
	if err != nil {
		t.Fatal(err)
	}
	if start.Format("2006-01-02 15:04") != "2026-07-20 20:00" {
		t.Fatalf("shift2 start = %s", start)
	}
	if end.Format("2006-01-02 15:04") != "2026-07-21 08:00" {
		t.Fatalf("shift2 end = %s", end)
	}
}

func TestProductionShiftAssign(t *testing.T) {
	settings := ProductionShiftSettings{
		Shift1Start: "08:00",
		Shift1End:   "20:00",
		Shift2Start: "20:00",
		Shift2End:   "08:00",
	}
	loc := time.Local

	cases := []struct {
		at       time.Time
		shiftNo  int
		planDate string
	}{
		{time.Date(2026, 7, 20, 10, 0, 0, 0, loc), 1, "2026-07-20"},
		{time.Date(2026, 7, 20, 21, 0, 0, 0, loc), 2, "2026-07-20"},
		{time.Date(2026, 7, 21, 3, 0, 0, 0, loc), 2, "2026-07-20"},
		{time.Date(2026, 7, 21, 8, 0, 0, 0, loc), 1, "2026-07-21"},
	}
	for _, tc := range cases {
		shiftNo, planDate := settings.Assign(tc.at)
		if shiftNo != tc.shiftNo || planDate != tc.planDate {
			t.Fatalf("at=%s got shift=%d date=%s want shift=%d date=%s",
				tc.at, shiftNo, planDate, tc.shiftNo, tc.planDate)
		}
	}
}

func TestNormalizeShiftTime(t *testing.T) {
	got, err := normalizeShiftTime("8:05")
	if err != nil {
		t.Fatal(err)
	}
	if got != "08:05" {
		t.Fatalf("got %s", got)
	}
}
