//go:build !windows

package utils

import "errors"

func rawPrintWindows(_, _, _ string, _ []byte) error {
	return errors.New("RAW print faqat Windows serverda ishlaydi")
}
