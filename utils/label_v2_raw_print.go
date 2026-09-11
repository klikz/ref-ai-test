package utils

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"image"
	"image/color"
	_ "image/png"
	"strings"
)

const monochromeBlackThreshold = 0.5

// RawPrintSettings controls darkness/speed/gap for TSPL/ZPL jobs.
// When Apply is false, firmware/printer defaults are left unchanged
// (only SIZE / bitmap / PRINT are sent).
type RawPrintSettings struct {
	Apply   bool
	Density int     // TSPL DENSITY 0-15; ZPL ^MD relative (-30..30), typically 0..30 absolute via ~SD
	Speed   int     // inches/sec style integer for TSPL SPEED / ZPL ^PR
	GapMm   float64 // TSPL GAP
}

func luminanceOnWhite(c color.Color) float64 {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return 1
	}
	af := float64(a) / 65535
	rr := float64(r)/65535 + (1 - af)
	gg := float64(g)/65535 + (1 - af)
	bb := float64(b)/65535 + (1 - af)
	return 0.299*rr + 0.587*gg + 0.114*bb
}

func imageToMonochromePacked(img image.Image) (width, height, rowBytes int, data []byte) {
	bounds := img.Bounds()
	width = bounds.Dx()
	height = bounds.Dy()
	if width < 1 || height < 1 {
		return 0, 0, 0, nil
	}

	rowBytes = (width + 7) / 8
	data = make([]byte, rowBytes*height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if luminanceOnWhite(img.At(bounds.Min.X+x, bounds.Min.Y+y)) < monochromeBlackThreshold {
				byteIndex := y*rowBytes + x/8
				bit := uint(7 - (x % 8))
				data[byteIndex] |= 1 << bit
			}
		}
	}

	return width, height, rowBytes, data
}

func clampInt(v, lo, hi int) int {
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}

// rawPrinterDPIHint picks the dot density that RAW thermal printers should use.
// GDI can rely on Windows scaling, but TSPL/ZPL bitmaps must match the printer
// head resolution or they spill into the next label. Gprinter/XPrinter class
// devices in production are overwhelmingly 203 DPI, so that family is forced to
// 203 even if the template was accidentally saved as 300.
func rawPrinterDPIHint(templateDPI int, language, printerName string) int {
	if templateDPI <= 0 {
		templateDPI = 203
	}
	switch NormalizePrintLanguage(language) {
	case PrintLanguageTSPL, PrintLanguageZPL:
	default:
		return templateDPI
	}

	printerLower := strings.ToLower(strings.TrimSpace(printerName))
	if _, ok := matchGprinterOrXPrinter("", printerLower); ok {
		return 203
	}
	threeHundredMarkers := []string{
		"300dpi", "300 dpi", "te300", "ta300", "tx300", "mh340", "mh640",
		"zt230-300", "zt410-300", "zt411-300", "gx430", "gk420t",
	}
	for _, marker := range threeHundredMarkers {
		if strings.Contains(printerLower, marker) {
			return 300
		}
	}
	if templateDPI == 203 || templateDPI == 300 {
		return templateDPI
	}
	return 203
}

func buildTSPLPayload(widthMm, heightMm float64, copies int, img image.Image, rotationDeg int, settings RawPrintSettings) ([]byte, error) {
	if copies < 1 {
		copies = 1
	}
	if normalizePrintRotationDeg(rotationDeg) == 90 {
		img = rotateImage90CW(img)
	}

	widthPx, height, rowBytes, bitmap := imageToMonochromePacked(img)
	if len(bitmap) == 0 {
		return nil, fmt.Errorf("bitmap bo'sh")
	}
	// Gprinter/TSC TSPL BITMAP: packed bit 1 is treated as white on many firmwares,
	// so invert after packing (ZPL ^GFA keeps the original polarity).
	for i := range bitmap {
		bitmap[i] ^= 0xFF
	}

	printW, printH := widthMm, heightMm
	// Media size: design W×H; after 90° software rotate the bitmap is H×W —
	// SIZE must follow the bitmap (albom qog'oz).
	if normalizePrintRotationDeg(rotationDeg) == 90 {
		printW, printH = heightMm, widthMm
	}

	gapMm := settings.GapMm
	sendGap := gapMm > 0
	density := clampInt(settings.Density, 0, 15)
	speed := clampInt(settings.Speed, 1, 14)

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "SIZE %.2f mm, %.2f mm\r\n", printW, printH)
	if settings.Apply {
		if !sendGap {
			return nil, fmt.Errorf("gap_mm 0 dan katta bo'lishi kerak (TSPL)")
		}
		fmt.Fprintf(&buf, "GAP %.2f mm, 0 mm\r\n", gapMm)
		fmt.Fprintf(&buf, "DENSITY %d\r\n", density)
		fmt.Fprintf(&buf, "SPEED %d\r\n", speed)
	} else if sendGap {
		// size_only / printer defaults mode can still carry the physical media gap,
		// but only when the template explicitly knows it.
		fmt.Fprintf(&buf, "GAP %.2f mm, 0 mm\r\n", gapMm)
	}
	buf.WriteString("DIRECTION 1\r\n")
	buf.WriteString("REFERENCE 0,0\r\n")
	buf.WriteString("OFFSET 0 mm\r\n")
	buf.WriteString("SET PEEL OFF\r\n")
	buf.WriteString("SET CUTTER OFF\r\n")
	buf.WriteString("SET PARTIAL_CUTTER OFF\r\n")
	buf.WriteString("SET TEAR ON\r\n")
	buf.WriteString("CLS\r\n")
	fmt.Fprintf(&buf, "BITMAP 0,0,%d,%d,0,", rowBytes, height)
	buf.Write(bitmap)
	fmt.Fprintf(&buf, "\r\nPRINT %d,1\r\n", copies)
	_ = widthPx
	return buf.Bytes(), nil
}

