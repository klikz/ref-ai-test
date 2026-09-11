//go:build windows

package utils

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type rawPrintWorkerClient struct {
	language string
	index    int

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	// generation guards against a supervisor for an old process tearing down a
	// freshly started one.
	generation uint64
}

// rawPrintWorkerPool keeps several worker processes per language so jobs for
// different printers do not serialize behind one another.
type rawPrintWorkerPool struct {
	language string
	once     sync.Once
	free     chan *rawPrintWorkerClient
	all      []*rawPrintWorkerClient
}

var (
	tsplRawPool = &rawPrintWorkerPool{language: PrintLanguageTSPL}
	zplRawPool  = &rawPrintWorkerPool{language: PrintLanguageZPL}
)

func (p *rawPrintWorkerPool) ensure() {
	p.once.Do(func() {
		n := printWorkerCount(p.language)
		p.free = make(chan *rawPrintWorkerClient, n)
		p.all = make([]*rawPrintWorkerClient, 0, n)
		for i := 0; i < n; i++ {
			w := &rawPrintWorkerClient{language: p.language, index: i}
			p.all = append(p.all, w)
			p.free <- w
		}
	})
}

func (p *rawPrintWorkerPool) warmUp() {
	p.ensure()
	var wg sync.WaitGroup
	for _, w := range p.all {
		wg.Add(1)
		go func(w *rawPrintWorkerClient) {
			defer wg.Done()
			w.mu.Lock()
			defer w.mu.Unlock()
			_ = w.ensureStartedLocked()
		}(w)
	}
	wg.Wait()
}

func (p *rawPrintWorkerPool) acquire() (*rawPrintWorkerClient, error) {
	p.ensure()
	select {
	case w := <-p.free:
		return w, nil
	default:
	}

	wait := submitQueueTimeout()
	timer := time.NewTimer(wait)
	defer timer.Stop()
	select {
	case w := <-p.free:
		return w, nil
	case <-timer.C:
		return nil, fmt.Errorf("%s print navbati band (%s kutildi)", p.language, wait)
	}
}

func (p *rawPrintWorkerPool) release(w *rawPrintWorkerClient) {
	p.free <- w
}

func rawPoolForLanguage(language string) *rawPrintWorkerPool {
	switch NormalizePrintLanguage(language) {
	case PrintLanguageTSPL:
		return tsplRawPool
	case PrintLanguageZPL:
		return zplRawPool
	default:
		return nil
	}
}

// rawPrintViaWorker sends a RAW payload through a TSPL or ZPL child process and
// returns the spooler job id it was given.
func rawPrintViaWorker(language, printerName, docName, datatype string, payload []byte) (uint32, error) {
	pool := rawPoolForLanguage(language)
	if pool == nil {
		return 0, fmt.Errorf("RAW worker faqat tspl/zpl uchun: %s", language)
	}
	w, err := pool.acquire()
	if err != nil {
		return 0, err
	}
	defer pool.release(w)

	jobID, err := w.print(printerName, docName, datatype, payload)
	if err == nil {
		return jobID, nil
	}
	// A dead worker fails before touching the spooler, so one respawn+resend is
	// safe. Anything else is surfaced so the caller can decide about retrying.
	if !errors.Is(err, errRawWorkerUnavailable) {
		return 0, err
	}
	return w.print(printerName, docName, datatype, payload)
}

func (w *rawPrintWorkerClient) name() string {
	return fmt.Sprintf("%s#%d", w.language, w.index)
}

func (w *rawPrintWorkerClient) stopLocked() {
	// Bumping the generation first stops the supervisor of the process we are
	// about to kill from clobbering a replacement.
	w.generation++
	if w.stdin != nil {
		_ = writeRawPrintWorkerFrame(w.stdin, rawPrintWorkerHeader{Cmd: "exit"}, nil)
		_ = w.stdin.Close()
		w.stdin = nil
	}
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
	}
	w.cmd = nil
	w.reader = nil
}

// markExited is called by the supervisor when the worker process dies on its
// own, so the next print starts a fresh one instead of writing to a dead pipe.
func (w *rawPrintWorkerClient) markExited(generation uint64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.generation != generation {
		return
	}
	if w.stdin != nil {
		_ = w.stdin.Close()
		w.stdin = nil
	}
	w.cmd = nil
	w.reader = nil
}

