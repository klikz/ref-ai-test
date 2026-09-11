package utils

import (
	"fmt"
	"os"
	"strconv"
	"sync"

	"github.com/golang/freetype/truetype"
	"golang.org/x/image/font"
)

// Font handling for label rendering.
//
// gg.LoadFontFace re-reads and re-parses the TTF on every call, and
// drawTextInBox calls it once per step of the auto-shrink search. On a table
// label that is hundreds of ~1 MB reads per render. Two caches fix it:
//
//   - parsed *truetype.Font, cached globally (immutable, safe to share)
//   - font.Face, cached per render (truetype faces keep a mutable glyph cache,
//     so they must not be shared between concurrent renders)

type parsedFontEntry struct {
	font *truetype.Font
	err  error
}

var (
	parsedFonts       sync.Map // path -> *parsedFontEntry
	resolvedFontPaths sync.Map // cache key -> resolved font path
)

func loadParsedFont(path string) (*truetype.Font, error) {
	if cached, ok := parsedFonts.Load(path); ok {
		entry := cached.(*parsedFontEntry)
		return entry.font, entry.err
	}
	entry := &parsedFontEntry{}
	raw, err := os.ReadFile(path)
	if err != nil {
		entry.err = fmt.Errorf("shrift o'qilmadi (%s): %w", path, err)
	} else if parsed, parseErr := truetype.Parse(raw); parseErr != nil {
		entry.err = fmt.Errorf("shrift parse qilinmadi (%s): %w", path, parseErr)
	} else {
		entry.font = parsed
	}
	actual, _ := parsedFonts.LoadOrStore(path, entry)
	stored := actual.(*parsedFontEntry)
	return stored.font, stored.err
}

// labelFontCache holds faces for a single renderLabelImage call. It is not safe
// for concurrent use — that is intentional, see the note above.
type labelFontCache struct {
	faces map[string]font.Face
}

func newLabelFontCache() *labelFontCache {
	return &labelFontCache{faces: make(map[string]font.Face, 8)}
}

// face returns a face for path at pixelSize, matching what gg.LoadFontFace
// would have produced (truetype.NewFace with Size set and default hinting).
func (c *labelFontCache) face(path string, pixelSize float64) (font.Face, error) {
	key := path + "|" + strconv.FormatFloat(pixelSize, 'f', 4, 64)
	if f, ok := c.faces[key]; ok {
		return f, nil
	}
	parsed, err := loadParsedFont(path)
	if err != nil {
		return nil, err
	}
	f := truetype.NewFace(parsed, &truetype.Options{Size: pixelSize})
	c.faces[key] = f
	return f, nil
}

// cachedResolveLabelFontPath memoizes the bold/regular variant lookup, which
// otherwise hits os.Stat for every element and every table cell.
func cachedResolveLabelFontPath(fontPath, fontWeight string) string {
	key := fontPath + "|" + fontWeight
	if v, ok := resolvedFontPaths.Load(key); ok {
		return v.(string)
	}
	resolved := resolveLabelFontPath(fontPath, fontWeight)
	resolvedFontPaths.Store(key, resolved)
	return resolved
}

// cachedLabelElementFontPath memoizes family -> file resolution (also os.Stat).
func cachedLabelElementFontPath(fontFamily string) string {
	key := "family|" + fontFamily
	if v, ok := resolvedFontPaths.Load(key); ok {
		return v.(string)
	}
	resolved := labelElementFontPath(fontFamily)
	resolvedFontPaths.Store(key, resolved)
	return resolved
}

// measureLabelLines reproduces gg.Context.MeasureMultilineString exactly, using
// the LoadFontFace line-height convention (pixelSize * 72 / 96) rather than the
// face metrics that gg.SetFontFace would derive.
func measureLabelLines(face font.Face, fontHeight float64, lines []string, lineSpacing float64) (width, height float64) {
	height = float64(len(lines)) * fontHeight * lineSpacing
	height -= (lineSpacing - 1) * fontHeight

	d := &font.Drawer{Face: face}
	for _, line := range lines {
		if w := float64(d.MeasureString(line) >> 6); w > width {
			width = w
		}
	}
	return width, height
}

func measureLabelLineWidth(face font.Face, line string) float64 {
	d := &font.Drawer{Face: face}
	return float64(d.MeasureString(line) >> 6)
}

// labelFontHeight is gg's LoadFontFace line height for a given pixel size.
func labelFontHeight(pixelSize float64) float64 {
	return pixelSize * 72 / 96
}
