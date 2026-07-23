package utils

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/boombuler/barcode"
	"github.com/boombuler/barcode/code128"
	"github.com/boombuler/barcode/datamatrix"
	"github.com/boombuler/barcode/ean"
	"github.com/boombuler/barcode/qr"
	"github.com/fogleman/gg"
	"github.com/klikz/api_v3/internal/models"
)

var (
	cachedFontOnce sync.Once
	cachedFontPath string
)

type labelDefinition struct {
	Version  int            `json:"version"`
	Elements []labelElement `json:"elements"`
}

type labelTableCell struct {
	DataSource string  `json:"dataSource"`
	Binding    string  `json:"binding"`
	StaticText string  `json:"staticText"`
	Prefix     string  `json:"prefix"`
	FontSize   float64 `json:"fontSize"`
	FontFamily string  `json:"fontFamily"`
	DateFormat string  `json:"dateFormat"`
	FontWeight string  `json:"fontWeight"`
	Align      string  `json:"align"`
	FillColor  string  `json:"fillColor"`
}

type labelElement struct {
	ID          string           `json:"id"`
	Type        string           `json:"type"`
	X           float64          `json:"x"`
	Y           float64          `json:"y"`
	Width       float64          `json:"width"`
	Height      float64          `json:"height"`
	DataSource  string           `json:"dataSource"`
	Binding     string           `json:"binding"`
	StaticText  string           `json:"staticText"`
	Prefix      string           `json:"prefix"`
	FontSize    float64          `json:"fontSize"`
	FontFamily  string           `json:"fontFamily"`
	DateFormat  string           `json:"dateFormat"`
	FontWeight  string           `json:"fontWeight"`
	Align       string           `json:"align"`
	Format      string           `json:"format"`
	Src         string           `json:"src"`
	StrokeWidth float64          `json:"strokeWidth"`
	FillColor   string           `json:"fillColor"`
	Rows        int              `json:"rows"`
	Cols        int              `json:"cols"`
	Cells       []labelTableCell `json:"cells"`
	ColWidths   []float64        `json:"colWidths"`
	ZIndex      int              `json:"zIndex"`
}

func (u *UtilsStruct) PrintLabelV2(
	template models.LabelTemplate,
	printer models.PrinterV2,
	copies int,
	data map[string]any,
) error {
	printerName := strings.TrimSpace(printer.PrinterName)
	if printerName == "" {
		return errors.New("printer nomi bo'sh")
	}
	if copies < 1 {
		copies = 1
	}
	if runtime.GOOS != "windows" {
		return errors.New("V2 etiketka chop etish faqat Windows serverda ishlaydi")
	}
	if data == nil {
		data = map[string]any{}
	}

	definition, err := parseLabelDefinition(template.Definition)
	if err != nil {
		return err
	}
	img, err := renderLabelImage(template, definition, data)
	if err != nil {
		return err
	}

	serial, _ := data["serial"].(string)
	if serial == "" {
		serial = "label"
	}

	printLanguage := ResolvePrinterLanguage(printerName)
	printWidthMm, printHeightMm := effectiveLabelDimensions(template.WidthMm, template.HeightMm, template.PrintRotationDeg)
	rotationDeg := normalizePrintRotationDeg(template.PrintRotationDeg)
	dpi := template.DPI
	if dpi <= 0 {
		dpi = 203
	}
	labelWidthDots := int(mmToDots(template.WidthMm, dpi))
	labelHeightDots := int(mmToDots(template.HeightMm, dpi))

	switch printLanguage {
	case PrintLanguageTSPL, PrintLanguageZPL:
		var payload []byte
		if printLanguage == PrintLanguageTSPL {
			payload, err = buildTSPLPayload(printWidthMm, printHeightMm, copies, img, rotationDeg)
		} else {
			payload, err = buildZPLPayload(copies, img, labelWidthDots, labelHeightDots, rotationDeg)
		}
		if err != nil {
			return err
		}
		if err := rawPrintWindows(printerName, serial, RawSpoolDatatype(printLanguage), payload); err != nil {
			return err
		}
	default:
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := enc.Encode(&buf, img); err != nil {
			return fmt.Errorf("PNG kodlash xatosi: %w", err)
		}
		pngBytes := buf.Bytes()
		tempDir := os.TempDir()
		tempFile, err := os.CreateTemp(tempDir, fmt.Sprintf("label-v2-%s-*.png", sanitizeLabelFilenamePart(serial)))
		if err != nil {
			return fmt.Errorf("vaqtinchalik fayl yaratilmadi: %w", err)
		}
		tempPath := tempFile.Name()
		if _, err := tempFile.Write(pngBytes); err != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
			return fmt.Errorf("vaqtinchalik fayl saqlanmadi: %w", err)
		}
		if err := tempFile.Close(); err != nil {
			_ = os.Remove(tempPath)
			return fmt.Errorf("vaqtinchalik fayl yopilmadi: %w", err)
		}
		defer os.Remove(tempPath)

		pdfOutputPath := ""
		if strings.EqualFold(printerName, "Microsoft Print to PDF") {
			if outputDir := strings.TrimSpace(os.Getenv("LABEL_PDF_OUTPUT_DIR")); outputDir != "" {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					return fmt.Errorf("PDF test papkasi yaratilmadi: %w", err)
				}
				pdfOutputPath = filepath.Join(
					outputDir,
					fmt.Sprintf("%s-%d.pdf", sanitizeLabelFilenamePart(serial), time.Now().UnixNano()),
				)
			}
		}

		if err := printImageWindows(
			printerName,
			serial,
			tempPath,
			pdfOutputPath,
			printWidthMm,
			printHeightMm,
			copies,
			rotationDeg,
		); err != nil {
			return err
		}
	}

	if dpi <= 0 {
		dpi = 203
	}

	u.DebugLogAny("V2 label printed: ", map[string]any{
		"printer":            printerName,
		"print_language":     printLanguage,
		"spool_datatype":     RawSpoolDatatype(printLanguage),
		"template":           template.ID,
		"serial":             serial,
		"copies":             copies,
		"width_mm":           template.WidthMm,
		"height_mm":          template.HeightMm,
		"print_width_mm":     printWidthMm,
		"print_height_mm":    printHeightMm,
		"print_rotation_deg": normalizePrintRotationDeg(template.PrintRotationDeg),
		"dpi":                dpi,
	})
	return nil
}

