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
	"golang.org/x/image/font"
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
	Orientation string           `json:"orientation"`
	Rows        int              `json:"rows"`
	Cols        int              `json:"cols"`
	Cells       []labelTableCell `json:"cells"`
	ColWidths   []float64        `json:"colWidths"`
	ZIndex      int              `json:"zIndex"`
	RotationDeg int              `json:"rotationDeg"`
}

func (u *UtilsStruct) PrintLabelV2(
	template models.LabelTemplate,
	printer models.PrinterV2,
	copies int,
	data map[string]any,
) error {
	start := time.Now()
	printV2InFlight.Add(1)
	defer printV2InFlight.Add(-1)

	printLanguage := NormalizePrintLanguage(printer.PrintLanguage)
	printerName := strings.TrimSpace(printer.PrinterName)
	effectiveLanguage, _ := resolveEffectivePrintRoute(printLanguage, printerName, template.UsePrinterDefaults, template.SizeOnly)
	serial, _ := data["serial"].(string)
	if serial == "" {
		serial = "label"
	}
	attemptsUsed := 0
	var renderMs, submitMs int64
	var spoolJobID uint32

	err := func() error {
		// Render + payload once, under the render semaphore only; RetryPrintV2
		// re-submits to spool/verify and takes a worker slot per attempt.
		var job *preparedLabelV2Job
		var prepErr error
		renderStart := time.Now()
		_ = WithRenderSlotV2(func() error {
			job, prepErr = u.prepareLabelV2Job(template, printer, copies, data)
			return nil
		})
		renderMs = time.Since(renderStart).Milliseconds()
		if prepErr != nil {
			attemptsUsed = 1
			return prepErr
		}
		defer job.cleanup()
		if job.done {
			attemptsUsed = 1
			return nil
		}
		return RetryPrintV2(printRetryAttempts, func(attempt int) error {
			attemptsUsed = attempt
			if !IsLabelFileSink(printerName) {
				if n, clearErr := cancelStuckPrinterJobs(printerName); n > 0 {
					u.PrintLogWarn("print spool: stuck jobs cleared before attempt", map[string]any{
						"printer":        printerName,
						"printer_v2_id":  printer.ID,
						"line_id":        printer.LineID,
						"line_name":      strings.TrimSpace(printer.LineName),
						"product_serial": serial,
						"cancelled":      n,
						"attempt":        attempt,
						"queue_snapshot": u.printQueueSnapshot(printerName),
					})
					if clearErr != nil {
						u.PrintLogWarn("print spool: stuck clear partial error", map[string]any{
							"printer":        printerName,
							"product_serial": serial,
							"error":          clearErr.Error(),
						})
					}
				}
			}
			submitStart := time.Now()
			attemptErr := job.submitAndVerify()
			submitMs += time.Since(submitStart).Milliseconds()
			spoolJobID = job.spoolJobID
			if attemptErr != nil {
				u.PrintLogWarn("print attempt failed", map[string]any{
					"printer":            printerName,
					"printer_v2_id":      printer.ID,
					"line_id":            printer.LineID,
					"line_name":          strings.TrimSpace(printer.LineName),
					"product_serial":     serial,
					"template_id":        template.ID,
					"print_language":     printLanguage,
					"effective_language": effectiveLanguage,
					"attempt":            attempt,
					"attempts_max":       printRetryAttempts,
					"stage":              printErrorStage(attemptErr.Error()),
					"error":              attemptErr.Error(),
					"queue_snapshot":     u.printQueueSnapshot(printerName),
					"will_retry":         attempt < printRetryAttempts && isRetryablePrintError(attemptErr),
				})
			}
			return attemptErr
		})
	}()

	elapsed := time.Since(start).Milliseconds()
	timings := printV2Timings{RenderMs: renderMs, SubmitMs: submitMs, SpoolJob: spoolJobID}
	stage := "print"
	if err != nil {
		msg := err.Error()
		stage = printErrorStage(msg)
		recordPrintV2Fail(elapsed)
		queueSnap := u.printQueueSnapshot(printerName)
		u.PrintLogError("print failed", map[string]any{
			"printer_v2_id":      printer.ID,
			"printer":            printerName,
			"line_id":            printer.LineID,
			"line_name":          strings.TrimSpace(printer.LineName),
			"product_serial":     serial,
			"print_language":     printLanguage,
			"effective_language": effectiveLanguage,
			"template_id":        template.ID,
			"duration_ms":        elapsed,
			"render_ms":          renderMs,
			"submit_ms":          submitMs,
			"attempts":           attemptsUsed,
			"error":              msg,
			"stage":              stage,
			"queue_snapshot":     queueSnap,
			"language_hint":      ProductionPrintLanguageHint(effectiveLanguage),
			"message":            fmt.Sprintf("serial %s: %s", serial, msg),
		})
		u.persistPrintV2Event(models.PrintV2Event{
			OK:                false,
			DurationMs:        int(elapsed),
			LineID:            printer.LineID,
			LineName:          strings.TrimSpace(printer.LineName),
			PrinterV2ID:       printer.ID,
			PrinterName:       printerName,
			TemplateID:        template.ID,
			PrintLanguage:     printLanguage,
			EffectiveLanguage: effectiveLanguage,
			Serial:            serial,
			Stage:             stage,
			ErrorMessage:      truncatePrintErr(fmt.Sprintf("serial %s: %s", serial, msg), 500),
			ErrorDetail:       fmt.Sprintf("serial %s: %s", serial, msg),
			Meta:              printV2EventMetaTimed(template, copies, attemptsUsed, queueSnap, timings),
		})
		return fmt.Errorf("serial %s: %w", serial, err)
	}
	recordPrintV2Success(elapsed)
	if attemptsUsed > 1 {
		u.PrintLogWarn("print recovered after retry", map[string]any{
			"printer":        printerName,
			"printer_v2_id":  printer.ID,
			"product_serial": serial,
			"attempts":       attemptsUsed,
			"duration_ms":    elapsed,
		})
	}
	u.persistPrintV2Event(models.PrintV2Event{
		OK:                true,
		DurationMs:        int(elapsed),
		LineID:            printer.LineID,
		LineName:          strings.TrimSpace(printer.LineName),
		PrinterV2ID:       printer.ID,
		PrinterName:       printerName,
		TemplateID:        template.ID,
		PrintLanguage:     printLanguage,
		EffectiveLanguage: effectiveLanguage,
		Serial:            serial,
		Stage:             "ok",
		Meta:              printV2EventMetaTimed(template, copies, attemptsUsed, nil, timings),
	})
	return nil
}

