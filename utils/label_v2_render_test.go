package utils

import (
	"bytes"
	"flag"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"

	"github.com/klikz/api_v3/internal/models"
)

var updateGolden = flag.Bool("update-golden", false, "regenerate golden label PNGs")

// renderFixture is a production-like label: mixed text sizes/weights, a table
// with per-cell fonts, and the three code symbologies. Bindings that depend on
// wall-clock time are avoided so the render stays byte-deterministic.
type renderFixture struct {
	name     string
	template models.LabelTemplate
	def      labelDefinition
}

func labelRenderFixtures() []renderFixture {
	return []renderFixture{
		{
			name:     "text_and_barcode",
			template: models.LabelTemplate{WidthMm: 100, HeightMm: 50, DPI: 203},
			def: labelDefinition{
				Version: 1,
				Elements: []labelElement{
					{Type: "text", X: 2, Y: 2, Width: 60, Height: 8, StaticText: "PREMIER AC", FontSize: 14, FontWeight: "bold", ZIndex: 1},
					{Type: "text", X: 2, Y: 11, Width: 60, Height: 6, DataSource: "backend", Binding: "model.model_nomi", FontSize: 9, ZIndex: 2},
					{Type: "text", X: 2, Y: 18, Width: 60, Height: 6, DataSource: "backend", Binding: "serial", FontSize: 11, FontWeight: "bold", Align: "left", ZIndex: 3},
					{Type: "text", X: 64, Y: 2, Width: 34, Height: 6, StaticText: "Made in Uzbekistan", FontSize: 7, Align: "right", ZIndex: 4},
					{Type: "line", X: 2, Y: 25, Width: 96, Height: 1, StrokeWidth: 0.4, ZIndex: 5},
					{Type: "barcode", X: 4, Y: 28, Width: 60, Height: 18, DataSource: "backend", Binding: "serial", Format: "code128", ZIndex: 6},
					{Type: "qrcode", X: 70, Y: 28, Width: 18, Height: 18, DataSource: "backend", Binding: "serial", ZIndex: 7},
				},
			},
		},
		{
			// Tables are the worst case for font handling: every cell runs the
			// auto-shrink search independently.
			name:     "table_heavy",
			template: models.LabelTemplate{WidthMm: 100, HeightMm: 60, DPI: 203},
			def: labelDefinition{
				Version: 1,
				Elements: []labelElement{
					{Type: "text", X: 2, Y: 1, Width: 96, Height: 7, StaticText: "TEXNIK XARAKTERISTIKA", FontSize: 12, FontWeight: "bold", Align: "center", ZIndex: 1},
					{
						Type: "table", X: 2, Y: 9, Width: 96, Height: 34,
						Rows: 4, Cols: 3, StrokeWidth: 0.3, FontSize: 8, ZIndex: 2,
						ColWidths: []float64{40, 30, 26},
						Cells: []labelTableCell{
							{StaticText: "Parametr", FontWeight: "bold", Align: "center", FillColor: "#EEEEEE"},
							{StaticText: "Qiymat", FontWeight: "bold", Align: "center", FillColor: "#EEEEEE"},
							{StaticText: "Birlik", FontWeight: "bold", Align: "center", FillColor: "#EEEEEE"},
							{StaticText: "Quvvat", Align: "left"}, {StaticText: "9000", Align: "center"}, {StaticText: "BTU", Align: "center"},
							{StaticText: "Kuchlanish", Align: "left"}, {StaticText: "220-240", Align: "center"}, {StaticText: "V", Align: "center"},
							{StaticText: "Seriya", Align: "left"}, {DataSource: "backend", Binding: "serial", Align: "center"}, {StaticText: "-", Align: "center"},
						},
					},
					{Type: "datamatrix", X: 2, Y: 45, Width: 14, Height: 14, DataSource: "backend", Binding: "gscode.data", ZIndex: 3},
					{Type: "text", X: 20, Y: 45, Width: 78, Height: 14, DataSource: "backend", Binding: "gscode.data", FontSize: 8, ZIndex: 4},
				},
			},
		},
	}
}

func renderFixtureImage(t testing.TB, f renderFixture) image.Image {
	t.Helper()
	img, err := renderLabelImage(f.template, f.def, BuildLabelPreviewSampleData())
	if err != nil {
		t.Fatalf("render %s: %v", f.name, err)
	}
	return img
}

func encodePNG(t testing.TB, img image.Image) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatalf("png encode: %v", err)
	}
	return buf.Bytes()
}

// TestLabelRenderGolden pins the rendered output so font-caching and layout
// refactors can be proven pixel-identical.
//
// Golden files depend on the installed Windows font versions. On a machine with
// different fonts, regenerate once:
//
//	go test ./utils -run TestLabelRenderGolden -update-golden
func TestLabelRenderGolden(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows fonts")
	}

	for _, f := range labelRenderFixtures() {
		t.Run(f.name, func(t *testing.T) {
			got := encodePNG(t, renderFixtureImage(t, f))
			goldenPath := filepath.Join("testdata", "golden_"+f.name+".png")

			if *updateGolden {
				if err := os.MkdirAll("testdata", 0o755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(goldenPath, got, 0o644); err != nil {
					t.Fatal(err)
				}
				t.Logf("golden yangilandi: %s (%d bayt)", goldenPath, len(got))
				return
			}

			want, err := os.ReadFile(goldenPath)
			if err != nil {
				t.Fatalf("golden o'qilmadi (%s): %v; -update-golden bilan yarating", goldenPath, err)
			}
			if !bytes.Equal(got, want) {
				failPath := filepath.Join("testdata", "failed_"+f.name+".png")
				_ = os.WriteFile(failPath, got, 0o644)
				t.Errorf("render golden bilan mos emas; natija: %s", failPath)
			}
		})
	}
}