func parseLabelDefinition(raw json.RawMessage) (labelDefinition, error) {
	definition := labelDefinition{Version: 1, Elements: []labelElement{}}
	if len(raw) == 0 {
		return definition, nil
	}
	if err := json.Unmarshal(raw, &definition); err != nil {
		return definition, fmt.Errorf("shablon o'qilmadi: %w", err)
	}
	if definition.Elements == nil {
		definition.Elements = []labelElement{}
	}
	return definition, nil
}

// LabelTemplateNeedsGSCode returns true when the template has elements bound to gscode.data.
func LabelTemplateNeedsGSCode(template models.LabelTemplate) (bool, error) {
	definition, err := parseLabelDefinition(template.Definition)
	if err != nil {
		return false, err
	}
	for _, el := range definition.Elements {
		binding := strings.ToLower(strings.TrimSpace(el.Binding))
		if binding == "gscode.data" || strings.HasPrefix(binding, "gscode.") {
			return true, nil
		}
	}
	return false, nil
}

func (u *UtilsStruct) RenderLabelV2PNG(template models.LabelTemplate, data map[string]any) ([]byte, error) {
	definition, err := parseLabelDefinition(template.Definition)
	if err != nil {
		return nil, err
	}
	u.DebugLogAny("V2 label render: ", map[string]any{
		"template_id":    template.ID,
		"elements_count": len(definition.Elements),
		"width_mm":       template.WidthMm,
		"height_mm":      template.HeightMm,
	})
	img, err := renderLabelImage(template, definition, data)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	enc := png.Encoder{CompressionLevel: png.BestSpeed}
	if err := enc.Encode(&buf, img); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (u *UtilsStruct) RenderLabelV2PreviewDataURL(template models.LabelTemplate, data map[string]any) (string, int, int, error) {
	pngBytes, err := u.RenderLabelV2PNG(template, data)
	if err != nil {
		return "", 0, 0, err
	}
	if normalizePrintRotationDeg(template.PrintRotationDeg) == 90 {
		img, decodeErr := decodePNGToImage(pngBytes)
		if decodeErr != nil {
			return "", 0, 0, decodeErr
		}
		img = rotateImage90CW(img)
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := enc.Encode(&buf, img); err != nil {
			return "", 0, 0, err
		}
		pngBytes = buf.Bytes()
	}
	dpi := template.DPI
	if dpi <= 0 {
		dpi = 203
	}
	widthPx := int(mmToDots(template.WidthMm, dpi))
	heightPx := int(mmToDots(template.HeightMm, dpi))
	encoded := base64.StdEncoding.EncodeToString(pngBytes)
	return "data:image/png;base64," + encoded, widthPx, heightPx, nil
}

func modelInfoToPrintMap(model models.ModelInfo) map[string]any {
	raw, err := json.Marshal(model)
	if err != nil {
		return map[string]any{}
	}
	var out map[string]any
	if err := json.Unmarshal(raw, &out); err != nil {
		return map[string]any{}
	}
	return out
}

func sampleLabelPreviewModel() models.ModelInfo {
	return models.ModelInfo{
		Modeli:                      "PRM-211TFDF/W",
		Qisqa_nomi:                  "211",
		Sovutgich_turi:              "Defrost",
		Brend:                       "Premier",
		GS1_EAN13:                   "4780092900049",
		GOST:                        "GOST ISO 62552-2013",
		Rangi:                       "Oq",
		Brutto:                      "44",
		Netto:                       "39",
		Manzil:                      "Oxangaron",
		Korxon_nomi:                 "Premier LLC",
		Ishlab_chiqaruvchi_mamlakat: "O'zbekiston",
		Taminot_kuchlanishi_v:       "220В-240В/50Гц",
		Umumiy_hajmi_l:              "211",
		Nominal_tok_quvvati_w:       "62",
		Qadoq_hajmi:                 "578x600x1475",
		Freon:                       "R600a",
		Xladagent_miqdori_g:         "45",
		OdooCode:                    "103012111001",
	}
}

func BuildLabelPreviewSampleData() map[string]any {
	return map[string]any{
		"serial":           "ABC123456789",
		"acc_serial":       "ACC001234",
		"seriya_raqami":    "RD",
		"radiator_counter": 1,
		"klapan_counter":   1,
		"eshik_counter":    1,
		"index_1":          "IDX-1",
		"index_2":          "IDX-2",
		"radiator": map[string]any{
			"index1": "R-IDX-1",
			"index2": "R-IDX-2",
		},
		"klapan": map[string]any{
			"index1": "K-IDX-1",
			"index2": "K-IDX-2",
		},
		"eshik": map[string]any{
			"index1": "E-IDX-1",
			"index2": "E-IDX-2",
		},
		"count": 50,
		"gscode": map[string]any{
			"data": "010860123456789021ABC123456789012345678901234567890",
		},
		"model": modelInfoToPrintMap(sampleLabelPreviewModel()),
	}
}

func MergeLabelPreviewSampleData(overrides map[string]any) map[string]any {
	data := BuildLabelPreviewSampleData()
	if overrides == nil {
		return data
	}
	if v, ok := overrides["serial"].(string); ok {
		data["serial"] = strings.TrimSpace(v)
	}
	if v, ok := overrides["acc_serial"].(string); ok {
		data["acc_serial"] = strings.TrimSpace(v)
	}
	return data
}

func BuildLabelPrintData(serial, accSerial string, model models.ModelInfo, gsCode string) map[string]any {
	return map[string]any{
		"serial":     serial,
		"acc_serial": strings.TrimSpace(accSerial),
		"index_1":    "",
		"index_2":    "",
		"gscode": map[string]any{
			"data": gsCode,
		},
		"model": modelInfoToPrintMap(model),
	}
}

func resolveLabelBinding(data map[string]any, binding string) string {
	if binding == "" {
		return ""
	}
	parts := strings.Split(binding, ".")
	var current any = data
	for _, part := range parts {
		asMap, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = asMap[part]
	}
	if current == nil {
		return ""
	}
	return strings.TrimSpace(fmt.Sprint(current))
}

const labelTodayBinding = "today"
const labelGS1Data38Binding = "gscode.data38"
const labelRadiatorSerialBinding = "radiator.serial"
const labelKlapanSerialBinding = "klapan.serial"
const labelEshikSerialBinding = "eshik.serial"
const labelGS1Data38Length = 38

func sanitizeLabelFilenamePart(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return "label"
	}
	replacer := strings.NewReplacer(
		"/", "_",
		"\\", "_",
		":", "_",
		"*", "_",
		"?", "_",
		"\"", "_",
		"<", "_",
		">", "_",
		"|", "_",
	)
	result := replacer.Replace(value)
	if len(result) > 80 {
		result = result[:80]
	}
	return result
}

