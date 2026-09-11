//go:build windows

package utils

import (
	"bufio"
	_ "embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

//go:embed gdi_print_worker.ps1
var gdiPrintWorkerPSScript []byte

type gdiPrintJob struct {
	ReqID          uint64  `json:"ReqId"`
	ImagePath      string  `json:"ImagePath"`
	PrinterName    string  `json:"PrinterName"`
	OutputPath     string  `json:"OutputPath,omitempty"`
	WidthMm        float64 `json:"WidthMm"`
	HeightMm       float64 `json:"HeightMm"`
	Copies         int     `json:"Copies"`
	DocumentName   string  `json:"DocumentName"`
	RotationDeg    int     `json:"RotationDeg"`
	ForcePaperSize bool    `json:"ForcePaperSize"` // PDF/generic only — avoid resetting thermal DEVMODE
}

type gdiPrintWorker struct {
	index int

	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
	// generation guards against a supervisor for an old process tearing down a
	// freshly started one.
	generation uint64
}

// gdiPrintWorkerPool keeps several PowerShell workers so a slow printer does
// not block GDI jobs headed for a different one.
type gdiPrintWorkerPool struct {
	once sync.Once
	free chan *gdiPrintWorker
	all  []*gdiPrintWorker
}

var (
	gdiWorkerScriptOnce sync.Once
	gdiWorkerScriptPath string
	gdiWorkerScriptErr  error
	gdiPool             = &gdiPrintWorkerPool{}
)

func (p *gdiPrintWorkerPool) ensure() {
	p.once.Do(func() {
		n := printWorkerCount(PrintLanguageGDI)
		p.free = make(chan *gdiPrintWorker, n)
		p.all = make([]*gdiPrintWorker, 0, n)
		for i := 0; i < n; i++ {
			w := &gdiPrintWorker{index: i}
			p.all = append(p.all, w)
			p.free <- w
		}
	})
}

func (p *gdiPrintWorkerPool) warmUp() {
	p.ensure()
	var wg sync.WaitGroup
	for _, w := range p.all {
		wg.Add(1)
		go func(w *gdiPrintWorker) {
			defer wg.Done()
			w.mu.Lock()
			defer w.mu.Unlock()
			_ = w.ensureStartedLocked()
		}(w)
	}
	wg.Wait()
}

func (p *gdiPrintWorkerPool) acquire() (*gdiPrintWorker, error) {
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
		return nil, fmt.Errorf("GDI print navbati band (%s kutildi)", wait)
	}
}

func (p *gdiPrintWorkerPool) release(w *gdiPrintWorker) {
	p.free <- w
}

func (w *gdiPrintWorker) name() string {
	return fmt.Sprintf("GDI#%d", w.index)
}

func getGDIWorkerScriptPath() (string, error) {
	gdiWorkerScriptOnce.Do(func() {
		path := filepath.Join(os.TempDir(), "ac-label-v2-gdi-worker-v6.ps1")
		gdiWorkerScriptErr = os.WriteFile(path, gdiPrintWorkerPSScript, 0644)
		gdiWorkerScriptPath = path
	})
	return gdiWorkerScriptPath, gdiWorkerScriptErr
}

// RefreshGDIPrintWorker restarts the PowerShell GDI workers so the next print
// picks up current Windows printer preferences (darkness, speed, media).
func RefreshGDIPrintWorker() error {
	scriptPath, err := getGDIWorkerScriptPath()
	if err != nil {
		return fmt.Errorf("GDI worker skripti yozilmadi: %w", err)
	}
	// Rewrite embedded script in case binary was updated while process still runs.
	if err := os.WriteFile(scriptPath, gdiPrintWorkerPSScript, 0644); err != nil {
		return fmt.Errorf("GDI worker skripti yangilanmadi: %w", err)
	}

	gdiPool.ensure()
	var firstErr error
	for _, w := range gdiPool.all {
		w.mu.Lock()
		w.stopLocked()
		if err := w.ensureStartedLocked(); err != nil && firstErr == nil {
			firstErr = err
		}
		w.mu.Unlock()
	}
	if firstErr != nil {
		return fmt.Errorf("GDI worker qayta ishga tushmadi: %w", firstErr)
	}
	return nil
}

