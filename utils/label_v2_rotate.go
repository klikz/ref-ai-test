package utils

import (
	"image"
)

func normalizePrintRotationDeg(deg int) int {
	if deg == 90 {
		return 90
	}
	return 0
}

// normalizeElementRotationDeg snaps per-element rotation to 0/90/180/270.
func normalizeElementRotationDeg(deg int) int {
	switch deg {
	case 90, 180, 270:
		return deg
	default:
		return 0
	}
}

func effectiveLabelDimensions(widthMm, heightMm float64, rotationDeg int) (float64, float64) {
	// 90° = albom: physical media is swapped (content is rotated to match).
	if normalizePrintRotationDeg(rotationDeg) == 90 {
		return heightMm, widthMm
	}
	return widthMm, heightMm
}

// preparePrintImage applies print rotation in software so the bitmap matches
// physical media (swapped W×H). Callers should pass rotationDeg=0 to GDI after this.
func preparePrintImage(img image.Image, rotationDeg int) (image.Image, int) {
	if normalizePrintRotationDeg(rotationDeg) != 90 {
		return img, 0
	}
	return rotateImage90CW(img), 0
}

func rotateImage90CW(src image.Image) image.Image {
	bounds := src.Bounds()
	w, h := bounds.Dx(), bounds.Dy()
	if w < 1 || h < 1 {
		return src
	}

	dst := image.NewRGBA(image.Rect(0, 0, h, w))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			dst.Set(h-1-y, x, src.At(bounds.Min.X+x, bounds.Min.Y+y))
		}
	}
	return dst
}