func truncateLabelString(value string, maxLen int) string {
	value = strings.TrimSpace(value)
	if maxLen <= 0 || len(value) <= maxLen {
		return value
	}
	return value[:maxLen]
}

func formatLabelDate(t time.Time, format string) string {
	if strings.TrimSpace(format) == "" {
		format = "DD.MM.YYYY"
	}
	layout := format
	layout = strings.ReplaceAll(layout, "YYYY", "2006")
	layout = strings.ReplaceAll(layout, "YY", "06")
	layout = strings.ReplaceAll(layout, "DD", "02")
	layout = strings.ReplaceAll(layout, "MM", "01")
	return t.Format(layout)
}

func resolveDerivedBinding(data map[string]any, binding, dateFormat string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(binding)) {
	case labelTodayBinding:
		return formatLabelDate(time.Now(), dateFormat), true
	case labelGS1Data38Binding:
		return truncateLabelString(resolveLabelBinding(data, "gscode.data"), labelGS1Data38Length), true
	case labelRadiatorSerialBinding:
		if serial := strings.TrimSpace(resolveLabelBinding(data, "serial")); serial != "" {
			return serial, true
		}
		counter := 1
		switch v := data["radiator_counter"].(type) {
		case int:
			counter = v
		case int64:
			counter = int(v)
		case float64:
			counter = int(v)
		}
		return GenerateSerial(resolveLabelBinding(data, "seriya_raqami"), counter), true
	case labelKlapanSerialBinding:
		if serial := strings.TrimSpace(resolveLabelBinding(data, "serial")); serial != "" {
			return serial, true
		}
		counter := 1
		switch v := data["klapan_counter"].(type) {
		case int:
			counter = v
		case int64:
			counter = int(v)
		case float64:
			counter = int(v)
		}
		return GenerateSerial(resolveLabelBinding(data, "seriya_raqami"), counter), true
	case labelEshikSerialBinding:
		if serial := strings.TrimSpace(resolveLabelBinding(data, "serial")); serial != "" {
			return serial, true
		}
		counter := 1
		switch v := data["eshik_counter"].(type) {
		case int:
			counter = v
		case int64:
			counter = int(v)
		case float64:
			counter = int(v)
		}
		return GenerateSerial(resolveLabelBinding(data, "seriya_raqami"), counter), true
	default:
		return "", false
	}
}