func printErrorStage(msg string) string {
	switch {
	case strings.Contains(msg, "render:"):
		return "render"
	case strings.Contains(msg, "payload:"):
		return "payload"
	case strings.Contains(msg, "spool:"):
		return "spool"
	case strings.Contains(msg, "gdi:"):
		return "gdi"
	case strings.Contains(msg, "printer tayyor emas"):
		return "health"
	default:
		return "print"
	}
}

func (u *UtilsStruct) printQueueSnapshot(printerName string) []map[string]any {
	printerName = strings.TrimSpace(printerName)
	if printerName == "" {
		return nil
	}
	jobs, err := LocalPrinterJobs(printerName)
	if err != nil {
		return []map[string]any{{"query_error": err.Error()}}
	}
	if len(jobs) == 0 {
		return nil
	}
	out := make([]map[string]any, 0, len(jobs))
	for _, job := range jobs {
		out = append(out, map[string]any{
			"id":             job.ID,
			"document_name":  job.DocumentName,
			"job_status":     job.JobStatus,
			"user_name":      job.UserName,
			"submitted_time": job.SubmittedTime,
			"total_pages":    job.TotalPages,
			"pages_printed":  job.PagesPrinted,
			"size":           job.Size,
		})
	}
	return out
}

func truncatePrintErr(s string, max int) string {
	if max <= 0 || len(s) <= max {
		return s
	}
	return s[:max] + "…"
}

func printV2EventMeta(template models.LabelTemplate, copies int) json.RawMessage {
	return printV2EventMetaExtra(template, copies, 0, nil)
}

// printV2Timings carries the per-stage breakdown so slow prints can be
// attributed to rendering versus the spooler without guesswork.
type printV2Timings struct {
	RenderMs int64
	SubmitMs int64
	SpoolJob uint32
}

