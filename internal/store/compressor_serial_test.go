package store

import "testing"

func TestModelCompressorSerialMatches(t *testing.T) {
	tests := []struct {
		scanned string
		stored  string
		want    bool
	}{
		{"GMCC123", "GMCC\nSEC", true},
		{"SEC999", "GMCC\nSEC", true},
		{"OTHER", "GMCC\nSEC", false},
		{"ANY123", "*", true},
		{"", "GMCC", false},
		{"X1", "GMCC\n*", true},
	}

	for _, tc := range tests {
		got := ModelCompressorSerialMatches(tc.scanned, tc.stored)
		if got != tc.want {
			t.Fatalf("ModelCompressorSerialMatches(%q, %q) = %v, want %v", tc.scanned, tc.stored, got, tc.want)
		}
	}
}

func TestValidateModelCompressorSerial(t *testing.T) {
	if err := validateModelCompressorSerial("*"); err != nil {
		t.Fatalf("expected * to be valid: %v", err)
	}
	if err := validateModelCompressorSerial(""); err == nil {
		t.Fatal("expected empty to fail")
	}
}