func inferLabelDataSource(el labelElement) string {
	if ds := strings.ToLower(strings.TrimSpace(el.DataSource)); ds != "" {
		return ds
	}
	if el.Type == "barcode" || el.Type == "datamatrix" || el.Type == "qrcode" {
		return "backend"
	}
	if el.Binding != "" {
		return "backend"
	}
	return "static"
}

func resolveLabelElementText(data map[string]any, el labelElement) string {
	dataSource := inferLabelDataSource(el)
	if dataSource == "backend" && el.Binding != "" {
		if value, ok := resolveDerivedBinding(data, el.Binding, el.DateFormat); ok {
			return el.Prefix + value
		}
		value := resolveLabelBinding(data, el.Binding)
		if value != "" {
			return el.Prefix + value
		}
		return el.Prefix
	}
	if el.StaticText != "" {
		return el.StaticText
	}
	return ""
}

func resolveLabelImageSource(data map[string]any, el labelElement) string {
	dataSource := inferLabelDataSource(el)
	if dataSource == "backend" && strings.TrimSpace(el.Binding) != "" {
		value := resolveLabelBinding(data, el.Binding)
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return strings.TrimSpace(el.Src)
}

func resolveTableCellText(data map[string]any, cell labelTableCell, defaultDateFormat string) string {
	dataSource := cell.DataSource
	if dataSource == "" {
		if cell.Binding != "" {
			dataSource = "backend"
		} else {
			dataSource = "static"
		}
	}
	if dataSource == "backend" && cell.Binding != "" {
		dateFormat := cell.DateFormat
		if strings.TrimSpace(dateFormat) == "" {
			dateFormat = defaultDateFormat
		}
		if value, ok := resolveDerivedBinding(data, cell.Binding, dateFormat); ok {
			return cell.Prefix + value
		}
		value := resolveLabelBinding(data, cell.Binding)
		if value != "" {
			return cell.Prefix + value
		}
		return cell.Prefix
	}
	return cell.StaticText
}

func mmToDots(mm float64, dpi int) float64 {
	return mm / 25.4 * float64(dpi)
}

func sortElementsForRender(elements []labelElement) []labelElement {
	backgrounds := make([]labelElement, 0, len(elements))
	foreground := make([]labelElement, 0, len(elements))
	for _, el := range elements {
		switch el.Type {
		case "text", "barcode", "datamatrix", "qrcode":
			foreground = append(foreground, el)
		default:
			backgrounds = append(backgrounds, el)
		}
	}
	sort.SliceStable(backgrounds, func(i, j int) bool {
		return backgrounds[i].ZIndex < backgrounds[j].ZIndex
	})
	sort.SliceStable(foreground, func(i, j int) bool {
		return foreground[i].ZIndex < foreground[j].ZIndex
	})
	return append(backgrounds, foreground...)
}

func renderLabelImage(template models.LabelTemplate, definition labelDefinition, data map[string]any) (image.Image, error) {
	dpi := template.DPI
	if dpi <= 0 {
		dpi = 203
	}

	width := int(mmToDots(template.WidthMm, dpi))
	height := int(mmToDots(template.HeightMm, dpi))
	if width <= 0 || height <= 0 {
		return nil, errors.New("noto'g'ri etiketka o'lchami")
	}

	dc := gg.NewContext(width, height)
	dc.SetColor(color.White)
	dc.Clear()

	elements := sortElementsForRender(append([]labelElement(nil), definition.Elements...))

	for _, el := range elements {
		if err := drawLabelElement(dc, el, data, dpi); err != nil {
			return nil, err
		}
	}

	return dc.Image(), nil
}

func firstExistingFont(paths ...string) string {
	for _, path := range paths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}
	return ""
}

