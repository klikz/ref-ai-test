package api

import (
	"errors"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
)

func printerV2ResponseItem(printer models.PrinterV2) gin.H {
	lang := utils.NormalizePrintLanguage(printer.PrintLanguage)
	return gin.H{
		"id":                  printer.ID,
		"line_id":             printer.LineID,
		"line_name":           printer.LineName,
		"printer_name":        printer.PrinterName,
		"address":             printer.Address,
		"label_template_id":   printer.LabelTemplateID,
		"label_template_name": printer.LabelTemplateName,
		"print_language":      lang,
		"language_hint":       utils.ProductionPrintLanguageHint(lang),
	}
}

func (s *ServerModel) PrintersV2GetAll(c *gin.Context) {
	data, err := s.Store.Repo().PrintersV2GetAll()
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2GetAll", "")
		return
	}

	items := make([]gin.H, 0, len(data))
	for _, printer := range data {
		items = append(items, printerV2ResponseItem(printer))
	}
	s.Utils.SendOK(c, items)
}

func (s *ServerModel) PrintersV2GetByLine(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2GetByLine: ReadBody", "")
		return
	}

	lineID := int(jsonMap["line_id"].(float64))
	if lineID <= 0 {
		s.Utils.SendError(c, errors.New("line_id noto'g'ri"), "PrintersV2GetByLine", "")
		return
	}

	data, err := s.Store.Repo().PrintersV2GetByLine(lineID)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2GetByLine", "")
		return
	}

	items := make([]gin.H, 0, len(data))
	for _, printer := range data {
		items = append(items, printerV2ResponseItem(printer))
	}
	s.Utils.SendOK(c, items)
}

func (s *ServerModel) PrinterV2GetByID(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrinterV2GetByID: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	data, err := s.Store.Repo().PrinterV2GetByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "PrinterV2GetByID", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) PrintersV2Add(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Add: ReadBody", "")
		return
	}

	lineID := int(jsonMap["line_id"].(float64))
	printerName, ok := jsonMap["printer_name"].(string)
	if !ok || printerName == "" {
		s.Utils.SendError(c, errors.New("printer_name is required"), "PrintersV2Add", "")
		return
	}

	address := ""
	if v, ok := jsonMap["address"].(string); ok {
		address = v
	}

	labelTemplateID := 0
	if v, ok := jsonMap["label_template_id"].(float64); ok {
		labelTemplateID = int(v)
	}

	printLanguage := utils.PrintLanguageGDI
	detectedLanguage, driverName, detectReason := utils.DetectPrinterLanguage(printerName)
	printLanguage = detectedLanguage
	if v, ok := jsonMap["print_language"].(string); ok && strings.TrimSpace(v) != "" {
		printLanguage = utils.NormalizePrintLanguage(v)
	}

	userID := c.GetInt("user_id")
	id, err := s.Store.Repo().PrintersV2Add(lineID, printerName, address, labelTemplateID, printLanguage, userID)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Add", "")
		return
	}

	data, err := s.Store.Repo().PrinterV2GetByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Add: get", "")
		return
	}
	s.Utils.SendOK(c, gin.H{
		"id":                  data.ID,
		"line_id":             data.LineID,
		"line_name":           data.LineName,
		"printer_name":        data.PrinterName,
		"address":             data.Address,
		"label_template_id":   data.LabelTemplateID,
		"label_template_name": data.LabelTemplateName,
		"print_language":      utils.NormalizePrintLanguage(data.PrintLanguage),
		"language_hint":       utils.ProductionPrintLanguageHint(data.PrintLanguage),
		"driver_name":         driverName,
		"detect_reason":       detectReason,
	})
}

func (s *ServerModel) PrintersV2Delete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Delete: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	err = s.Store.Repo().PrintersV2Delete(id)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Delete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

type localPrinterView struct {
	Name         string `json:"name"`
	IsDefault    bool   `json:"is_default"`
	Configured   bool   `json:"configured"`
	LineName     string `json:"line_name,omitempty"`
	ConfiguredID int    `json:"configured_id,omitempty"`
}

