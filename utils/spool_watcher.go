package utils

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/klikz/api_v3/internal/models"
)

// errSpoolJobBlocked means the job was found blocked in the queue and has been
// cancelled, so nothing was printed and resubmitting is safe.
var errSpoolJobBlocked = errors.New("spool stuck")

// Spool statuses that mean the job will not print until an operator intervenes.
func isStuckSpoolStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	markers := []string{
		"error", "paused", "offline", "paperout", "paper out",
		"userintervention", "user intervention", "blocked",
	}
	for _, m := range markers {
		if strings.Contains(s, m) {
			return true
		}
	}
	return false
}

func isActiveSpoolStatus(status string) bool {
	s := strings.ToLower(strings.TrimSpace(status))
	return strings.Contains(s, "spooling") ||
		strings.Contains(s, "printing") ||
		strings.Contains(s, "restarting")
}

// watchedSpoolJob is a job that was accepted by the spooler but had not drained
// by the time the HTTP response went out.
type watchedSpoolJob struct {
	u           *UtilsStruct
	printer     string
	jobID       uint32
	serial      string
	printerV2ID int
	lineID      int
	lineName    string
	templateID  int
	language    string
	submitted   time.Time
}

// spoolWatcher follows accepted jobs after the operator has already been told
// the print succeeded.
//
// It reports and clears jobs that go bad later, but it never reprints: by the
// time a job reaches the watcher the printer may have produced the label
// already, so an automatic retry risks a duplicate. Late failures surface as
// stage=spool_late events on the metrics page for the operator to act on.
type spoolWatcher struct {
	mu        sync.Mutex
	jobs      map[string]*watchedSpoolJob
	startOnce sync.Once
}

var globalSpoolWatcher = &spoolWatcher{jobs: make(map[string]*watchedSpoolJob)}

func spoolWatchKey(printer string, jobID uint32) string {
	return strings.ToLower(strings.TrimSpace(printer)) + "#" + strconv.FormatUint(uint64(jobID), 10)
}

func spoolWatchDeadline() time.Duration {
	ms := envIntDefault("PRINT_V2_SPOOL_WATCH_MS", 60000)
	if ms < 5000 {
		ms = 5000
	}
	if ms > 600000 {
		ms = 600000
	}
	return time.Duration(ms) * time.Millisecond
}

func spoolWatchInterval() time.Duration {
	ms := envIntDefault("PRINT_V2_SPOOL_WATCH_INTERVAL_MS", 1000)
	if ms < 200 {
		ms = 200
	}
	if ms > 10000 {
		ms = 10000
	}
	return time.Duration(ms) * time.Millisecond
}

// spoolWatcherTracks reports whether a job belongs to an in-flight print, so
// queue cleanup does not cancel someone else's work.
func spoolWatcherTracks(printer string, jobID uint32) bool {
	if jobID == 0 {
		return false
	}
	globalSpoolWatcher.mu.Lock()
	defer globalSpoolWatcher.mu.Unlock()
	_, ok := globalSpoolWatcher.jobs[spoolWatchKey(printer, jobID)]
	return ok
}

func (w *spoolWatcher) track(job watchedSpoolJob) {
	if job.jobID == 0 || strings.TrimSpace(job.printer) == "" {
		return
	}
	if spoolVerifySkipped(job.printer) {
		return
	}
	job.submitted = time.Now()

	w.mu.Lock()
	w.jobs[spoolWatchKey(job.printer, job.jobID)] = &job
	w.mu.Unlock()

	w.startOnce.Do(func() { go w.loop() })
}

func (w *spoolWatcher) snapshot() map[string][]*watchedSpoolJob {
	w.mu.Lock()
	defer w.mu.Unlock()
	if len(w.jobs) == 0 {
		return nil
	}
	byPrinter := make(map[string][]*watchedSpoolJob, len(w.jobs))
	for _, job := range w.jobs {
		byPrinter[job.printer] = append(byPrinter[job.printer], job)
	}
	return byPrinter
}

func (w *spoolWatcher) untrack(job *watchedSpoolJob) {
	w.mu.Lock()
	delete(w.jobs, spoolWatchKey(job.printer, job.jobID))
	w.mu.Unlock()
}

func (w *spoolWatcher) loop() {
	ticker := time.NewTicker(spoolWatchInterval())
	defer ticker.Stop()
	for range ticker.C {
		w.tick()
	}
}

func (w *spoolWatcher) tick() {
	defer func() { _ = recover() }()

	for printer, tracked := range w.snapshot() {
		jobs, err := LocalPrinterJobs(printer)
		if err != nil {
			continue
		}
		queue := make(map[uint32]LocalPrintJob, len(jobs))
		for _, job := range jobs {
			queue[uint32(job.ID)] = job
		}
		for _, job := range tracked {
			w.evaluate(job, queue)
		}
	}
}

func (w *spoolWatcher) evaluate(job *watchedSpoolJob, queue map[uint32]LocalPrintJob) {
	current, stillQueued := queue[job.jobID]
	if !stillQueued {
		w.untrack(job)
		return
	}

	age := time.Since(job.submitted)
	if isStuckSpoolStatus(current.JobStatus) {
		w.untrack(job)
		_ = cancelPrintJobByID(job.printer, current.ID)
		job.reportLateFailure(fmt.Sprintf(
			"spool stuck: job #%d status=%s (yuborilgandan %s keyin)",
			current.ID, current.JobStatus, age.Round(time.Second),
		), age)
		return
	}

	if age >= spoolWatchDeadline() {
		// Still moving, just slow or a large batch. Left in the queue on
		// purpose; only stop watching it.
		w.untrack(job)
		if job.u != nil {
			job.u.PrintLogWarn("print spool: job hali navbatda", map[string]any{
				"printer":        job.printer,
				"printer_v2_id":  job.printerV2ID,
				"line_id":        job.lineID,
				"line_name":      job.lineName,
				"product_serial": job.serial,
				"spool_job_id":   current.ID,
				"job_status":     current.JobStatus,
				"age_ms":         age.Milliseconds(),
				"hint":           "sekin printer yoki katta partiya; job bekor qilinmadi",
			})
		}
	}
}

func (j *watchedSpoolJob) reportLateFailure(message string, age time.Duration) {
	if j.u == nil {
		return
	}
	detail := fmt.Sprintf("serial %s: %s", j.serial, message)
	j.u.PrintLogError("print spool: job yuborilgandan keyin qotib qoldi", map[string]any{
		"printer":        j.printer,
		"printer_v2_id":  j.printerV2ID,
		"line_id":        j.lineID,
		"line_name":      j.lineName,
		"product_serial": j.serial,
		"template_id":    j.templateID,
		"spool_job_id":   j.jobID,
		"age_ms":         age.Milliseconds(),
		"stage":          "spool_late",
		"message":        detail,
		"hint":           "etiketka chiqmagan bo'lishi mumkin — qo'lda qayta chop eting",
	})
	j.u.persistPrintV2Event(models.PrintV2Event{
		OK:                false,
		DurationMs:        int(age.Milliseconds()),
		LineID:            j.lineID,
		LineName:          j.lineName,
		PrinterV2ID:       j.printerV2ID,
		PrinterName:       j.printer,
		TemplateID:        j.templateID,
		EffectiveLanguage: j.language,
		Serial:            j.serial,
		Stage:             "spool_late",
		ErrorMessage:      truncatePrintErr(detail, 500),
		ErrorDetail:       detail,
	})
}