func (w *rawPrintWorkerClient) ensureStartedLocked() error {
	if w.cmd != nil && w.stdin != nil && w.reader != nil {
		return nil
	}

	w.stopLocked()

	exe, err := os.Executable()
	if err != nil {
		return fmt.Errorf("%s worker: executable: %w", w.name(), err)
	}

	cmd := exec.Command(exe, printWorkerArgPrefix+w.language)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}
	// Keep stderr visible in parent logs for worker crashes.
	cmd.Stderr = os.Stderr

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("%s worker ishga tushmadi: %w", w.name(), err)
	}
	assignCmdToPrintWorkersJob(cmd)

	reader := bufio.NewReader(stdout)
	readyCh := make(chan error, 1)
	go func() {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			readyCh <- readErr
			return
		}
		if strings.TrimSpace(line) != "READY" {
			readyCh <- fmt.Errorf("%s worker javobi kutilmagan: %q", w.name(), strings.TrimSpace(line))
			return
		}
		readyCh <- nil
	}()

	select {
	case err := <-readyCh:
		if err != nil {
			_ = stdin.Close()
			_ = cmd.Process.Kill()
			return fmt.Errorf("%s worker tayyor emas: %w", w.name(), err)
		}
	case <-time.After(45 * time.Second):
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		return fmt.Errorf("%s worker ishga tushish vaqti tugadi", w.name())
	}

	w.cmd = cmd
	w.stdin = stdin
	w.reader = reader

	generation := w.generation
	// os.Process.Wait reaps the child without closing the pipes that the print
	// path reads from, unlike exec.Cmd.Wait.
	go func(proc *os.Process) {
		_, _ = proc.Wait()
		w.markExited(generation)
	}(cmd.Process)

	return nil
}

func (w *rawPrintWorkerClient) print(printerName, docName, datatype string, payload []byte) (uint32, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.ensureStartedLocked(); err != nil {
		return 0, fmt.Errorf("%w: %v", errRawWorkerUnavailable, err)
	}

	reqID := nextPrintRequestID()
	header := rawPrintWorkerHeader{
		ReqID:    reqID,
		Printer:  printerName,
		Doc:      docName,
		Datatype: datatype,
	}
	if err := writeRawPrintWorkerFrame(w.stdin, header, payload); err != nil {
		w.stopLocked()
		return 0, fmt.Errorf("%w: %s workerga yozib bo'lmadi: %v", errRawWorkerUnavailable, w.name(), err)
	}

	reply, err := w.readReplyLocked(reqID)
	if err != nil {
		// The frame was already handed over, so the job may well have been
		// spooled. Never treat this as safe to resend.
		w.stopLocked()
		return 0, err
	}
	if !reply.ok {
		return 0, errors.New(reply.msg)
	}
	return reply.jobID, nil
}

// maxStaleWorkerReplies bounds how many leftover lines are skipped before the
// stream is declared unusable.
const maxStaleWorkerReplies = 8

// readReplyLocked waits for the reply to reqID, discarding replies to earlier
// requests that were abandoned on timeout. Without this, one late reply would
// shift every subsequent response by one and force a worker restart.
func (w *rawPrintWorkerClient) readReplyLocked(reqID uint64) (rawWorkerReply, error) {
	deadline := time.Now().Add(printWorkerReplyTimeout())

	for i := 0; i < maxStaleWorkerReplies; i++ {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			return rawWorkerReply{}, fmt.Errorf("%s worker javob vaqti tugadi", w.name())
		}
		line, err := readLineWithTimeout(w.reader, remaining)
		if err != nil {
			return rawWorkerReply{}, fmt.Errorf("%s worker javobi o'qilmadi: %w", w.name(), err)
		}

		reply, valid := parseRawWorkerReply(line)
		if !valid {
			return rawWorkerReply{}, fmt.Errorf("%s worker noma'lum javob: %s", w.name(), strings.TrimSpace(line))
		}
		if reply.reqID != reqID {
			continue
		}
		return reply, nil
	}
	return rawWorkerReply{}, fmt.Errorf("%s worker javoblari sinxrondan chiqdi", w.name())
}
