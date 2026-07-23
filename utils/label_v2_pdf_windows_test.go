//go:build windows

package utils

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/klikz/api_v3/internal/models"
)

func TestPrintLabelV2MicrosoftPDFOutput(t *testing.T) {
	outputDir := t.TempDir()
	t.Setenv("LABEL_PDF_OUTPUT_DIR", outputDir)

	definition, err := json.Marshal(labelDefinition{
		Version: 1,
		Elements: []labelElement{
			{
				Type:       "text",
				X:          2,
				Y:          2,
				Width:      46,
				Height:     10,
				DataSource: "backend",
				Binding:    "serial",
				FontSize:   10,
			},
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	template := models.LabelTemplate{
		ID:         1,
		Name:       "PDF test",
		WidthMm:    50,
		HeightMm:   20,
		DPI:        203,
		Definition: definition,
	}
	printer := models.PrinterV2{
		ID:              1,
		PrinterName:     "Microsoft Print to PDF",
		LabelTemplateID: template.ID,
	}

	const serial = "ZZTEST-PDF-001"
	if err := (&UtilsStruct{}).PrintLabelV2(
		template,
		printer,
		1,
		BuildLabelPrintData(serial, "", models.ModelInfo{}, ""),
	); err != nil {
		t.Fatal(err)
	}

	files, err := filepath.Glob(filepath.Join(outputDir, serial+"-*.pdf"))
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 1 {
		t.Fatalf("bitta PDF kutilgan, topildi: %d", len(files))
	}
	info, err := os.Stat(files[0])
	if err != nil {
		t.Fatal(err)
	}
	if info.Size() == 0 {
		t.Fatal("PDF fayl bo'sh")
	}
}
