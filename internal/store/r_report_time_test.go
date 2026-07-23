package store

import "testing"

func TestNormalizeReportTimeRangeDateOnly(t *testing.T) {
	start, end, err := NormalizeReportTimeRange("2026-07-20", "2026-07-20")
	if err != nil {
		t.Fatal(err)
	}
	if start != "2026-07-20 00:00:00" {
		t.Fatalf("start = %s", start)
	}
	if end != "2026-07-21 00:00:00" {
		t.Fatalf("end = %s", end)
	}
}

func TestNormalizeReportTimeRangeDateTime(t *testing.T) {
	start, end, err := NormalizeReportTimeRange("2026-07-20T08:00", "2026-07-20T20:00")
	if err != nil {
		t.Fatal(err)
	}
	if start != "2026-07-20 08:00:00" {
		t.Fatalf("start = %s", start)
	}
	if end != "2026-07-20 20:01:00" {
		t.Fatalf("end = %s", end)
	}
}
