package utils

import (
	"image"
	"strings"
	"testing"
)

func TestRawPrinterDPIHint(t *testing.T) {
	tests := []struct {
		name        string
		templateDPI int
		language    string
		printer     string
		want        int
	}{
		{name: "gdi keeps template dpi", templateDPI: 300, language: PrintLanguageGDI, printer: "Canon MF440", want: 300},
		{name: "gprinter forces 203", templateDPI: 300, language: PrintLanguageTSPL, printer: "Gprinter GP-1124T", want: 203},
		{name: "xprinter forces 203", templateDPI: 300, language: PrintLanguageZPL, printer: "XPrinter XP-420B", want: 203},
		{name: "300dpi model kept", templateDPI: 203, language: PrintLanguageTSPL, printer: "TSC TE300", want: 300},
		{name: "unknown thermal keeps valid template dpi", templateDPI: 300, language: PrintLanguageZPL, printer: "Zebra ZT411", want: 300},
		{name: "bad dpi falls back to 203", templateDPI: 0, language: PrintLanguageTSPL, printer: "Some TSPL", want: 203},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := rawPrinterDPIHint(tt.templateDPI, tt.language, tt.printer); got != tt.want {
				t.Fatalf("rawPrinterDPIHint(%d, %q, %q) = %d, want %d", tt.templateDPI, tt.language, tt.printer, got, tt.want)
			}
		})
	}
}

func TestBuildTSPLPayloadRejectsMissingGapWhenApply(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	_, err := buildTSPLPayload(40, 30, 1, img, 0, RawPrintSettings{
		Apply:   true,
		Density: 8,
		Speed:   4,
		GapMm:   0,
	})
	if err == nil || !strings.Contains(err.Error(), "gap_mm") {
		t.Fatalf("gap_mm xatosi kutilgan, got: %v", err)
	}
}

func TestBuildTSPLPayloadOmitsGapWhenDefaultsUnknown(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	payload, err := buildTSPLPayload(40, 30, 2, img, 0, RawPrintSettings{
		Apply:   false,
		Density: 8,
		Speed:   4,
		GapMm:   0,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	text := string(payload)
	if strings.Contains(text, "GAP ") {
		t.Fatalf("GAP buyrug'i bo'lmasligi kerak:\n%s", text)
	}
	if !strings.Contains(text, "SIZE 40.00 mm, 30.00 mm") {
		t.Fatalf("SIZE topilmadi:\n%s", text)
	}
	if !strings.Contains(text, "PRINT 2,1") {
		t.Fatalf("PRINT topilmadi:\n%s", text)
	}
}

func TestBuildTSPLPayloadKeepsExplicitGap(t *testing.T) {
	img := image.NewGray(image.Rect(0, 0, 10, 10))
	payload, err := buildTSPLPayload(72, 105, 1, img, 0, RawPrintSettings{
		Apply:   false,
		Density: 8,
		Speed:   4,
		GapMm:   3,
	})
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if !strings.Contains(string(payload), "GAP 3.00 mm, 0 mm") {
		t.Fatalf("explicit GAP yuborilmadi:\n%s", string(payload))
	}
}