func resolveBundledFont(relPath string) string {
	candidates := []string{filepath.Join("fonts", relPath)}
	if wd, err := os.Getwd(); err == nil {
		candidates = append(candidates, filepath.Join(wd, "fonts", relPath))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), "fonts", relPath))
	}
	return firstExistingFont(candidates...)
}

func isConsolaFontPath(fontPath string) bool {
	return strings.Contains(strings.ToLower(filepath.Base(fontPath)), "consola")
}

func resolveLabelFontFamily(fontFamily string) string {
	switch strings.ToLower(strings.TrimSpace(fontFamily)) {
	case "segoe":
		return firstExistingFont(`C:\Windows\Fonts\segoeui.ttf`)
	case "tahoma":
		return firstExistingFont(`C:\Windows\Fonts\tahoma.ttf`)
	case "calibri":
		return firstExistingFont(`C:\Windows\Fonts\calibri.ttf`)
	case "consola":
		return firstExistingFont(
			resolveBundledFont("consola/CONSOLA.TTF"),
			resolveBundledFont("consola/consola.ttf"),
			`C:\Windows\Fonts\consola.ttf`,
		)
	case "arial":
		return firstExistingFont(`C:\Windows\Fonts\arial.ttf`)
	default:
		return firstExistingFont(`C:\Windows\Fonts\arial.ttf`)
	}
}

func labelElementFontPath(fontFamily string) string {
	if path := resolveLabelFontFamily(fontFamily); path != "" {
		return path
	}
	return findWindowsFont()
}

func findWindowsFont() string {
	cachedFontOnce.Do(func() {
		candidates := []string{
			`C:\Windows\Fonts\arial.ttf`,
			`C:\Windows\Fonts\segoeui.ttf`,
			`C:\Windows\Fonts\tahoma.ttf`,
		}
		for _, path := range candidates {
			if _, err := os.Stat(path); err == nil {
				cachedFontPath = path
				return
			}
		}
	})
	return cachedFontPath
}

func resolveLabelFontPath(fontPath, fontWeight string) string {
	if isConsolaFontPath(fontPath) {
		dir := filepath.Dir(fontPath)
		if fontWeight == "bold" {
			if bold := firstExistingFont(
				filepath.Join(dir, "CONSOLAB.TTF"),
				filepath.Join(dir, "consolab.ttf"),
				resolveBundledFont("consola/CONSOLAB.TTF"),
			); bold != "" {
				return bold
			}
		}
		return fontPath
	}
	if fontWeight == "bold" {
		dir := filepath.Dir(fontPath)
		base := strings.TrimSuffix(filepath.Base(fontPath), filepath.Ext(fontPath))
		for _, boldName := range []string{base + "bd.ttf", base + "b.ttf", "arialbd.ttf"} {
			candidate := filepath.Join(dir, boldName)
			if _, err := os.Stat(candidate); err == nil {
				return candidate
			}
		}
	}
	return fontPath
}