func (s *ServerModel) PrintersV2LocalList(c *gin.Context) {
	localPrinters, hostname, err := utils.LocalInstalledPrinters()
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2LocalList", "")
		return
	}

	configured, err := s.Store.Repo().PrintersV2GetAll()
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2LocalList: PrintersV2GetAll", "")
		return
	}

	byName := map[string]struct {
		id       int
		lineName string
	}{}
	for _, p := range configured {
		byName[p.PrinterName] = struct {
			id       int
			lineName string
		}{id: p.ID, lineName: p.LineName}
	}

	items := make([]localPrinterView, 0, len(localPrinters))
	for _, p := range localPrinters {
		item := localPrinterView{
			Name:      p.Name,
			IsDefault: p.IsDefault,
		}
		if cfg, ok := byName[p.Name]; ok {
			item.Configured = true
			item.LineName = cfg.lineName
			item.ConfiguredID = cfg.id
		}
		items = append(items, item)
	}

	s.Utils.SendOK(c, gin.H{
		"hostname": hostname,
		"printers": items,
	})
}

func (s *ServerModel) PrintersV2Jobs(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Jobs: ReadBody", "")
		return
	}

	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer_v2_id is required"), "PrintersV2Jobs", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Jobs: PrinterV2GetByID", "")
		return
	}
	if printerV2.PrinterName == "" {
		s.Utils.SendError(c, errors.New("printer nomi bo'sh"), "PrintersV2Jobs", "")
		return
	}

	jobs, err := utils.LocalPrinterJobs(printerV2.PrinterName)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Jobs: LocalPrinterJobs", "")
		return
	}

	s.Utils.SendOK(c, gin.H{
		"printer_v2_id": printerV2.ID,
		"printer_name":  printerV2.PrinterName,
		"jobs":          jobs,
	})
}

func (s *ServerModel) PrintersV2DetectLanguage(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2DetectLanguage: ReadBody", "")
		return
	}

	printerName, _ := jsonMap["printer_name"].(string)
	if strings.TrimSpace(printerName) == "" {
		s.Utils.SendError(c, errors.New("printer_name is required"), "PrintersV2DetectLanguage", "")
		return
	}

	utils.InvalidatePrinterLanguageCache(printerName)
	language, driverName, reason := utils.DetectPrinterLanguage(printerName)
	s.Utils.SendOK(c, gin.H{
		"print_language": language,
		"driver_name":    driverName,
		"detect_reason":  reason,
		"language_hint":  utils.ProductionPrintLanguageHint(language),
	})
}

func (s *ServerModel) PrintersV2RefreshGDI(c *gin.Context) {
	utils.InvalidatePrinterLanguageCache("")
	if err := utils.RefreshGDIPrintWorker(); err != nil {
		s.Utils.SendError(c, err, "PrintersV2RefreshGDI", "")
		return
	}
	s.Utils.SendOK(c, gin.H{
		"message": "Windows printer sozlamalari yangilandi (GDI worker qayta ishga tushdi)",
		"metrics": utils.PrintV2Metrics(),
	})
}

func (s *ServerModel) PrintersV2UpdateLanguage(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2UpdateLanguage: ReadBody", "")
		return
	}

	id, ok := jsonMap["id"].(float64)
	if !ok || int(id) <= 0 {
		s.Utils.SendError(c, errors.New("id noto'g'ri"), "PrintersV2UpdateLanguage", "")
		return
	}

	langRaw, _ := jsonMap["print_language"].(string)
	printLanguage := utils.NormalizePrintLanguage(langRaw)

	if err := s.Store.Repo().PrintersV2UpdateLanguage(int(id), printLanguage); err != nil {
		s.Utils.SendError(c, err, "PrintersV2UpdateLanguage", "")
		return
	}

	data, err := s.Store.Repo().PrinterV2GetByID(int(id))
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2UpdateLanguage: get", "")
		return
	}
	utils.InvalidatePrinterLanguageCache(data.PrinterName)
	item := printerV2ResponseItem(data)
	item["language_hint"] = utils.ProductionPrintLanguageHint(data.PrintLanguage)
	s.Utils.SendOK(c, item)
}

