package utils

import "testing"

func TestApplyDBPortKeyValueWithoutPort(t *testing.T) {
	got := applyDBPort(
		"host=localhost database='ac' user='postgres' password='postgres' sslmode='disable'",
		"5433",
	)
	want := "host=localhost database='ac' user='postgres' password='postgres' sslmode='disable' port=5433"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestApplyDBPortKeyValueReplaceExisting(t *testing.T) {
	got := applyDBPort("host=localhost port=5432 dbname=ac", "5433")
	want := "host=localhost port=5433 dbname=ac"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestApplyDBPortURL(t *testing.T) {
	got := applyDBPort("postgres://postgres:postgres@localhost:5432/ac?sslmode=disable", "5433")
	want := "postgres://postgres:postgres@localhost:5433/ac?sslmode=disable"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestApplyDBPortURLWithoutPort(t *testing.T) {
	got := applyDBPort("postgres://postgres:postgres@localhost/ac?sslmode=disable", "5433")
	want := "postgres://postgres:postgres@localhost:5433/ac?sslmode=disable"
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}
