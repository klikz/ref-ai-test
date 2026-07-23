package store

import (
	"errors"
	"strings"
)

func ParseModelAccSerialPrefixes(raw string) []string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil
	}

	seen := map[string]struct{}{}
	out := []string{}
	for _, chunk := range strings.FieldsFunc(raw, isAccSerialDelimiter) {
		item := strings.TrimSpace(chunk)
		if item == "" {
			continue
		}
		if _, ok := seen[item]; ok {
			continue
		}
		seen[item] = struct{}{}
		out = append(out, item)
	}
	return out
}

func isAccSerialDelimiter(r rune) bool {
	switch r {
	case '\n', '\r', ',', ';':
		return true
	default:
		return false
	}
}

func NormalizeModelAccSerialStorage(raw string) string {
	return strings.Join(ParseModelAccSerialPrefixes(raw), "\n")
}

func validateModelAccSerial(raw string) error {
	if len(ParseModelAccSerialPrefixes(raw)) == 0 {
		return errors.New("acc_serial bo'sh bo'lishi mumkin emas")
	}
	return nil
}

// ModelAccSerialMatches checks whether scanned acc serial starts with one of the
// model prefixes. Scanned value may be longer; only the beginning is compared.
// Wildcard "*" accepts any non-empty scanned value.
func ModelAccSerialMatches(scanned, stored string) bool {
	scanned = strings.TrimSpace(scanned)
	if scanned == "" {
		return false
	}
	if modelAccSerialWildcard(stored) {
		return true
	}
	for _, prefix := range ParseModelAccSerialPrefixes(stored) {
		if prefix != "" && strings.HasPrefix(scanned, prefix) {
			return true
		}
	}
	return false
}
