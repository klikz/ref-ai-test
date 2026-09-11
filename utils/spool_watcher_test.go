package utils

import (
	"testing"
	"time"
)

func TestSpoolStatusClassification(t *testing.T) {
	stuck := []string{
		"Error", "Paused", "Offline", "PaperOut", "Paper Out",
		"UserIntervention", "Blocked", "Error,Printing", "OFFLINE",
	}
	for _, s := range stuck {
		if !isStuckSpoolStatus(s) {
			t.Errorf("isStuckSpoolStatus(%q) = false, kutilgan true", s)
		}
	}

	notStuck := []string{"Printing", "Spooling", "Printed", "Retained", "", "Unknown"}
	for _, s := range notStuck {
		if isStuckSpoolStatus(s) {
			t.Errorf("isStuckSpoolStatus(%q) = true, kutilgan false", s)
		}
	}

	active := []string{"Spooling", "Printing", "Restarting", "Spooling,Printing"}
	for _, s := range active {
		if !isActiveSpoolStatus(s) {
			t.Errorf("isActiveSpoolStatus(%q) = false, kutilgan true", s)
		}
	}
	for _, s := range []string{"Printed", "Retained", "Deleted", ""} {
		if isActiveSpoolStatus(s) {
			t.Errorf("isActiveSpoolStatus(%q) = true, kutilgan false", s)
		}
	}
}

func TestSpoolWatcherTracking(t *testing.T) {
	w := &spoolWatcher{jobs: make(map[string]*watchedSpoolJob)}
	prev := globalSpoolWatcher
	globalSpoolWatcher = w
	t.Cleanup(func() { globalSpoolWatcher = prev })

	if spoolWatcherTracks("Zebra ZT411", 42) {
		t.Fatal("bo'sh watcher job kuzatmasligi kerak")
	}

	job := &watchedSpoolJob{printer: "Zebra ZT411", jobID: 42}
	w.jobs[spoolWatchKey(job.printer, job.jobID)] = job

	if !spoolWatcherTracks("Zebra ZT411", 42) {
		t.Error("kuzatilayotgan job topilmadi")
	}
	// Windows printer names are case-insensitive.
	if !spoolWatcherTracks("zebra zt411", 42) {
		t.Error("printer nomi katta-kichik harfga sezgir bo'lmasligi kerak")
	}
	if spoolWatcherTracks("Zebra ZT411", 43) {
		t.Error("boshqa job id mos kelmasligi kerak")
	}
	// jobID 0 means "unknown", never a match.
	if spoolWatcherTracks("Zebra ZT411", 0) {
		t.Error("noma'lum job id kuzatilgan deb hisoblanmasligi kerak")
	}

	w.untrack(job)
	if spoolWatcherTracks("Zebra ZT411", 42) {
		t.Error("untrack ishlamadi")
	}
}

func newTestWatcher(job *watchedSpoolJob) *spoolWatcher {
	w := &spoolWatcher{jobs: make(map[string]*watchedSpoolJob)}
	w.jobs[spoolWatchKey(job.printer, job.jobID)] = job
	return w
}

// A job that is merely slow must survive: cancelling it is what produced false
// failures and duplicate labels.
func TestSpoolWatcherLeavesActiveJobQueued(t *testing.T) {
	job := &watchedSpoolJob{printer: "Gprinter GP-1", jobID: 7, submitted: time.Now()}
	w := newTestWatcher(job)

	w.evaluate(job, map[uint32]LocalPrintJob{
		7: {ID: 7, JobStatus: "Printing"},
	})

	if len(w.jobs) != 1 {
		t.Fatal("faol job kuzatuvda qolishi kerak edi")
	}
}

func TestSpoolWatcherForgetsDrainedJob(t *testing.T) {
	job := &watchedSpoolJob{printer: "Gprinter GP-1", jobID: 7, submitted: time.Now()}
	w := newTestWatcher(job)

	w.evaluate(job, map[uint32]LocalPrintJob{})

	if len(w.jobs) != 0 {
		t.Fatal("navbatdan chiqqan job kuzatuvdan olib tashlanishi kerak")
	}
}

func TestSpoolWatcherDropsStuckJob(t *testing.T) {
	job := &watchedSpoolJob{printer: "Gprinter GP-1", jobID: 7, submitted: time.Now()}
	w := newTestWatcher(job)

	w.evaluate(job, map[uint32]LocalPrintJob{
		7: {ID: 7, JobStatus: "Error,Offline"},
	})

	if len(w.jobs) != 0 {
		t.Fatal("qotib qolgan job kuzatuvdan olib tashlanishi kerak")
	}
}

// Beyond the watch window a still-moving job is released, not cancelled.
func TestSpoolWatcherReleasesLongRunningJob(t *testing.T) {
	job := &watchedSpoolJob{
		printer:   "Gprinter GP-1",
		jobID:     7,
		submitted: time.Now().Add(-2 * spoolWatchDeadline()),
	}
	w := newTestWatcher(job)

	w.evaluate(job, map[uint32]LocalPrintJob{
		7: {ID: 7, JobStatus: "Printing"},
	})

	if len(w.jobs) != 0 {
		t.Fatal("deadline'dan keyin job kuzatuvdan chiqishi kerak")
	}
}
