package utils

import (
	"image/png"
	"os"
	"runtime"
	"testing"

	"github.com/klikz/api_v3/internal/models"
)

func TestLabelRenderMultipleElements(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("windows fonts")
	}

	def := labelDefinition{
		Version: 1,
		Elements: []labelElement{
			{Type: "text", X: 2, Y: 2, Width: 40, Height: 8, StaticText: "Model nomi", FontSize: 10, ZIndex: 1},
			{Type: "text", X: 2, Y: 12, Width: 40, Height: 8, DataSource: "backend", Binding: "serial", FontSize: 10, ZIndex: 2},
			{Type: "rect", X: 2, Y: 22, Width: 50, Height: 20, StrokeWidth: 0.3, FillColor: "#FFFF00", ZIndex: 3},
			{Type: "barcode", X: 5, Y: 25, Width: 40, Height: 15, DataSource: "backend", Binding: "serial", Format: "code128", ZIndex: 4},
		},
	}

	tmpl := models.LabelTemplate{WidthMm: 100, HeightMm: 50, DPI: 203}
	img, err := renderLabelImage(tmpl, def, BuildLabelPreviewSampleData())
	if err != nil {
		t.Fatal(err)
	}

	f, err := os.Create("_test_label.png")
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, img); err != nil {
		t.Fatal(err)
	}
}
