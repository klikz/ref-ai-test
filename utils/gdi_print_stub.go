//go:build !windows

package utils

import "errors"

func printImageWindows(_, _, _, _ string, _, _ float64, _, _ int) error {
	return errors.New("GDI chop etish faqat Windowsda mavjud")
}
