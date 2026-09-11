//go:build !windows

package utils

import "errors"

func rawPrintWindows(_, _, _ string, _ []byte) (uint32, error) {
	return 0, errors.New("RAW print faqat Windows serverda ishlaydi")
}

func PrinterReachable(_ string) error {
	return errors.New("printer health check faqat Windowsda mavjud")
}
