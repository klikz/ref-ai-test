package store

import (
	"errors"
	"strings"
)

func NormalizeModelCompressorSerialStorage(raw string) string {
	return strings.Join(ParseModelAccSerialPrefixes(raw), "\n")
}

func validateModelCompressorSerial(raw string) error {
	if len(ParseModelAccSerialPrefixes(raw)) == 0 {
		return errors.New("compressor_serial bo'sh bo'lishi mumkin emas")
	}
	return nil
}

func modelCompressorSerialWildcard(stored string) bool {
	for _, prefix := range ParseModelAccSerialPrefixes(stored) {
		if prefix == "*" {
			return true
		}
	}
	return false
}

// ModelCompressorSerialMatches checks scanned compressor against model rules.
// Wildcard "*" in stored rules accepts any non-empty scanned value.
func ModelCompressorSerialMatches(scanned, stored string) bool {
	scanned = strings.TrimSpace(scanned)
	if scanned == "" {
		return false
	}
	if modelCompressorSerialWildcard(stored) {
		return true
	}
	for _, prefix := range ParseModelAccSerialPrefixes(stored) {
		if prefix != "" && strings.HasPrefix(scanned, prefix) {
			return true
		}
	}
	return false
}
