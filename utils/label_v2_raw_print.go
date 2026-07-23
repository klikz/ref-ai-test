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

func luminanceOnWhite(c color.Color) float64 {
	r, g, b, a := c.RGBA()
	if a == 0 {
		return 1
	}
	af := float64(a) / 65535
	rr := float64(r)/65535 + (1-af)
	gg := float64(g)/65535 + (1-af)
	bb := float64(b)/65535 + (1-af)
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

func buildTSPLPayload(widthMm, heightMm float64, copies int, img image.Image, rotationDeg int) ([]byte, error) {
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

	printW, printH := widthMm, heightMm
	if normalizePrintRotationDeg(rotationDeg) == 90 {
		printW, printH = heightMm, widthMm
	}

	var buf bytes.Buffer
	fmt.Fprintf(&buf, "SIZE %.2f mm, %.2f mm\r\n", printW, printH)
	buf.WriteString("GAP 2 mm, 0 mm\r\n")
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

func buildZPLPayload(copies int, img image.Image, labelWidthDots, labelHeightDots, rotationDeg int) ([]byte, error) {
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

	var buf bytes.Buffer
	buf.WriteString("^XA\r\n")
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
