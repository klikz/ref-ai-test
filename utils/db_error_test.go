package utils

import (
	"strings"
	"testing"

	"github.com/lib/pq"
)

func TestFormatDBErrorPostgres(t *testing.T) {
	err := &pq.Error{
		Code:       "23505",
		Message:    "duplicate key value violates unique constraint",
		Detail:     "Key (serial)=(ABC) already exists.",
		Constraint: "products_unique",
		Table:      "products",
	}
	got := FormatDBError(err)
	if got == "" {
		t.Fatal("expected formatted message")
	}
	for _, part := range []string{"23505", "duplicate key", "products_unique", "products"} {
		if !strings.Contains(got, part) {
			t.Fatalf("expected %q in %q", part, got)
		}
	}
}