// TestLabelRenderConcurrent guards the font cache: parsed fonts are shared
// globally while faces must stay per-render, because a truetype face keeps a
// mutable glyph cache. A leaked shared face shows up as corrupted output here.
func TestLabelRenderConcurrent(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows fonts")
	}

	for _, f := range labelRenderFixtures() {
		t.Run(f.name, func(t *testing.T) {
			want, err := os.ReadFile(filepath.Join("testdata", "golden_"+f.name+".png"))
			if err != nil {
				t.Skipf("golden yo'q: %v", err)
			}

			const goroutines = 16
			results := make(chan []byte, goroutines)
			var wg sync.WaitGroup
			for i := 0; i < goroutines; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					img, err := renderLabelImage(f.template, f.def, BuildLabelPreviewSampleData())
					if err != nil {
						results <- nil
						return
					}
					var buf bytes.Buffer
					if err := png.Encode(&buf, img); err != nil {
						results <- nil
						return
					}
					results <- buf.Bytes()
				}()
			}
			wg.Wait()
			close(results)

			for got := range results {
				if got == nil {
					t.Fatal("parallel render xato qaytardi")
				}
				if !bytes.Equal(got, want) {
					t.Fatal("parallel render natijasi golden bilan mos emas (shrift face ulashilgan bo'lishi mumkin)")
				}
			}
		})
	}
}

func BenchmarkRenderLabelImage(b *testing.B) {
	if runtime.GOOS != "windows" {
		b.Skip("windows fonts")
	}
	for _, f := range labelRenderFixtures() {
		b.Run(f.name, func(b *testing.B) {
			data := BuildLabelPreviewSampleData()
			b.ReportAllocs()
			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				if _, err := renderLabelImage(f.template, f.def, data); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}

// TestLabelTextHorizontalSqueeze verifies long single-line text is compressed
// horizontally into the box instead of overflowing past the right edge.
func TestLabelTextHorizontalSqueeze(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows fonts")
	}

	const (
		boxXmm = 2.0
		boxYmm = 5.0
		boxWmm = 25.0
		boxHmm = 8.0
		dpi    = 203
	)

	tmpl := models.LabelTemplate{WidthMm: 60, HeightMm: 30, DPI: dpi}
	def := labelDefinition{
		Version: 1,
		Elements: []labelElement{
			{
				Type:       "text",
				X:          boxXmm,
				Y:          boxYmm,
				Width:      boxWmm,
				Height:     boxHmm,
				StaticText: "VERYLONGSERIALNUMBER1234567890ABCDEF",
				FontSize:   12,
				FontWeight: "bold",
				Align:      "left",
				ZIndex:     1,
			},
		},
	}

	img, err := renderLabelImage(tmpl, def, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}

	boxLeft := int(mmToDots(boxXmm, dpi))
	boxTop := int(mmToDots(boxYmm, dpi))
	boxRight := int(mmToDots(boxXmm+boxWmm, dpi))
	boxBottom := int(mmToDots(boxYmm+boxHmm, dpi))

	bounds := img.Bounds()
	var inkMinX, inkMaxX, inkMinY, inkMaxY int
	inkMinX, inkMinY = bounds.Max.X, bounds.Max.Y
	inkMaxX, inkMaxY = bounds.Min.X, bounds.Min.Y
	hasInk := false

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a == 0 {
				continue
			}
			// Near-black ink (print raster is black on white).
			if r < 0x8000 && g < 0x8000 && b < 0x8000 {
				hasInk = true
				if x < inkMinX {
					inkMinX = x
				}
				if x > inkMaxX {
					inkMaxX = x
				}
				if y < inkMinY {
					inkMinY = y
				}
				if y > inkMaxY {
					inkMaxY = y
				}
			}
		}
	}

	if !hasInk {
		t.Fatal("matn chizilmadi")
	}

	// Allow 1px antialias bleed past the padded content area; still must stay
	// inside the element box.
	tol := 2
	if inkMaxX > boxRight+tol {
		t.Errorf("matn o'ng chegara tashqarisiga chiqdi: inkMaxX=%d boxRight=%d", inkMaxX, boxRight)
	}
	if inkMinX < boxLeft-tol {
		t.Errorf("matn chap chegara tashqarisiga chiqdi: inkMinX=%d boxLeft=%d", inkMinX, boxLeft)
	}
	if inkMinY < boxTop-tol || inkMaxY > boxBottom+tol {
		t.Errorf("matn box balandligidan chiqdi: inkY=[%d,%d] boxY=[%d,%d]", inkMinY, inkMaxY, boxTop, boxBottom)
	}

	// Without squeeze, 12pt bold of that string is much wider than 25mm.
	// Ink width should be roughly the box width (squeezed), not a tiny shrunk font.
	inkW := inkMaxX - inkMinX
	boxW := boxRight - boxLeft
	if inkW < boxW/3 {
		t.Errorf("matn juda tor (ehtimol shrink o'rniga squeeze ishlamagan): inkW=%d boxW=%d", inkW, boxW)
	}
}

