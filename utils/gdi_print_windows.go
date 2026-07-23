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
	ImagePath    string  `json:"ImagePath"`
	PrinterName  string  `json:"PrinterName"`
	OutputPath   string  `json:"OutputPath,omitempty"`
	WidthMm      float64 `json:"WidthMm"`
	HeightMm     float64 `json:"HeightMm"`
	Copies       int     `json:"Copies"`
	DocumentName string  `json:"DocumentName"`
	RotationDeg  int     `json:"RotationDeg"`
}

type gdiPrintWorker struct {
	mu     sync.Mutex
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	reader *bufio.Reader
}

var (
	gdiWorkerScriptOnce sync.Once
	gdiWorkerScriptPath string
	gdiWorkerScriptErr  error
	gdiWorker           gdiPrintWorker
)

func init() {
	go func() {
		gdiWorker.mu.Lock()
		defer gdiWorker.mu.Unlock()
		_ = gdiWorker.ensureStartedLocked()
	}()
}

func getGDIWorkerScriptPath() (string, error) {
	gdiWorkerScriptOnce.Do(func() {
		path := filepath.Join(os.TempDir(), "ac-label-v2-gdi-worker-v2.ps1")
		gdiWorkerScriptErr = os.WriteFile(path, gdiPrintWorkerPSScript, 0644)
		gdiWorkerScriptPath = path
	})
	return gdiWorkerScriptPath, gdiWorkerScriptErr
}

func (w *gdiPrintWorker) stopLocked() {
	if w.stdin != nil {
		_, _ = io.WriteString(w.stdin, "EXIT\n")
		_ = w.stdin.Close()
		w.stdin = nil
	}
	if w.cmd != nil && w.cmd.Process != nil {
		_ = w.cmd.Process.Kill()
		_, _ = w.cmd.Process.Wait()
	}
	w.cmd = nil
	w.reader = nil
}

func (w *gdiPrintWorker) ensureStartedLocked() error {
	if w.cmd != nil && w.stdin != nil && w.reader != nil {
		if w.cmd.ProcessState == nil {
			return nil
		}
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

	reader := bufio.NewReader(stdout)
	readyCh := make(chan error, 1)
	go func() {
		line, readErr := reader.ReadString('\n')
		if readErr != nil {
			readyCh <- readErr
			return
		}
		if strings.TrimSpace(line) != "READY" {
			readyCh <- fmt.Errorf("GDI worker javobi kutilmagan: %q", strings.TrimSpace(line))
			return
		}
		readyCh <- nil
	}()

	select {
	case err := <-readyCh:
		if err != nil {
			_ = stdin.Close()
			_ = cmd.Process.Kill()
			_, _ = cmd.Process.Wait()
			return fmt.Errorf("GDI worker tayyor emas: %w", err)
		}
	case <-time.After(45 * time.Second):
		_ = stdin.Close()
		_ = cmd.Process.Kill()
		_, _ = cmd.Process.Wait()
		return errors.New("GDI worker ishga tushish vaqti tugadi")
	}

	w.cmd = cmd
	w.stdin = stdin
	w.reader = reader
	return nil
}

func (w *gdiPrintWorker) print(job gdiPrintJob) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if err := w.ensureStartedLocked(); err != nil {
		return err
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	if _, err := w.stdin.Write(append(payload, '\n')); err != nil {
		w.stopLocked()
		return fmt.Errorf("GDI workerga yozib bo'lmadi: %w", err)
	}

	line, err := w.reader.ReadString('\n')
	if err != nil {
		w.stopLocked()
		return fmt.Errorf("GDI worker javobi o'qilmadi: %w", err)
	}

	response := strings.TrimSpace(line)
	switch {
	case response == "OK":
		return nil
	case strings.HasPrefix(response, "ERR:"):
		return errors.New(strings.TrimPrefix(response, "ERR:"))
	default:
		w.stopLocked()
		return fmt.Errorf("GDI worker noma'lum javob: %s", response)
	}
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
		ImagePath:    imagePath,
		PrinterName:  printerName,
		OutputPath:   outputPath,
		WidthMm:      widthMm,
		HeightMm:     heightMm,
		Copies:       copies,
		DocumentName: documentName,
		RotationDeg:  normalizePrintRotationDeg(rotationDeg),
	}

	if err := gdiWorker.print(job); err != nil {
		if retryErr := gdiWorker.print(job); retryErr != nil {
			return fmt.Errorf("printerga yuborishda xatolik: %w", retryErr)
		}
	}
	return nil
}