func (s *ServerModel) PrintersV2TestPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2TestPrint: ReadBody", "")
		return
	}

	id, ok := jsonMap["id"].(float64)
	if !ok || int(id) <= 0 {
		s.Utils.SendError(c, errors.New("id noto'g'ri"), "PrintersV2TestPrint", "")
		return
	}

	printer, err := s.Store.Repo().PrinterV2GetByID(int(id))
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2TestPrint: get printer", "")
		return
	}
	if printer.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("printerga shablon biriktirilmagan"), "PrintersV2TestPrint", "")
		return
	}
	if err := utils.PrinterReachable(printer.PrinterName); err != nil {
		s.Utils.SendError(c, err, "PrintersV2TestPrint: health", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printer.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2TestPrint: get template", "")
		return
	}

	printData := utils.BuildLabelPreviewSampleData()
	printData["serial"] = "TEST-PRINT"
	if err := s.Utils.PrintLabelV2(template, printer, 1, printData); err != nil {
		s.Utils.SendError(c, err, "PrintersV2TestPrint", "")
		return
	}
	s.Utils.SendOK(c, gin.H{
		"message":        "Test print yuborildi",
		"print_language": utils.NormalizePrintLanguage(printer.PrintLanguage),
		"language_hint":  utils.ProductionPrintLanguageHint(printer.PrintLanguage),
		"metrics":        utils.PrintV2Metrics(),
	})
}

func (s *ServerModel) PrintersV2Metrics(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Metrics: ReadBody", "")
		return
	}
	dateFrom, _ := jsonMap["date_from"].(string)
	dateTo, _ := jsonMap["date_to"].(string)
	from, to, err := store.ParseMetricsDateRange(dateFrom, dateTo)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Metrics: date", "")
		return
	}

	summary, err := s.Store.Repo().PrintV2MetricsSummary(from, to)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Metrics", "")
		return
	}
	byLine, err := s.Store.Repo().PrintV2MetricsByLine(from, to)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2Metrics: by line", "")
		return
	}
	live := utils.PrintV2Metrics()
	summary.InFlight = live.InFlight
	summary.DateFrom = from.Format("2006-01-02")
	summary.DateTo = to.Add(-time.Nanosecond).Format("2006-01-02")

	s.Utils.SendOK(c, gin.H{
		"success":   summary.Success,
		"fail":      summary.Fail,
		"total_ms":  summary.TotalMs,
		"last_ms":   summary.LastMs,
		"in_flight": summary.InFlight,
		"date_from": summary.DateFrom,
		"date_to":   summary.DateTo,
		"lines":     byLine,
		"process":   live,
	})
}

func (s *ServerModel) PrintersV2MetricsErrors(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2MetricsErrors: ReadBody", "")
		return
	}
	dateFrom, _ := jsonMap["date_from"].(string)
	dateTo, _ := jsonMap["date_to"].(string)
	from, to, err := store.ParseMetricsDateRange(dateFrom, dateTo)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2MetricsErrors: date", "")
		return
	}

	limit := 100
	offset := 0
	if v, ok := jsonMap["limit"].(float64); ok && int(v) > 0 {
		limit = int(v)
	}
	if v, ok := jsonMap["offset"].(float64); ok && int(v) >= 0 {
		offset = int(v)
	}

	items, err := s.Store.Repo().PrintV2EventsList(from, to, true, limit, offset)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2MetricsErrors", "")
		return
	}
	s.Utils.SendOK(c, gin.H{
		"items":     items,
		"date_from": from.Format("2006-01-02"),
		"date_to":   to.Add(-time.Nanosecond).Format("2006-01-02"),
		"limit":     limit,
		"offset":    offset,
	})
}

func (s *ServerModel) PrintersV2MetricsReset(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2MetricsReset: ReadBody", "")
		return
	}
	dateFrom, _ := jsonMap["date_from"].(string)
	dateTo, _ := jsonMap["date_to"].(string)
	from, to, err := store.ParseMetricsDateRange(dateFrom, dateTo)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2MetricsReset: date", "")
		return
	}

	n, err := s.Store.Repo().PrintV2EventsReset(from, to)
	if err != nil {
		s.Utils.SendError(c, err, "PrintersV2MetricsReset", "")
		return
	}
	s.Utils.SendOK(c, gin.H{
		"deleted":   n,
		"date_from": from.Format("2006-01-02"),
		"date_to":   to.Add(-time.Nanosecond).Format("2006-01-02"),
	})
}
