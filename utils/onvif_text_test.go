package utils

import (
	"errors"
	"testing"
)

func TestIsONVIFNotAuthorized(t *testing.T) {
	if !isONVIFNotAuthorized(errors.New(`ter:NotAuthorized`)) {
		t.Fatal("expected true")
	}
	if isONVIFNotAuthorized(errors.New("timeout")) {
		t.Fatal("expected false")
	}
}

func TestParseSOAPFault(t *testing.T) {
	body := []byte(`<s:Fault><s:Reason><s:Text>Sender not Authorized. Invalid username</s:Text></s:Reason></s:Fault>`)
	if got := parseSOAPFault(body); got == "" {
		t.Fatal("expected fault text")
	}
}

func TestBuildONVIFPasswordTextSOAP(t *testing.T) {
	got := buildONVIFPasswordTextSOAP("admin", "pass&1", `<tds:GetCapabilities/>`)
	if !containsAll(got, "PasswordText", "admin", "pass&amp;1", "GetCapabilities") {
		t.Fatalf("unexpected soap: %s", got)
	}
}

func containsAll(s string, parts ...string) bool {
	for _, p := range parts {
		if !contains(s, p) {
			return false
		}
	}
	return true
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(sub) == 0 || indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
