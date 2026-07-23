package utils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
)

func annotateT3ScanPhotoJPEG(data []byte, serial string) ([]byte, error) {
	img, _, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode image: %w", err)
	}

	dc := gg.NewContextForImage(img)
	height := float64(dc.Height())

	fontSize := height / 18
	if fontSize < 14 {
		fontSize = 14
	}
	if fontSize > 72 {
		fontSize = 72
	}

	fontPath := t3ScanAnnotateFontPath()
	if fontPath == "" {
		return nil, fmt.Errorf("annotate font topilmadi")
	}
	if err := dc.LoadFontFace(fontPath, fontSize); err != nil {
		return nil, fmt.Errorf("load font: %w", err)
	}

	padding := fontSize * 0.55
	margin := fontSize * 0.4
	fontH := dc.FontHeight()
	textW, _ := dc.MeasureString(serial)
	boxW := textW + padding*2
	boxH := fontH + padding*2
	boxY := height - boxH - margin

	dc.SetRGBA(0, 0, 0, 0.6)
	dc.DrawRectangle(0, boxY, boxW, boxH)
	dc.Fill()

	dc.SetRGB(1, 1, 1)
	dc.DrawStringAnchored(serial, padding, boxY+boxH/2, 0, 0.5)

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, dc.Image(), &jpeg.Options{Quality: 92}); err != nil {
		return nil, fmt.Errorf("encode jpeg: %w", err)
	}
	return buf.Bytes(), nil
}

func t3ScanAnnotateFontPath() string {
	rel := filepath.Join("fonts", "consola", "CONSOLAB.TTF")
	candidates := []string{rel}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, rel))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), rel))
	}
	for _, path := range candidates {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}
