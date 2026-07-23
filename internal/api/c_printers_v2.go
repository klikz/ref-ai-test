package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/utils"
)

func printerV2ResponseItem(printer models.PrinterV2) gin.H {
	return gin.H{
		"id":                  printer.ID,
		"line_id":             printer.LineID,
		"line_name":           printer.LineName,
		"printer_name":        printer.PrinterName,
		"address":             printer.Address,
		"label_template_id":   printer.LabelTemplateID,
		"label_template_name": printer.LabelTemplateName,
		"print_language":      utils.NormalizePrintLanguage(printer.PrintLanguage),
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
		"id":                 data.ID,
		"line_id":            data.LineID,
		"line_name":          data.LineName,
		"printer_name":       data.PrinterName,
		"address":            data.Address,
		"label_template_id":  data.LabelTemplateID,
		"label_template_name": data.LabelTemplateName,
		"print_language":     data.PrintLanguage,
		"driver_name":        driverName,
		"detect_reason":      detectReason,
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

	language, driverName, reason := utils.DetectPrinterLanguage(printerName)
	s.Utils.SendOK(c, gin.H{
		"print_language": language,
		"driver_name":    driverName,
		"detect_reason":  reason,
	})
}