func printV2EventMetaExtra(template models.LabelTemplate, copies, attempts int, queueSnap []map[string]any) json.RawMessage {
	return printV2EventMetaTimed(template, copies, attempts, queueSnap, printV2Timings{})
}

func printV2EventMetaTimed(
	template models.LabelTemplate,
	copies, attempts int,
	queueSnap []map[string]any,
	timings printV2Timings,
) json.RawMessage {
	payload := map[string]any{
		"copies":               copies,
		"dpi":                  template.DPI,
		"density":              template.Density,
		"speed":                template.Speed,
		"gap_mm":               template.GapMm,
		"use_printer_defaults": template.UsePrinterDefaults,
		"size_only":            template.SizeOnly,
		"width_mm":             template.WidthMm,
		"height_mm":            template.HeightMm,
	}
	if attempts > 0 {
		payload["attempts"] = attempts
	}
	if len(queueSnap) > 0 {
		payload["queue_snapshot"] = queueSnap
	}
	if timings.RenderMs > 0 {
		payload["render_ms"] = timings.RenderMs
	}
	if timings.SubmitMs > 0 {
		payload["submit_ms"] = timings.SubmitMs
	}
	if timings.SpoolJob > 0 {
		payload["spool_job_id"] = timings.SpoolJob
	}
	b, _ := json.Marshal(payload)
	return b
}

func (u *UtilsStruct) persistPrintV2Event(ev models.PrintV2Event) {
	go func() {
		defer func() { _ = recover() }()
		if err := u.Store.Repo().PrintV2EventInsert(ev); err != nil {
			u.PrintLogWarn("print_v2_event insert failed", map[string]any{
				"error":          err.Error(),
				"printer":        ev.PrinterName,
				"product_serial": ev.Serial,
				"ok":             ev.OK,
			})
		}
	}()
}

