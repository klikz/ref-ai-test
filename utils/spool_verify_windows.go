//go:build windows

package utils

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

func spoolVerifyEnabled() bool {
	v := strings.TrimSpace(strings.ToLower(os.Getenv("PRINT_V2_SPOOL_VERIFY")))
	if v == "0" || v == "false" || v == "off" || v == "no" {
		return false
	}
	return true
}

func spoolVerifyTimeout() time.Duration {
	ms := envIntDefault("PRINT_V2_SPOOL_VERIFY_MS", 400)
	if ms < 100 {
		ms = 100
	}
	if ms > 15000 {
		ms = 15000
	}
	return time.Duration(ms) * time.Millisecond
}

func spoolVerifySkipped(printerName string) bool {
	return !spoolVerifyEnabled() ||
		strings.TrimSpace(printerName) == "" ||
		strings.EqualFold(strings.TrimSpace(printerName), "Microsoft Print to PDF")
}

// verifySpoolJob inspects a just-submitted job by its spooler job id.
//
//   - gone from the queue          → printed, OK
//   - hard stuck (Error/Offline/…) → cancel that one job, return a retryable error
//   - still Spooling/Printing      → OK
//
// An active job is deliberately left alone. Cancelling it (what this used to do
// after a fixed timeout) produced false "navbatda qolib ketdi" errors on slower
// printers and could reprint a label that had already come out. Long-running
// jobs are handed to the spool watcher instead, which reports but never
// reprints.
//
// jobID 0 means the id is unknown (older worker, or a GDI job that finished
// before it could be observed); in that case only a hard-stuck job matching
// documentName is detected.
func verifySpoolJob(printerName string, jobID uint32, documentName string) error {
	printerName = strings.TrimSpace(printerName)
	documentName = strings.TrimSpace(documentName)
	if spoolVerifySkipped(printerName) {
		return nil
	}
	if jobID == 0 && documentName == "" {
		return nil
	}

	deadline := time.Now().Add(spoolVerifyTimeout())
	for {
		jobs, err := LocalPrinterJobs(printerName)
		if err != nil {
			// Queue query failure must not fail an otherwise successful print.
			return nil
		}

		job, found := findSpoolJob(jobs, jobID, documentName)
		if !found {
			return nil
		}
		if isStuckSpoolStatus(job.JobStatus) {
			_ = cancelPrintJobByID(printerName, job.ID)
			return fmt.Errorf(
				"%w: job #%d status=%s (printer pauza/offline/xato yoki media)",
				errSpoolJobBlocked, job.ID, job.JobStatus,
			)
		}
		if !time.Now().Before(deadline) {
			// Accepted and progressing — the watcher takes it from here.
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
}

func findSpoolJob(jobs []LocalPrintJob, jobID uint32, documentName string) (LocalPrintJob, bool) {
	if jobID != 0 {
		for _, job := range jobs {
			if uint32(job.ID) == jobID {
				return job, true
			}
		}
		return LocalPrintJob{}, false
	}
	matches := filterPrintJobsByDocument(jobs, documentName)
	if len(matches) == 0 {
		return LocalPrintJob{}, false
	}
	return matches[0], true
}

// snapshotSpoolJobIDs records the queue before a GDI submit. PrintDocument does
// not report a job id, so the new job is identified by difference afterwards.
func snapshotSpoolJobIDs(printerName string) map[int]bool {
	if spoolVerifySkipped(printerName) {
		return nil
	}
	jobs, err := LocalPrinterJobs(printerName)
	if err != nil {
		return nil
	}
	seen := make(map[int]bool, len(jobs))
	for _, job := range jobs {
		seen[job.ID] = true
	}
	return seen
}

// findNewSpoolJobID returns the id of the job created since the snapshot. It
// returns 0 when the job already drained, which callers treat as unknown.
func findNewSpoolJobID(printerName, documentName string, before map[int]bool) uint32 {
	if spoolVerifySkipped(printerName) {
		return 0
	}
	jobs, err := LocalPrinterJobs(printerName)
	if err != nil {
		return 0
	}
	documentName = strings.TrimSpace(documentName)
	for _, job := range jobs {
		if before[job.ID] {
			continue
		}
		if documentName != "" && !strings.EqualFold(strings.TrimSpace(job.DocumentName), documentName) {
			continue
		}
		return uint32(job.ID)
	}
	return 0
}

func filterPrintJobsByDocument(jobs []LocalPrintJob, documentName string) []LocalPrintJob {
	want := strings.TrimSpace(documentName)
	out := make([]LocalPrintJob, 0, len(jobs))
	for _, job := range jobs {
		if strings.EqualFold(strings.TrimSpace(job.DocumentName), want) {
			out = append(out, job)
		}
	}
	return out
}

func cancelPrintJobsByDocument(printerName, documentName string) error {
	printerName = strings.TrimSpace(printerName)
	documentName = strings.TrimSpace(documentName)
	if printerName == "" || documentName == "" {
		return nil
	}
	jobs, err := LocalPrinterJobs(printerName)
	if err != nil {
		return cancelPrintJobsByDocumentPowerShell(printerName, documentName)
	}
	var lastErr error
	cancelled := 0
	for _, job := range filterPrintJobsByDocument(jobs, documentName) {
		if err := cancelPrintJobByID(printerName, job.ID); err != nil {
			lastErr = err
			continue
		}
		cancelled++
	}
	if cancelled == 0 && lastErr != nil {
		return lastErr
	}
	return nil
}

func cancelPrintJobsByDocumentPowerShell(printerName, documentName string) error {
	script := `
$printerName = $env:AC_PRINTER_NAME
$docName = $env:AC_PRINT_DOC
$jobs = @(Get-PrintJob -PrinterName $printerName -ErrorAction SilentlyContinue)
foreach ($j in $jobs) {
	if ($null -ne $j -and [string]$j.DocumentName -eq $docName) {
		try { Remove-PrintJob -InputObject $j -Confirm:$false -ErrorAction Stop } catch {}
	}
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(),
		"AC_PRINTER_NAME="+printerName,
		"AC_PRINT_DOC="+documentName,
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("print job bekor qilinmadi: %s", msg)
	}
	return nil
}

// stuckJobClearAge keeps the pre-print queue cleanup from touching jobs that a
// concurrent print just submitted.
const stuckJobClearAge = 20 * time.Second

// cancelStuckPrinterJobs clears Error/Paused/Offline jobs that block the queue
// before a print. Only jobs that are demonstrably stale are removed: anything
// the spool watcher is tracking, or that was submitted moments ago, belongs to
// another in-flight print and must be left alone.
func cancelStuckPrinterJobs(printerName string) (int, error) {
	printerName = strings.TrimSpace(printerName)
	if printerName == "" {
		return 0, nil
	}
	jobs, err := LocalPrinterJobs(printerName)
	if err != nil {
		return 0, err
	}
	cutoff := time.Now().Add(-stuckJobClearAge)
	cancelled := 0
	for _, job := range jobs {
		if !isStuckSpoolStatus(job.JobStatus) {
			continue
		}
		if spoolWatcherTracks(printerName, uint32(job.ID)) {
			continue
		}
		if submitted, ok := parseSpoolSubmittedTime(job.SubmittedTime); ok && submitted.After(cutoff) {
			continue
		}
		if err := cancelPrintJobByID(printerName, job.ID); err == nil {
			cancelled++
		}
	}
	return cancelled, nil
}

func parseSpoolSubmittedTime(value string) (time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, false
	}
	t, err := time.ParseInLocation("2006-01-02 15:04:05", value, time.Local)
	if err != nil {
		return time.Time{}, false
	}
	return t, true
}

func cancelPrintJobByID(printerName string, jobID int) error {
	if jobID <= 0 {
		return nil
	}
	if err := cancelPrintJobByIDWin32(printerName, jobID); err == nil {
		return nil
	}
	return cancelPrintJobByIDPowerShell(printerName, jobID)
}

func cancelPrintJobByIDPowerShell(printerName string, jobID int) error {
	script := `
$printerName = $env:AC_PRINTER_NAME
$jobId = [int]$env:AC_PRINT_JOB_ID
$j = Get-PrintJob -PrinterName $printerName -ID $jobId -ErrorAction SilentlyContinue
if ($null -ne $j) {
	Remove-PrintJob -InputObject $j -Confirm:$false -ErrorAction Stop
}
`
	cmd := exec.Command("powershell", "-NoProfile", "-NonInteractive", "-Command", script)
	cmd.Env = append(os.Environ(),
		"AC_PRINTER_NAME="+printerName,
		"AC_PRINT_JOB_ID="+strconv.Itoa(jobID),
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		msg := strings.TrimSpace(string(out))
		if msg == "" {
			msg = err.Error()
		}
		return fmt.Errorf("job #%d bekor qilinmadi: %s", jobID, msg)
	}
	return nil
}
