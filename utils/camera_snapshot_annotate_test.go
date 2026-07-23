package utils

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func TestAnnotateT3ScanPhotoJPEG(t *testing.T) {
	if t3ScanAnnotateFontPath() == "" {
		t.Skip("consola font yo'q")
	}

	src := image.NewRGBA(image.Rect(0, 0, 320, 240))
	for y := 0; y < 240; y++ {
		for x := 0; x < 320; x++ {
			src.Set(x, y, color.RGBA{uint8(x % 256), uint8(y % 256), 80, 255})
		}
	}
	var raw bytes.Buffer
	if err := jpeg.Encode(&raw, src, &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}

	out, err := annotateT3ScanPhotoJPEG(raw.Bytes(), "SN12345")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) < len(raw.Bytes())/2 {
		t.Fatalf("annotated jpeg too small: %d", len(out))
	}
}
