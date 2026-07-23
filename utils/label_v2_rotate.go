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

func effectiveLabelDimensions(widthMm, heightMm float64, rotationDeg int) (float64, float64) {
	// Physical label/media size stays the same; rotation applies to content only.
	_ = rotationDeg
	return widthMm, heightMm
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