func (u *UtilsStruct) prepareLabelV2Job(
	template models.LabelTemplate,
	printer models.PrinterV2,
	copies int,
	data map[string]any,
) (*preparedLabelV2Job, error) {
	printerName := strings.TrimSpace(printer.PrinterName)
	if printerName == "" {
		return nil, errors.New("printer nomi bo'sh")
	}
	if copies < 1 {
		copies = 1
	}
	fileSink := IsLabelFileSink(printerName)
	if runtime.GOOS != "windows" && !fileSink {
		return nil, errors.New("V2 etiketka chop etish faqat Windows serverda ishlaydi")
	}
	if data == nil {
		data = map[string]any{}
	}

	if !fileSink {
		if err := PrinterReachable(printerName); err != nil {
			return nil, fmt.Errorf("printer tayyor emas: %w", err)
		}
	}

	definition, err := parseLabelDefinition(template.Definition)
	if err != nil {
		return nil, err
	}
	printLanguage := NormalizePrintLanguage(printer.PrintLanguage)
	effectiveLanguage, rawSizeOnly := resolveEffectivePrintRoute(printLanguage, printerName, template.UsePrinterDefaults, template.SizeOnly)
	rawDPI := rawPrinterDPIHint(template.DPI, effectiveLanguage, printerName)
	renderTemplate := template
	renderTemplate.DPI = rawDPI
	img, err := renderLabelImage(renderTemplate, definition, data)
	if err != nil {
		return nil, fmt.Errorf("render: %w", err)
	}

	serial, _ := data["serial"].(string)
	if serial == "" {
		serial = "label"
	}

	if effectiveLanguage == PrintLanguageGDI {
		detected, driverName, reason := DetectPrinterLanguage(printerName)
		if detected == PrintLanguageZPL || detected == PrintLanguageTSPL {
			u.PrintLogWarn("print GDI on thermal printer", map[string]any{
				"printer":              printerName,
				"printer_v2_id":        printer.ID,
				"product_serial":       serial,
				"detected_language":    detected,
				"effective_language":   effectiveLanguage,
				"driver_name":          driverName,
				"use_printer_defaults": template.UsePrinterDefaults,
				"reason":               reason,
				"hint":                 "Thermal uchun ZPL/TSPL tavsiya; Use defaults GDI jobni spool'da qoldirishi mumkin",
			})
		}
	}
	rotationDeg := normalizePrintRotationDeg(template.PrintRotationDeg)
	designWidthMm, designHeightMm := template.WidthMm, template.HeightMm
	printWidthMm, printHeightMm := effectiveLabelDimensions(designWidthMm, designHeightMm, rotationDeg)
	dpi := rawDPI
	if dpi != 203 && dpi != 300 {
		u.PrintLogWarn("print unusual DPI", map[string]any{
			"printer":     printerName,
			"template_id": template.ID,
			"dpi":         dpi,
			"hint":        "thermal printerlar odatda 203 yoki 300 DPI",
		})
	}
	if dpi != template.DPI && template.DPI > 0 {
		u.PrintLogWarn("print raw DPI overridden", map[string]any{
			"printer":        printerName,
			"printer_v2_id":  printer.ID,
			"template_id":    template.ID,
			"product_serial": serial,
			"template_dpi":   template.DPI,
			"raw_dpi":        dpi,
			"print_language": effectiveLanguage,
			"hint":           "RAW thermal bitmap printer head DPI ga moslashtirildi",
		})
	}
	labelWidthDots := int(mmToDots(designWidthMm, dpi))
	labelHeightDots := int(mmToDots(designHeightMm, dpi))
	rawSettings := RawSettingsFromTemplate(template.Density, template.Speed, template.GapMm, rawSizeOnly)

	job := &preparedLabelV2Job{
		u:                  u,
		printer:            printer,
		template:           template,
		printerName:        printerName,
		serial:             serial,
		copies:             copies,
		printLanguage:      printLanguage,
		effectiveLanguage:  effectiveLanguage,
		fileSink:           fileSink,
		rawSizeOnly:        rawSizeOnly,
		rawSettingsApplied: rawSettings.Apply && effectiveLanguage != PrintLanguageGDI,
		printWidthMm:       printWidthMm,
		printHeightMm:      printHeightMm,
		rotationDeg:        rotationDeg,
		dpi:                dpi,
	}

	switch effectiveLanguage {
	case PrintLanguageTSPL:
		payload, err := buildTSPLPayload(designWidthMm, designHeightMm, copies, img, rotationDeg, rawSettings)
		if err != nil {
			return nil, fmt.Errorf("payload: %w", err)
		}
		if fileSink {
			printImg, _ := preparePrintImage(img, rotationDeg)
			if _, err := saveLabelFileSink(printerName, serial, PrintLanguageTSPL, printImg, 0, payload); err != nil {
				return nil, fmt.Errorf("file: %w", err)
			}
			job.done = true
			job.logSuccess()
			return job, nil
		}
		job.payload = payload
	case PrintLanguageZPL:
		payload, err := buildZPLPayload(copies, img, labelWidthDots, labelHeightDots, rotationDeg, rawSettings)
		if err != nil {
			return nil, fmt.Errorf("payload: %w", err)
		}
		if fileSink {
			printImg, _ := preparePrintImage(img, rotationDeg)
			if _, err := saveLabelFileSink(printerName, serial, PrintLanguageZPL, printImg, 0, payload); err != nil {
				return nil, fmt.Errorf("file: %w", err)
			}
			job.done = true
			job.logSuccess()
			return job, nil
		}
		job.payload = payload
	default:
		printImg, gdiRot := preparePrintImage(img, rotationDeg)
		job.rotationDeg = gdiRot
		job.printWidthMm = printWidthMm
		job.printHeightMm = printHeightMm
		if fileSink {
			if _, err := saveLabelFileSink(printerName, serial, PrintLanguageGDI, printImg, gdiRot, nil); err != nil {
				return nil, fmt.Errorf("file: %w", err)
			}
			job.done = true
			job.logSuccess()
			return job, nil
		}
		var buf bytes.Buffer
		enc := png.Encoder{CompressionLevel: png.BestSpeed}
		if err := enc.Encode(&buf, printImg); err != nil {
			return nil, fmt.Errorf("PNG kodlash xatosi: %w", err)
		}
		tempFile, err := os.CreateTemp(os.TempDir(), fmt.Sprintf("label-v2-%s-*.png", sanitizeLabelFilenamePart(serial)))
		if err != nil {
			return nil, fmt.Errorf("vaqtinchalik fayl yaratilmadi: %w", err)
		}
		tempPath := tempFile.Name()
		if _, err := tempFile.Write(buf.Bytes()); err != nil {
			_ = tempFile.Close()
			_ = os.Remove(tempPath)
			return nil, fmt.Errorf("vaqtinchalik fayl saqlanmadi: %w", err)
		}
		if err := tempFile.Close(); err != nil {
			_ = os.Remove(tempPath)
			return nil, fmt.Errorf("vaqtinchalik fayl yopilmadi: %w", err)
		}
		job.tempPath = tempPath
		if strings.EqualFold(printerName, "Microsoft Print to PDF") {
			if outputDir := strings.TrimSpace(os.Getenv("LABEL_PDF_OUTPUT_DIR")); outputDir != "" {
				if err := os.MkdirAll(outputDir, 0755); err != nil {
					_ = os.Remove(tempPath)
					return nil, fmt.Errorf("PDF test papkasi yaratilmadi: %w", err)
				}
				job.pdfOutputPath = filepath.Join(
					outputDir,
					fmt.Sprintf("%s-%d.pdf", sanitizeLabelFilenamePart(serial), time.Now().UnixNano()),
				)
			}
		}
	}

	return job, nil
}

