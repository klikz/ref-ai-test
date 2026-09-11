//go:build !windows

package utils

func verifySpoolJob(_ string, _ uint32, _ string) error { return nil }

func cancelStuckPrinterJobs(_ string) (int, error) { return 0, nil }

func cancelPrintJobByID(_ string, _ int) error { return nil }

func snapshotSpoolJobIDs(_ string) map[int]bool { return nil }

func findNewSpoolJobID(_, _ string, _ map[int]bool) uint32 { return 0 }

func spoolVerifySkipped(_ string) bool { return true }
