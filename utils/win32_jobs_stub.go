//go:build !windows

package utils

import "errors"

var errNotWindowsPrint = errors.New("print queue faqat Windows serverda")

func localPrinterJobsWin32(_ string) ([]LocalPrintJob, error) {
	return nil, errNotWindowsPrint
}

func cancelPrintJobByIDWin32(_ string, _ int) error {
	return errNotWindowsPrint
}
