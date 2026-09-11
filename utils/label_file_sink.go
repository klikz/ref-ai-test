package utils

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Label file-sink printers (no Windows queue): names start with "AC-FILE".
// Example: AC-FILE-ZPL, AC-FILE-TSPL — payload is built as ZPL/TSPL, output is PNG under print_preview/.

func IsLabelFileSink(printerName string) bool {
	n := strings.ToUpper(strings.TrimSpace(printerName))
	return strings.HasPrefix(n, "AC-FILE")
}

var (
	labelFileOutOnce sync.Once
	labelFileOutDir  string
)

func labelFileOutputDir() string {
	labelFileOutOnce.Do(func() {
		if v := strings.TrimSpace(os.Getenv("LABEL_FILE_OUTPUT_DIR")); v != "" {
			if abs, err := filepath.Abs(v); err == nil {
				labelFileOutDir = abs
			} else {
				labelFileOutDir = v
			}
			return
		}
		// Prefer process working directory (make/dev), then executable dir (Windows service).
		candidates := make([]string, 0, 2)
		if wd, err := os.Getwd(); err == nil && strings.TrimSpace(wd) != "" {
			candidates = append(candidates, filepath.Join(wd, "print_preview"))
		}
		if exe, err := os.Executable(); err == nil {
			candidates = append(candidates, filepath.Join(filepath.Dir(exe), "print_preview"))
		}
		for _, c := range candidates {
			if err := os.MkdirAll(c, 0755); err == nil {
				if abs, err := filepath.Abs(c); err == nil {
					labelFileOutDir = abs
				} else {
					labelFileOutDir = c
				}
				return
			}
		}
		labelFileOutDir = "print_preview"
	})
	return labelFileOutDir
}

// saveLabelFileSink writes a PNG preview (and optional RAW .zpl/.tspl) into the project output folder.
// img must already be in final media orientation (album = pre-rotated / swapped W×H).
func saveLabelFileSink(
	printerName, serial, language string,
	img image.Image,
	rotationDeg int,
	payload []byte,
) (string, error) {
	outImg := img
	if normalizePrintRotationDeg(rotationDeg) == 90 {
		outImg = rotateImage90CW(img)
	}

	dir := labelFileOutputDir()
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("preview papka yaratilmadi: %w", err)
	}

	bounds := outImg.Bounds()
	lang := NormalizePrintLanguage(language)
	base := fmt.Sprintf(
		"%s-%s-%s-%dx%d-%d",
		sanitizeLabelFilenamePart(strings.TrimSpace(printerName)),
		sanitizeLabelFilenamePart(serial),
		lang,
		bounds.Dx(),
		bounds.Dy(),
		time.Now().UnixNano(),
	)
	pngPath := filepath.Join(dir, base+".png")

	f, err := os.Create(pngPath)
	if err != nil {
		return "", fmt.Errorf("PNG yaratilmadi: %w", err)
	}
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(f, outImg); err != nil {
		_ = f.Close()
		_ = os.Remove(pngPath)
		return "", fmt.Errorf("PNG yozilmadi: %w", err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(pngPath)
		return "", err
	}

	abs, _ := filepath.Abs(pngPath)
	fmt.Fprintf(os.Stderr, "print file-sink: %s (%dx%d)\n", abs, bounds.Dx(), bounds.Dy())

	if len(payload) > 0 {
		ext := ".raw"
		switch lang {
		case PrintLanguageZPL:
			ext = ".zpl"
		case PrintLanguageTSPL:
			ext = ".tspl"
		}
		rawPath := filepath.Join(dir, base+ext)
		if err := os.WriteFile(rawPath, payload, 0644); err != nil {
			return pngPath, fmt.Errorf("RAW fayl yozilmadi: %w", err)
		}
	}

	return pngPath, nil
}