func loadLabelFontFace(dc *gg.Context, fontPath, fontWeight string, pixelSize float64) error {
	return dc.LoadFontFace(resolveLabelFontPath(fontPath, fontWeight), pixelSize)
}

func textLinesFit(dc *gg.Context, lines []string, maxW, maxH, lineSpacing float64) bool {
	joined := strings.Join(lines, "\n")
	textW, textH := dc.MeasureMultilineString(joined, lineSpacing)
	if textH > maxH || textW > maxW {
		return false
	}
	for _, line := range lines {
		w, _ := dc.MeasureString(line)
		if w > maxW {
			return false
		}
	}
	return true
}

func drawTextInBox(dc *gg.Context, text string, x, y, w, h float64, dpi int, fontPath string, fontSize float64, fontWeight, align string) error {
	if strings.TrimSpace(text) == "" {
		return nil
	}
	if fontSize <= 0 {
		fontSize = 10
	}

	pad := mmToDots(0.3, dpi)
	if pad*2 >= h {
		pad = h * 0.05
	}
	if pad*2 >= w {
		pad = w * 0.05
	}
	maxW := w - 2*pad
	maxH := h - 2*pad
	if maxW <= 0 {
		maxW = w
		pad = 0
	}
	if maxH <= 0 {
		maxH = h
		pad = 0
	}

	lines := strings.Split(text, "\n")
	const lineSpacing = 1.2

	for size := fontSize; size >= 4; size -= 0.5 {
		pixelSize := size * float64(dpi) / 72.0
		if err := loadLabelFontFace(dc, fontPath, fontWeight, pixelSize); err != nil {
			return err
		}
		if textLinesFit(dc, lines, maxW, maxH, lineSpacing) {
			fontSize = size
			break
		}
	}

	if err := loadLabelFontFace(dc, fontPath, fontWeight, fontSize*float64(dpi)/72.0); err != nil {
		return err
	}

	_, totalH := dc.MeasureMultilineString(text, lineSpacing)
	fontH := dc.FontHeight()
	blockTop := y + (h-totalH)/2

	dc.SetColor(color.Black)

	var ax, xPos float64
	switch align {
	case "center":
		ax = 0.5
		xPos = x + w/2
	case "right":
		ax = 1
		xPos = x + w - pad
	default:
		ax = 0
		xPos = x + pad
	}

	for i, line := range lines {
		lineY := blockTop + fontH/2 + float64(i)*fontH*lineSpacing
		dc.DrawStringAnchored(line, xPos, lineY, ax, 0.5)
	}
	return nil
}

func drawLabelElement(dc *gg.Context, el labelElement, data map[string]any, dpi int) error {
	fontPath := labelElementFontPath(el.FontFamily)
	if fontPath == "" {
		return errors.New("sistema shrifti topilmadi")
	}
	x := mmToDots(el.X, dpi)
	y := mmToDots(el.Y, dpi)
	w := mmToDots(el.Width, dpi)
	h := mmToDots(el.Height, dpi)
	if w <= 0 {
		w = mmToDots(10, dpi)
	}
	if h <= 0 {
		h = mmToDots(5, dpi)
	}

	switch el.Type {
	case "text":
		text := resolveLabelElementText(data, el)
		if text == "" {
			return nil
		}
		fontSize := el.FontSize
		if fontSize <= 0 {
			fontSize = 10
		}
		return drawTextInBox(dc, text, x, y, w, h, dpi, fontPath, fontSize, el.FontWeight, el.Align)
	case "line":
		stroke := mmToDots(el.StrokeWidth, dpi)
		if stroke <= 0 {
			stroke = mmToDots(0.3, dpi)
		}
		dc.SetColor(color.Black)
		dc.DrawRectangle(x, y+h/2-stroke/2, w, stroke)
		dc.Fill()
	case "rect":
		stroke := mmToDots(el.StrokeWidth, dpi)
		if stroke <= 0 {
			stroke = mmToDots(0.3, dpi)
		}
		if fill, ok := parseHexColor(el.FillColor); ok {
			dc.SetColor(fill)
			dc.DrawRectangle(x+stroke/2, y+stroke/2, w-stroke, h-stroke)
			dc.Fill()
		}
		dc.SetColor(color.Black)
		dc.SetLineWidth(stroke)
		dc.DrawRectangle(x+stroke/2, y+stroke/2, w-stroke, h-stroke)
		dc.Stroke()
	case "table":
		return drawTable(dc, el, data, x, y, w, h, dpi, fontPath)
	case "barcode":
		value := resolveLabelElementText(data, el)
		if value == "" {
			return nil
		}
		bc, err := encodeBarcode(value, el.Format, int(w), int(h))
		if err != nil {
			return nil
		}
		dc.DrawImage(bc, int(x), int(y))
	case "datamatrix":
		value := resolveLabelElementText(data, el)
		if value == "" {
			return nil
		}
		bc, err := datamatrix.Encode(value)
		if err != nil {
			return nil
		}
		scaled, err := barcode.Scale(bc, int(w), int(h))
		if err != nil {
			return nil
		}
		dc.DrawImage(scaled, int(x), int(y))
	case "qrcode":
		value := resolveLabelElementText(data, el)
		if value == "" {
			return nil
		}
		bc, err := qr.Encode(value, qr.M, qr.Auto)
		if err != nil {
			return nil
		}
		scaled, err := barcode.Scale(bc, int(w), int(h))
		if err != nil {
			return nil
		}
		dc.DrawImage(scaled, int(x), int(y))
	case "image":
		imageSrc := resolveLabelImageSource(data, el)
		if imageSrc == "" {
			return nil
		}
		img, loadedFrom, err := loadLabelImage(imageSrc)
		if err != nil {
			return nil
		}
		if img.Bounds().Dx() <= 0 || img.Bounds().Dy() <= 0 {
			return nil
		}
		_ = loadedFrom
		dc.Push()
		dc.Translate(x, y)
		sx := w / float64(img.Bounds().Dx())
		sy := h / float64(img.Bounds().Dy())
		dc.Scale(sx, sy)
		dc.DrawImage(img, 0, 0)
		dc.Pop()
	}
	return nil
}

