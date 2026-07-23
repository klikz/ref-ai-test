package store

import "testing"

func TestModelAccSerialMatches(t *testing.T) {
	stored := "ABC\nXYZ"

	tests := []struct {
		name    string
		scanned string
		want    bool
	}{
		{name: "exact prefix", scanned: "ABC", want: true},
		{name: "longer than prefix", scanned: "ABC123456789", want: true},
		{name: "second prefix exact", scanned: "XYZ", want: true},
		{name: "second prefix longer", scanned: "XYZ-999", want: true},
		{name: "too short", scanned: "AB", want: false},
		{name: "wrong start", scanned: "ABD123", want: false},
		{name: "empty scanned", scanned: "   ", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ModelAccSerialMatches(tt.scanned, stored); got != tt.want {
				t.Fatalf("ModelAccSerialMatches(%q, %q) = %v, want %v", tt.scanned, stored, got, tt.want)
			}
		})
	}
}

func TestParseModelAccSerialPrefixes(t *testing.T) {
	got := ParseModelAccSerialPrefixes(" ABC , XYZ;\n123 ")
	want := []string{"ABC", "XYZ", "123"}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v, want %v", got, want)
		}
	}
}
