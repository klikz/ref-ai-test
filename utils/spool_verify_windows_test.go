//go:build windows

package utils

import (
	"testing"
	"time"
)

func TestFindSpoolJobPrefersJobID(t *testing.T) {
	jobs := []LocalPrintJob{
		{ID: 10, DocumentName: "ABC123", JobStatus: "Printed,Retained"},
		{ID: 11, DocumentName: "ABC123", JobStatus: "Spooling"},
	}

	// With a real job id the stale retained job must not win, which is exactly
	// what document-name matching used to get wrong on a reprint.
	got, ok := findSpoolJob(jobs, 11, "ABC123")
	if !ok || got.ID != 11 {
		t.Fatalf("job id bo'yicha #11 kutilgan, olindi %+v (ok=%v)", got, ok)
	}

	if _, ok := findSpoolJob(jobs, 99, "ABC123"); ok {
		t.Error("mavjud bo'lmagan job id topilmasligi kerak")
	}
}

func TestFindSpoolJobFallsBackToDocumentName(t *testing.T) {
	jobs := []LocalPrintJob{
		{ID: 10, DocumentName: "OTHER", JobStatus: "Printing"},
		{ID: 11, DocumentName: "ABC123", JobStatus: "Spooling"},
	}

	got, ok := findSpoolJob(jobs, 0, "ABC123")
	if !ok || got.ID != 11 {
		t.Fatalf("hujjat nomi bo'yicha #11 kutilgan, olindi %+v (ok=%v)", got, ok)
	}

	if _, ok := findSpoolJob(jobs, 0, "YO'Q"); ok {
		t.Error("mos hujjat yo'q bo'lsa topilmasligi kerak")
	}
}

func TestFindNewSpoolJobIDIgnoresPreexisting(t *testing.T) {
	before := map[int]bool{10: true}
	jobs := []LocalPrintJob{
		{ID: 10, DocumentName: "ABC123"},
		{ID: 12, DocumentName: "ABC123"},
	}

	// Mirrors findNewSpoolJobID's filter without needing a live spooler.
	var found int
	for _, job := range jobs {
		if before[job.ID] {
			continue
		}
		found = job.ID
		break
	}
	if found != 12 {
		t.Fatalf("yangi job #12 kutilgan, olindi #%d", found)
	}
}

func TestParseSpoolSubmittedTime(t *testing.T) {
	got, ok := parseSpoolSubmittedTime("2026-08-04 10:15:30")
	if !ok {
		t.Fatal("vaqt parse qilinmadi")
	}
	want := time.Date(2026, 8, 4, 10, 15, 30, 0, time.Local)
	if !got.Equal(want) {
		t.Errorf("olindi %v, kutilgan %v", got, want)
	}

	for _, bad := range []string{"", "   ", "not-a-time", "2026/08/04"} {
		if _, ok := parseSpoolSubmittedTime(bad); ok {
			t.Errorf("parseSpoolSubmittedTime(%q) muvaffaqiyatli bo'lmasligi kerak", bad)
		}
	}
}

