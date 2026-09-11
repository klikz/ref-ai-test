//go:build windows

package utils

import (
	"os"
	"strings"
	"testing"
)

// TestMain lets this test binary double as a print worker child process, so the
// stdin/stdout protocol can be exercised against a real spawned process. The
// interception must happen before m.Run, which would reject the unknown flag.
func TestMain(m *testing.M) {
	if MaybeRunPrintWorker(os.Args) {
		return
	}
	os.Exit(m.Run())
}

const missingPrinterName = "AC-TEST-NONEXISTENT-PRINTER"

func startTestWorker(t *testing.T) *rawPrintWorkerClient {
	t.Helper()
	w := &rawPrintWorkerClient{language: PrintLanguageTSPL, index: 99}

	w.mu.Lock()
	err := w.ensureStartedLocked()
	w.mu.Unlock()
	if err != nil {
		t.Fatalf("worker ishga tushmadi: %v", err)
	}

	t.Cleanup(func() {
		w.mu.Lock()
		w.stopLocked()
		w.mu.Unlock()
	})
	return w
}

// A failed print must not poison the worker: the request id has to keep the
// reply stream aligned so the next job still works.
func TestRawPrintWorkerSurvivesPrintError(t *testing.T) {
	w := startTestWorker(t)

	for i, serial := range []string{"SN-A", "SN-B", "SN-C"} {
		jobID, err := w.print(missingPrinterName, serial, "RAW", []byte("SIZE 40 mm,30 mm\r\nPRINT 1\r\n"))
		if err == nil {
			t.Fatalf("#%d: mavjud bo'lmagan printer uchun xato kutilgan edi (job %d)", i, jobID)
		}
		// The error must come from the spooler call, not from a broken pipe or
		// a desynced reply.
		if !strings.Contains(err.Error(), "OpenPrinter") {
			t.Fatalf("#%d: OpenPrinter xatosi kutilgan, olindi: %v", i, err)
		}
	}
}

// Guards the "OK <reqID> <jobID>" / "ERR <reqID> <msg>" contract between the
// parent and the worker it spawns.
func TestRawPrintWorkerReplyCarriesRequestID(t *testing.T) {
	w := startTestWorker(t)

	before := printRequestSeq.Load()
	if _, err := w.print(missingPrinterName, "SN-1", "RAW", []byte("X")); err == nil {
		t.Fatal("xato kutilgan edi")
	}
	if printRequestSeq.Load() <= before {
		t.Error("har bir so'rov uchun request id oshishi kerak")
	}

	// A reply for a request we already gave up on must be skipped rather than
	// returned to the next caller.
	stale, valid := parseRawWorkerReply("OK 1 500")
	if !valid || stale.reqID != 1 {
		t.Fatalf("eskirgan javob parse qilinmadi: %+v", stale)
	}
}

func TestRawPrintWorkerRestartsAfterKill(t *testing.T) {
	w := startTestWorker(t)

	if _, err := w.print(missingPrinterName, "SN-1", "RAW", []byte("X")); err == nil {
		t.Fatal("xato kutilgan edi")
	}

	w.mu.Lock()
	w.stopLocked()
	w.mu.Unlock()

	// The next print must transparently start a fresh worker process.
	if _, err := w.print(missingPrinterName, "SN-2", "RAW", []byte("X")); err == nil {
		t.Fatal("xato kutilgan edi")
	} else if !strings.Contains(err.Error(), "OpenPrinter") {
		t.Fatalf("qayta ishga tushgan workerdan OpenPrinter xatosi kutilgan, olindi: %v", err)
	}
}