type preparedLabelV2Job struct {
	u                  *UtilsStruct
	printer            models.PrinterV2
	template           models.LabelTemplate
	printerName        string
	serial             string
	copies             int
	printLanguage      string
	effectiveLanguage  string
	fileSink           bool
	rawSizeOnly        bool
	rawSettingsApplied bool
	printWidthMm       float64
	printHeightMm      float64
	rotationDeg        int
	dpi                int
	payload            []byte
	tempPath           string
	pdfOutputPath      string
	spoolJobID         uint32
	done               bool
	logged             bool
}

func (j *preparedLabelV2Job) cleanup() {
	if j == nil {
		return
	}
	if j.tempPath != "" {
		_ = os.Remove(j.tempPath)
		j.tempPath = ""
	}
}

func (j *preparedLabelV2Job) submitAndVerify() error {
	if j == nil {
		return errors.New("print job tayyor emas")
	}
	if j.done {
		return nil
	}
	var jobID uint32
	switch j.effectiveLanguage {
	case PrintLanguageTSPL, PrintLanguageZPL:
		id, err := rawPrintViaWorker(
			j.effectiveLanguage,
			j.printerName,
			j.serial,
			RawSpoolDatatype(j.effectiveLanguage),
			j.payload,
		)
		if err != nil {
			return fmt.Errorf("spool: %w", err)
		}
		jobID = id
	default:
		// PrintDocument does not report a job id, so it is recovered by diffing
		// the queue around the submit.
		before := snapshotSpoolJobIDs(j.printerName)
		if err := printImageWindows(
			j.printerName,
			j.serial,
			j.tempPath,
			j.pdfOutputPath,
			j.printWidthMm,
			j.printHeightMm,
			j.copies,
			j.rotationDeg,
		); err != nil {
			return fmt.Errorf("gdi: %w", err)
		}
		jobID = findNewSpoolJobID(j.printerName, j.serial, before)
	}

	j.spoolJobID = jobID
	if err := verifySpoolJob(j.printerName, jobID, j.serial); err != nil {
		return fmt.Errorf("spool: %w", err)
	}
	j.watchSpoolJob(jobID)
	j.logSuccess()
	return nil
}

// watchSpoolJob hands a job that is still queued to the background watcher, so
// the operator is not kept waiting for a printer that is merely slow.
func (j *preparedLabelV2Job) watchSpoolJob(jobID uint32) {
	if jobID == 0 || j.fileSink {
		return
	}
	globalSpoolWatcher.track(watchedSpoolJob{
		u:           j.u,
		printer:     j.printerName,
		jobID:       jobID,
		serial:      j.serial,
		printerV2ID: j.printer.ID,
		lineID:      j.printer.LineID,
		lineName:    strings.TrimSpace(j.printer.LineName),
		templateID:  j.template.ID,
		language:    j.effectiveLanguage,
	})
}

