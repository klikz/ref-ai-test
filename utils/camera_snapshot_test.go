package utils

import (
	"fmt"
	"testing"
	"time"
)

func TestSanitizeT3ScanLabel(t *testing.T) {
	if got := SanitizeT3ScanLabel(""); got != "test" {
		t.Fatalf("empty: got %q", got)
	}
	if got := SanitizeT3ScanLabel("ABC-123_x"); got != "ABC-123_x" {
		t.Fatalf("alnum: got %q", got)
	}
	if got := SanitizeT3ScanLabel("bad label!"); got != "badlabel" {
		t.Fatalf("strip spaces/symbols: got %q", got)
	}
}

func TestSaveT3ScanPhotoPathFormat(t *testing.T) {
	capturedAt := time.Date(2026, 7, 3, 15, 4, 5, 0, time.Local)
	safeLabel := SanitizeT3ScanLabel("ABC-123")
	got := fmt.Sprintf(
		"/uploads/t3-scans/%s/%s/%s/%s.jpg",
		capturedAt.Format("2006"),
		capturedAt.Format("01"),
		capturedAt.Format("02"),
		safeLabel,
	)
	want := "/uploads/t3-scans/2026/07/03/ABC-123.jpg"
	if got != want {
		t.Fatalf("path format: got %q want %q", got, want)
	}
}