func drawTable(dc *gg.Context, el labelElement, data map[string]any, x, y, w, h float64, dpi int, fontPath string) error {
	rows := el.Rows
	if rows < 1 {
		rows = 2
	}
	cols := el.Cols
	if cols < 1 {
		cols = 3
	}

	stroke := mmToDots(el.StrokeWidth, dpi)
	if stroke <= 0 {
		stroke = mmToDots(0.3, dpi)
	}
	stroke = math.Max(1, math.Round(stroke))

	colWidths := tableColumnWidthsDots(el.ColWidths, cols, w)
	rowHeights := distributePixelHeights(int(math.Round(h)), rows)
	defaultFont := el.FontSize
	if defaultFont <= 0 {
		defaultFont = 8
	}

	cx := x
	for col := 0; col < cols; col++ {
		cellW := colWidths[col]
		cy := y
		for row := 0; row < rows; row++ {
			cellH := float64(rowHeights[row])
			idx := row*cols + col
			var cell labelTableCell
			if idx < len(el.Cells) {
				cell = el.Cells[idx]
			}

			if fill, ok := parseHexColor(cell.FillColor); ok {
				dc.SetColor(fill)
				fillRectPx(dc, cx+stroke/2, cy+stroke/2, cellW-stroke, cellH-stroke)
			}

			text := resolveTableCellText(data, cell, el.DateFormat)
			if strings.TrimSpace(text) != "" {
				fontSize := cell.FontSize
				if fontSize <= 0 {
					fontSize = defaultFont
				}
				cellFontFamily := cell.FontFamily
				if strings.TrimSpace(cellFontFamily) == "" {
					cellFontFamily = el.FontFamily
				}
				cellFontPath := labelElementFontPath(cellFontFamily)
				if cellFontPath == "" {
					cellFontPath = fontPath
				}
				align := cell.Align
				if align == "" {
					align = "center"
				}
				if err := drawTextInBox(
					dc, text,
					cx+stroke/2, cy+stroke/2, cellW-stroke, cellH-stroke,
					dpi, cellFontPath, fontSize, cell.FontWeight, align,
				); err != nil {
					return err
				}
			}
			cy += cellH
		}
		cx += cellW
	}

	dc.SetColor(color.Black)
	lineX := math.Round(x)
	topY := math.Round(y)
	bottomY := topY + float64(sumInts(rowHeights))
	for col := 0; col <= cols; col++ {
		fillRectPx(dc, lineX-stroke/2, topY, stroke, bottomY-topY)
		if col < cols {
			lineX += colWidths[col]
		}
	}
	lineY := topY
	for row := 0; row <= rows; row++ {
		fillRectPx(dc, math.Round(x), lineY-stroke/2, w, stroke)
		if row < rows {
			lineY += float64(rowHeights[row])
		}
	}

	return nil
}