func (w *gdiPrintWorker) stopLocked() {
	// Bumping the generation first stops the supervisor of the process we are
	// about to kill from clobbering a replacement.
	w.generation++
	if w.stdin != nil {
		_, _ = io.WriteString(w.stdin, "EXIT\n")
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
func (w *gdiPrintWorker) markExited(generation uint64) {
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

func (w *gdiPrintWorker) ensureStartedLocked() error {
	if w.cmd != nil && w.stdin != nil && w.reader != nil {
		return nil
	}

	w.stopLocked()

	scriptPath, err := getGDIWorkerScriptPath()
	if err != nil {
		return fmt.Errorf("GDI worker skripti yozilmadi: %w", err)
	}

	cmd := exec.Command(
		"powershell",
		"-NoProfile",
		"-NonInteractive",
		"-STA",
		"-ExecutionPolicy", "Bypass",
		"-File", scriptPath,
	)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_ = stdin.Close()
		return err
	}

	if err := cmd.Start(); err != nil {
		_ = stdin.Close()
		return fmt.Errorf("GDI worker ishga tushmadi: %w", err)
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

func (w *gdiPrintWorker) print(job gdiPrintJob) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.ensureStartedLocked(); err != nil {
		return fmt.Errorf("%w: %v", errRawWorkerUnavailable, err)
	}

	job.ReqID = nextPrintRequestID()
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	if _, err := w.stdin.Write(append(payload, '\n')); err != nil {
		w.stopLocked()
		return fmt.Errorf("%w: %s workerga yozib bo'lmadi: %v", errRawWorkerUnavailable, w.name(), err)
	}

	reply, err := w.readReplyLocked(job.ReqID)
	if err != nil {
		// The job was already handed to PowerShell, so it may have reached the
		// spooler. Never treat this as safe to resend.
		w.stopLocked()
		return err
	}
	if !reply.ok {
		return errors.New(reply.msg)
	}
	return nil
}

// readReplyLocked waits for the reply to reqID, discarding replies to earlier
// requests that were abandoned on timeout.
func (w *gdiPrintWorker) readReplyLocked(reqID uint64) (rawWorkerReply, error) {
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

func printImageWindows(
	printerName,
	documentName,
	imagePath,
	outputPath string,
	widthMm,
	heightMm float64,
	copies,
	rotationDeg int,
) error {
	if widthMm <= 0 || heightMm <= 0 {
		return errors.New("etiketka o'lchami noto'g'ri")
	}
	if copies < 1 {
		copies = 1
	}
	if strings.TrimSpace(documentName) == "" {
		documentName = "LabelV2"
	}

	job := gdiPrintJob{
		ImagePath:      imagePath,
		PrinterName:    printerName,
		OutputPath:     outputPath,
		WidthMm:        widthMm,
		HeightMm:       heightMm,
		Copies:         copies,
		DocumentName:   documentName,
		RotationDeg:    normalizePrintRotationDeg(rotationDeg),
		ForcePaperSize: gdiShouldForcePaperSize(printerName, outputPath),
	}

	w, err := gdiPool.acquire()
	if err != nil {
		return err
	}
	defer gdiPool.release(w)

	err = w.print(job)
	if err == nil {
		return nil
	}
	// A dead worker fails before touching the spooler, so one respawn+resend is
	// safe. Any other failure may already have produced a label.
	if errors.Is(err, errRawWorkerUnavailable) {
		if err = w.print(job); err == nil {
			return nil
		}
	}
	return fmt.Errorf("printerga yuborishda xatolik: %w", err)
}

func gdiShouldForcePaperSize(printerName, outputPath string) bool {
	if strings.TrimSpace(outputPath) != "" {
		return true
	}
	n := strings.ToLower(strings.TrimSpace(printerName))
	if strings.Contains(n, "print to pdf") || strings.Contains(n, "microsoft pdf") {
		return true
	}
	if strings.Contains(n, "onenote") || strings.Contains(n, "xps") || strings.Contains(n, "fax") {
		return true
	}
	return false
}