func (j *preparedLabelV2Job) logSuccess() {
	if j == nil || j.u == nil || j.logged {
		return
	}
	j.logged = true
	j.u.DebugLogAny("V2 label printed: ", map[string]any{
		"printer_v2_id":        j.printer.ID,
		"printer":              j.printerName,
		"print_language":       j.printLanguage,
		"effective_language":   j.effectiveLanguage,
		"spool_datatype":       RawSpoolDatatype(j.effectiveLanguage),
		"print_process":        printProcessName(j.effectiveLanguage),
		"file_sink":            j.fileSink,
		"template":             j.template.ID,
		"serial":               j.serial,
		"copies":               j.copies,
		"width_mm":             j.template.WidthMm,
		"height_mm":            j.template.HeightMm,
		"print_width_mm":       j.printWidthMm,
		"print_height_mm":      j.printHeightMm,
		"print_rotation_deg":   normalizePrintRotationDeg(j.template.PrintRotationDeg),
		"dpi":                  j.dpi,
		"use_printer_defaults": j.template.UsePrinterDefaults,
		"size_only":            j.rawSizeOnly,
		"raw_settings_applied": j.rawSettingsApplied,
		"density":              j.template.Density,
		"speed":                j.template.Speed,
		"gap_mm":               j.template.GapMm,
		"language_hint":        ProductionPrintLanguageHint(j.effectiveLanguage),
	})
}

func printProcessName(language string) string {
	switch NormalizePrintLanguage(language) {
	case PrintLanguageTSPL:
		return "tspl-worker"
	case PrintLanguageZPL:
		return "zpl-worker"
	default:
		return "gdi-worker"
	}
}

// resolveEffectivePrintRoute picks spool language and whether density/speed/gap are omitted.
//
// use_printer_defaults:
//   - GDI printer → Windows Printing Preferences (GDI path)
//   - ZPL/TSPL printer → keep RAW language; omit density/speed/gap (firmware defaults)
//   - AC-FILE* → always keep configured ZPL/TSPL (test sink)
func resolveEffectivePrintRoute(printLanguage, printerName string, usePrinterDefaults, sizeOnly bool) (effectiveLanguage string, rawSizeOnly bool) {
	effectiveLanguage = NormalizePrintLanguage(printLanguage)
	rawSizeOnly = sizeOnly
	if IsLabelFileSink(printerName) {
		return effectiveLanguage, rawSizeOnly
	}
	if !usePrinterDefaults {
		return effectiveLanguage, rawSizeOnly
	}
	switch effectiveLanguage {
	case PrintLanguageZPL, PrintLanguageTSPL:
		// "Use defaults" on thermal = do not override tone/speed/gap in the job.
		return effectiveLanguage, true
	default:
		return PrintLanguageGDI, rawSizeOnly
	}
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
	widthMm, heightMm := effectiveLabelDimensions(template.WidthMm, template.HeightMm, template.PrintRotationDeg)
	widthPx := int(mmToDots(widthMm, dpi))
	heightPx := int(mmToDots(heightMm, dpi))
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
		EshikRangi:                  "Oq",
		RangiEng:                    "White",
		KorpusRangiShortname:        "W",
		EshikRangiShortname:         "W",
		RangiKodi:                   "101",
		Brutto:                      "44",
		Netto:                       "39",
		Manzil:                      "Oxangaron",
		ManzilRu:                    "Ахангаран",
		Korxon_nomi:                 "Premier LLC",
		Ishlab_chiqaruvchi_mamlakat: "O'zbekiston",
		Taminot_kuchlanishi_v:       "220В-240В/50Гц",
		Umumiy_hajmi_l:              "211",
		Nominal_tok_quvvati_w:       "62",
		Nominal_tok_kuchi_a:         "0.7",
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
const labelNowBinding = "now"
const labelNowDateFormat = "YYYY-MM-DD HH24:MI"
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
	// Longer tokens first so HH24/MI are not broken by shorter replacements.
	layout = strings.ReplaceAll(layout, "YYYY", "2006")
	layout = strings.ReplaceAll(layout, "YY", "06")
	layout = strings.ReplaceAll(layout, "DD", "02")
	layout = strings.ReplaceAll(layout, "HH24", "15")
	layout = strings.ReplaceAll(layout, "MI", "04")
	layout = strings.ReplaceAll(layout, "MM", "01")
	return t.Format(layout)
}