func buildZPLPayload(copies int, img image.Image, labelWidthDots, labelHeightDots, rotationDeg int, settings RawPrintSettings) ([]byte, error) {
	if copies < 1 {
		copies = 1
	}
	if labelWidthDots < 1 || labelHeightDots < 1 {
		return nil, fmt.Errorf("noto'g'ri etiketka o'lchami")
	}

	if normalizePrintRotationDeg(rotationDeg) == 90 {
		// ZPL ^FWR does not rotate ^GFA bitmaps; rotate in software (same as TSPL/preview).
		img = rotateImage90CW(img)
	}

	widthPx, height, rowBytes, bitmap := imageToMonochromePacked(img)
	if len(bitmap) == 0 {
		return nil, fmt.Errorf("bitmap bo'sh")
	}
	expectedW, expectedH := labelWidthDots, labelHeightDots
	if normalizePrintRotationDeg(rotationDeg) == 90 {
		expectedW, expectedH = labelHeightDots, labelWidthDots
	}
	if widthPx != expectedW || height != expectedH {
		return nil, fmt.Errorf(
			"bitmap o'lchami mos emas: %dx%d, kutilgan %dx%d",
			widthPx, height, expectedW, expectedH,
		)
	}

	totalBytes := len(bitmap)
	hexData := strings.ToUpper(hex.EncodeToString(bitmap))

	pwDots := labelWidthDots
	llDots := labelHeightDots
	if normalizePrintRotationDeg(rotationDeg) == 90 {
		// Rotated bitmap is H×W; label format must match or Zebra clips along feed.
		pwDots, llDots = labelHeightDots, labelWidthDots
	}

	density := clampInt(settings.Density, 0, 30)
	speed := clampInt(settings.Speed, 1, 14)

	var buf bytes.Buffer
	buf.WriteString("^XA\r\n")
	if settings.Apply {
		// ~SD sets absolute darkness for the session; ^PR sets print speed.
		fmt.Fprintf(&buf, "~SD%d\r\n", density)
		fmt.Fprintf(&buf, "^PR%d\r\n", speed)
	}
	buf.WriteString("^MMT\r\n")
	buf.WriteString("^LH0,0\r\n")
	buf.WriteString("^LT0\r\n")
	buf.WriteString("^LS0\r\n")
	fmt.Fprintf(&buf, "^PW%d\r\n", pwDots)
	fmt.Fprintf(&buf, "^LL%d\r\n", llDots)
	fmt.Fprintf(&buf, "^FO0,0^GFA,%d,%d,%d,%s^FS\r\n", totalBytes, totalBytes, rowBytes, hexData)
	fmt.Fprintf(&buf, "^PQ%d,0,1,Y\r\n", copies)
	buf.WriteString("^XZ\r\n")

	return buf.Bytes(), nil
}

func decodePNGToImage(pngBytes []byte) (image.Image, error) {
	img, _, err := image.Decode(bytes.NewReader(pngBytes))
	if err != nil {
		return nil, err
	}
	return img, nil
}

// RawSettingsFromTemplate maps label template print settings for RAW jobs.
// sizeOnly=true → Apply=false (faqat SIZE / ^PW /^LL; density/speed/gap yuborilmaydi).
func RawSettingsFromTemplate(density, speed int, gapMm float64, sizeOnly bool) RawPrintSettings {
	if sizeOnly {
		return RawPrintSettings{Apply: false, GapMm: gapMm, Density: density, Speed: speed}
	}
	return RawPrintSettings{
		Apply:   true,
		Density: density,
		Speed:   speed,
		GapMm:   gapMm,
	}
}
