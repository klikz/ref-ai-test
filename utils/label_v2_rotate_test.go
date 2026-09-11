package utils

import (
	"image"
	"testing"
)

func TestNormalizeElementRotationDeg(t *testing.T) {
	cases := []struct {
		in, want int
	}{
		{0, 0},
		{90, 90},
		{180, 180},
		{270, 270},
		{45, 0},
		{-90, 0},
		{360, 0},
	}
	for _, tc := range cases {
		if got := normalizeElementRotationDeg(tc.in); got != tc.want {
			t.Fatalf("normalizeElementRotationDeg(%d)=%d, want %d", tc.in, got, tc.want)
		}
	}
}

func TestEffectiveLabelDimensionsAlbumSwap(t *testing.T) {
	w, h := effectiveLabelDimensions(72, 105, 90)
	if w != 105 || h != 72 {
		t.Fatalf("album media: got %.0fx%.0f, want 105x72", w, h)
	}
	w, h = effectiveLabelDimensions(72, 105, 0)
	if w != 72 || h != 105 {
		t.Fatalf("portrait media: got %.0fx%.0f, want 72x105", w, h)
	}
}

func TestPreparePrintImageSwapsPixels(t *testing.T) {
	src := image.NewGray(image.Rect(0, 0, 72, 105))
	out, rot := preparePrintImage(src, 90)
	if rot != 0 {
		t.Fatalf("gdi rotation after prepare should be 0, got %d", rot)
	}
	b := out.Bounds()
	if b.Dx() != 105 || b.Dy() != 72 {
		t.Fatalf("rotated bitmap: got %dx%d, want 105x72", b.Dx(), b.Dy())
	}
}

func TestResolveEffectivePrintRoute(t *testing.T) {
	lang, sizeOnly := resolveEffectivePrintRoute("zpl", "Zebra ZT411", true, false)
	if lang != PrintLanguageZPL || !sizeOnly {
		t.Fatalf("thermal+defaults: lang=%s sizeOnly=%v", lang, sizeOnly)
	}
	lang, sizeOnly = resolveEffectivePrintRoute("gdi", "Microsoft Print to PDF", true, false)
	if lang != PrintLanguageGDI || sizeOnly {
		t.Fatalf("gdi+defaults: lang=%s sizeOnly=%v", lang, sizeOnly)
	}
	lang, sizeOnly = resolveEffectivePrintRoute("zpl", "AC-FILE-ZPL", true, false)
	if lang != PrintLanguageZPL || sizeOnly {
		t.Fatalf("ac-file keeps sizeOnly from template: lang=%s sizeOnly=%v", lang, sizeOnly)
	}
}