func resolveDerivedBinding(data map[string]any, binding, dateFormat string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(binding)) {
	case labelTodayBinding:
		return formatLabelDate(time.Now(), dateFormat), true
	case labelNowBinding:
		return formatLabelDate(time.Now(), labelNowDateFormat), true
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

	fc := newLabelFontCache()
	for _, el := range elements {
		if err := drawLabelElement(dc, fc, el, data, dpi); err != nil {
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

func textLinesFitHeight(face font.Face, fontHeight float64, lines []string, maxH, lineSpacing float64) bool {
	_, textH := measureLabelLines(face, fontHeight, lines, lineSpacing)
	return textH <= maxH
}

const labelTextMinScaleX = 0.15

func drawTextInBox(dc *gg.Context, fc *labelFontCache, text string, x, y, w, h float64, dpi int, fontPath string, fontSize float64, fontWeight, align string) error {
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

	resolvedPath := cachedResolveLabelFontPath(fontPath, fontWeight)

	// Shrink for height only; width overflow is handled by horizontal squeeze below
	// so glyph height stays closer to the requested font size.
	for size := fontSize; size >= 4; size -= 0.5 {
		pixelSize := size * float64(dpi) / 72.0
		face, err := fc.face(resolvedPath, pixelSize)
		if err != nil {
			return err
		}
		if textLinesFitHeight(face, labelFontHeight(pixelSize), lines, maxH, lineSpacing) {
			fontSize = size
			break
		}
	}

	// Prefer keeping scaleX >= ~0.35 by shrinking font a bit more when the line
	// is extremely wide; still fall through to exact width squeeze afterward.
	for size := fontSize; size >= 4; size -= 0.5 {
		pixelSize := size * float64(dpi) / 72.0
		face, err := fc.face(resolvedPath, pixelSize)
		if err != nil {
			return err
		}
		textW, _ := measureLabelLines(face, labelFontHeight(pixelSize), lines, lineSpacing)
		if textW <= 0 || textW <= maxW/labelTextMinScaleX {
			fontSize = size
			break
		}
		fontSize = size
	}

	pixelSize := fontSize * float64(dpi) / 72.0
	face, err := fc.face(resolvedPath, pixelSize)
	if err != nil {
		return err
	}
	dc.SetFontFace(face)
	fontH := labelFontHeight(pixelSize)

	textW, totalH := measureLabelLines(face, fontH, lines, lineSpacing)
	blockTop := y + (h-totalH)/2

	scaleX := 1.0
	if textW > maxW && textW > 0 {
		scaleX = maxW / textW
	}

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

	// gg.DrawString only transforms the baseline origin — glyph advances ignore
	// Scale — so width squeeze is done by rasterizing each line then scaling the bitmap.
	for i, line := range lines {
		lineY := blockTop + fontH/2 + float64(i)*fontH*lineSpacing
		lineW := measureLabelLineWidth(face, line)
		baselineY := lineY + 0.5*fontH
		if scaleX >= 1 || lineW <= 0 {
			dc.DrawString(line, xPos-ax*lineW, baselineY)
			continue
		}
		if err := drawScaledLabelLine(dc, face, line, lineW, fontH, xPos, baselineY, ax, scaleX); err != nil {
			return err
		}
	}
	return nil
}

// drawScaledLabelLine renders one line into a temporary image and draws it with
// horizontal scale so long text fits the box width while keeping glyph height.
func drawScaledLabelLine(dc *gg.Context, face font.Face, line string, lineW, fontH, xPos, baselineY, ax, scaleX float64) error {
	const padPx = 2.0
	tw := int(math.Ceil(lineW + padPx*2))
	// Extra vertical room for ascenders/descenders around the baseline.
	th := int(math.Ceil(fontH*2 + padPx*2))
	if tw < 1 {
		tw = 1
	}
	if th < 1 {
		th = 1
	}
	tmp := gg.NewContext(tw, th)
	tmp.SetFontFace(face)
	tmp.SetColor(color.Black)
	baseInTmp := fontH + padPx
	tmp.DrawString(line, padPx, baseInTmp)

	dc.Push()
	dc.Translate(xPos, baselineY-baseInTmp)
	dc.Scale(scaleX, 1)
	dc.DrawImage(tmp.Image(), int(math.Round(-ax*lineW-padPx)), 0)
	dc.Pop()
	return nil
}

func drawLabelElement(dc *gg.Context, fc *labelFontCache, el labelElement, data map[string]any, dpi int) error {
	fontPath := cachedLabelElementFontPath(el.FontFamily)
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

	rot := normalizeElementRotationDeg(el.RotationDeg)
	if rot != 0 {
		dc.Push()
		dc.Translate(x+w/2, y+h/2)
		dc.Rotate(gg.Radians(float64(rot)))
		x, y = -w/2, -h/2
		defer dc.Pop()
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
		return drawTextInBox(dc, fc, text, x, y, w, h, dpi, fontPath, fontSize, el.FontWeight, el.Align)
	case "line":
		stroke := mmToDots(el.StrokeWidth, dpi)
		if stroke <= 0 {
			stroke = mmToDots(0.3, dpi)
		}
		dc.SetColor(color.Black)
		if strings.EqualFold(strings.TrimSpace(el.Orientation), "vertical") {
			dc.DrawRectangle(x+w/2-stroke/2, y, stroke, h)
		} else {
			dc.DrawRectangle(x, y+h/2-stroke/2, w, stroke)
		}
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
		return drawTable(dc, fc, el, data, x, y, w, h, dpi, fontPath)
	case "barcode":
		value := resolveLabelElementText(data, el)
		if value == "" {
			return fmt.Errorf("barcode qiymati bo'sh (element %s)", el.ID)
		}
		bc, err := encodeBarcode(value, el.Format, int(w), int(h))
		if err != nil {
			return fmt.Errorf("barcode yaratilmadi (%s): %w", el.ID, err)
		}
		dc.DrawImage(bc, int(math.Round(x)), int(math.Round(y)))
	case "datamatrix":
		value := resolveLabelElementText(data, el)
		if value == "" {
			return fmt.Errorf("datamatrix qiymati bo'sh (element %s)", el.ID)
		}
		bc, err := datamatrix.Encode(value)
		if err != nil {
			return fmt.Errorf("datamatrix yaratilmadi (%s): %w", el.ID, err)
		}
		scaled, err := barcode.Scale(bc, int(w), int(h))
		if err != nil {
			return fmt.Errorf("datamatrix scale (%s): %w", el.ID, err)
		}
		dc.DrawImage(scaled, int(math.Round(x)), int(math.Round(y)))
	case "qrcode":
		value := resolveLabelElementText(data, el)
		if value == "" {
			return fmt.Errorf("qrcode qiymati bo'sh (element %s)", el.ID)
		}
		bc, err := qr.Encode(value, qr.M, qr.Auto)
		if err != nil {
			return fmt.Errorf("qrcode yaratilmadi (%s): %w", el.ID, err)
		}
		scaled, err := barcode.Scale(bc, int(w), int(h))
		if err != nil {
			return fmt.Errorf("qrcode scale (%s): %w", el.ID, err)
		}
		dc.DrawImage(scaled, int(math.Round(x)), int(math.Round(y)))
	case "image":
		imageSrc := resolveLabelImageSource(data, el)
		if imageSrc == "" {
			// Optional images (e.g. missing brand logo) are skipped.
			return nil
		}
		img, loadedFrom, err := loadLabelImage(imageSrc)
		if err != nil {
			// Missing/deleted upload assets must not block the whole label job.
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

func drawTable(dc *gg.Context, fc *labelFontCache, el labelElement, data map[string]any, x, y, w, h float64, dpi int, fontPath string) error {
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
				cellFontPath := cachedLabelElementFontPath(cellFontFamily)
				if cellFontPath == "" {
					cellFontPath = fontPath
				}
				align := cell.Align
				if align == "" {
					align = "center"
				}
				if err := drawTextInBox(
					dc, fc, text,
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
