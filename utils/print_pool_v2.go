package utils

import (
	"errors"
	"fmt"
	"os"
	"runtime"
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

// PrintPoolV2 bounds concurrent label rendering.
//
// Rendering is CPU-bound and independent of the printer, so it is limited
// separately from spool submission, which is bounded by the per-language worker
// process pools. They used to share one semaphore per language, which meant a
// slow render occupied a printer slot and a busy printer throttled rendering
// for every other line.
type PrintPoolV2 struct {
	slots chan struct{}
}

var (
	globalRenderPoolV2 *PrintPoolV2
	printPoolV2Once    sync.Once
)

// InitPrintPoolV2 sizes the render pool. maxWorkers <= 0 falls back to the CPU
// count; PRINT_V2_RENDER_WORKERS overrides it.
func InitPrintPoolV2(maxWorkers int) {
	printPoolV2Once.Do(func() {
		if maxWorkers <= 0 {
			maxWorkers = runtime.NumCPU()
		}
		n := envIntDefault("PRINT_V2_RENDER_WORKERS", maxWorkers)
		if n < 1 {
			n = 1
		}
		globalRenderPoolV2 = &PrintPoolV2{slots: make(chan struct{}, n)}
	})
}

func renderPoolV2() *PrintPoolV2 {
	if globalRenderPoolV2 == nil {
		InitPrintPoolV2(runtime.NumCPU())
	}
	return globalRenderPoolV2
}

func envIntDefault(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

func (p *PrintPoolV2) acquire() {
	p.slots <- struct{}{}
}

func (p *PrintPoolV2) release() {
	<-p.slots
}

// WithRenderSlotV2 runs fn under the render semaphore.
func WithRenderSlotV2(fn func() error) error {
	pool := renderPoolV2()
	pool.acquire()
	defer pool.release()
	return fn()
}

// printRetryAttempts is the number of submit attempts per print request.
const printRetryAttempts = 2

// isRetryablePrintError reports whether a failure is known to have happened
// before any byte reached the printer.
//
// Everything else — worker reply timeouts, unknown worker responses, spooler
// errors of unclear outcome — is treated as "the label may already be out" and
// is not retried, because a duplicate label is worse than a reported failure.
func isRetryablePrintError(err error) bool {
	return errors.Is(err, errSpoolJobBlocked) || errors.Is(err, errRawWorkerUnavailable)
}

// RetryPrintV2 retries transient spooler failures. attempt is 1-based.
func RetryPrintV2(attempts int, fn func(attempt int) error) error {
	if attempts < 1 {
		attempts = 1
	}
	var lastErr error
	for i := 0; i < attempts; i++ {
		lastErr = fn(i + 1)
		if lastErr == nil {
			return nil
		}
		if !isRetryablePrintError(lastErr) {
			return lastErr
		}
		if i+1 < attempts {
			time.Sleep(time.Duration(150*(i+1)) * time.Millisecond)
		}
	}
	return fmt.Errorf("print retry tugadi: %w", lastErr)
}

// --- lightweight metrics (Should) ---

type PrintV2MetricsSnapshot struct {
	Success  int64 `json:"success"`
	Fail     int64 `json:"fail"`
	TotalMs  int64 `json:"total_ms"`
	LastMs   int64 `json:"last_ms"`
	InFlight int64 `json:"in_flight"`
}

var (
	printV2Success  atomic.Int64
	printV2Fail     atomic.Int64
	printV2TotalMs  atomic.Int64
	printV2LastMs   atomic.Int64
	printV2InFlight atomic.Int64
)

func recordPrintV2Success(ms int64) {
	printV2Success.Add(1)
	printV2TotalMs.Add(ms)
	printV2LastMs.Store(ms)
}

func recordPrintV2Fail(ms int64) {
	printV2Fail.Add(1)
	printV2TotalMs.Add(ms)
	printV2LastMs.Store(ms)
}

func PrintV2Metrics() PrintV2MetricsSnapshot {
	return PrintV2MetricsSnapshot{
		Success:  printV2Success.Load(),
		Fail:     printV2Fail.Load(),
		TotalMs:  printV2TotalMs.Load(),
		LastMs:   printV2LastMs.Load(),
		InFlight: printV2InFlight.Load(),
	}
}