func distributePixelHeights(totalPx, rows int) []int {
	if rows < 1 {
		return nil
	}
	if totalPx < rows {
		totalPx = rows
	}
	base := totalPx / rows
	rem := totalPx % rows
	out := make([]int, rows)
	for i := range out {
		out[i] = base
		if i < rem {
			out[i]++
		}
	}
	return out
}

func sumInts(values []int) int {
	sum := 0
	for _, v := range values {
		sum += v
	}
	return sum
}

func fillRectPx(dc *gg.Context, x, y, w, h float64) {
	if w <= 0 || h <= 0 {
		return
	}
	dc.DrawRectangle(math.Round(x), math.Round(y), math.Max(1, math.Round(w)), math.Max(1, math.Round(h)))
	dc.Fill()
}

func tableColumnWidthsDots(colWidthsMM []float64, cols int, totalWDots float64) []float64 {
	if cols < 1 {
		return nil
	}
	if len(colWidthsMM) != cols {
		equal := totalWDots / float64(cols)
		out := make([]float64, cols)
		for i := range out {
			out[i] = equal
		}
		return out
	}
	sumMm := 0.0
	for _, cw := range colWidthsMM {
		sumMm += cw
	}
	if sumMm <= 0 {
		equal := totalWDots / float64(cols)
		out := make([]float64, cols)
		for i := range out {
			out[i] = equal
		}
		return out
	}
	out := make([]float64, cols)
	acc := 0.0
	for i := 0; i < cols-1; i++ {
		out[i] = totalWDots * (colWidthsMM[i] / sumMm)
		acc += out[i]
	}
	out[cols-1] = totalWDots - acc
	return out
}

func encodeBarcode(value, format string, width, height int) (image.Image, error) {
	format = strings.ToLower(strings.TrimSpace(format))
	tryFormats := []string{format}
	if format != "code128" {
		tryFormats = append(tryFormats, "code128")
	}
	if format != "ean13" && format != "ean" {
		tryFormats = append(tryFormats, "ean13")
	}

	var lastErr error
	for _, f := range tryFormats {
		var bc barcode.Barcode
		var err error
		switch f {
		case "code128":
			bc, err = code128.Encode(value)
		default:
			bc, err = ean.Encode(value)
		}
		if err != nil {
			lastErr = err
			continue
		}
		scaled, err := barcode.Scale(bc, width, height)
		if err != nil {
			lastErr = err
			continue
		}
		return scaled, nil
	}
	if lastErr != nil {
		return nil, lastErr
	}
	return nil, errors.New("shtrix kod yaratib bo'lmadi")
}

func parseHexColor(raw string) (color.Color, bool) {
	s := strings.TrimSpace(strings.TrimPrefix(raw, "#"))
	if len(s) != 6 {
		return nil, false
	}
	var r, g, b uint8
	if _, err := fmt.Sscanf(s, "%02x%02x%02x", &r, &g, &b); err != nil {
		return nil, false
	}
	return color.RGBA{R: r, G: g, B: b, A: 255}, true
}

func resolveLabelImagePath(src string) []string {
	clean := filepath.Clean(strings.TrimPrefix(src, "/"))
	candidates := []string{clean}
	if !strings.HasPrefix(clean, "uploads") {
		candidates = append(candidates, filepath.Join("uploads", "labels", filepath.Base(clean)))
	}
	roots := []string{"."}
	if wd, err := os.Getwd(); err == nil {
		roots = append(roots, wd)
	}
	if exe, err := os.Executable(); err == nil {
		roots = append(roots, filepath.Dir(exe))
	}
	for _, root := range roots {
		candidates = append(candidates, filepath.Join(root, clean))
		if !strings.HasPrefix(clean, "uploads") {
			candidates = append(candidates, filepath.Join(root, "uploads", "labels", filepath.Base(clean)))
		}
	}
	seen := map[string]bool{}
	unique := []string{}
	for _, c := range candidates {
		if c == "" || seen[c] {
			continue
		}
		seen[c] = true
		unique = append(unique, c)
	}
	return unique
}

func loadLabelImage(src string) (image.Image, string, error) {
	for _, path := range resolveLabelImagePath(src) {
		if strings.Contains(path, "..") {
			continue
		}
		if _, err := os.Stat(path); err != nil {
			continue
		}
		img, err := gg.LoadImage(path)
		if err == nil {
			return img, path, nil
		}
	}
	return nil, "", fmt.Errorf("rasm topilmadi: %s", src)
}
