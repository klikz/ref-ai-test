package api

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/utils"
)

func (s *ServerModel) LabelTemplatesGetAll(c *gin.Context) {
	data, err := s.Store.Repo().LabelTemplatesGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplatesGetAll", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LabelTemplateGetByID(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateGetByID: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	data, err := s.Store.Repo().LabelTemplateGetByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateGetByID", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LabelTemplateCreate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateCreate: ReadBody", "")
		return
	}

	name, ok := jsonMap["name"].(string)
	if !ok || name == "" {
		s.Utils.SendError(c, errors.New("name is required"), "LabelTemplateCreate", "")
		return
	}

	lineID := int(jsonMap["line_id"].(float64))
	widthMm := 100.0
	heightMm := 50.0
	dpi := 203

	if v, ok := jsonMap["width_mm"].(float64); ok {
		widthMm = v
	}
	if v, ok := jsonMap["height_mm"].(float64); ok {
		heightMm = v
	}
	if v, ok := jsonMap["dpi"].(float64); ok {
		dpi = int(v)
	}
	printRotationDeg := 0
	if v, ok := jsonMap["print_rotation_deg"].(float64); ok {
		printRotationDeg = int(v)
	}
	density := 8
	if v, ok := jsonMap["density"].(float64); ok {
		density = int(v)
	}
	speed := 4
	if v, ok := jsonMap["speed"].(float64); ok {
		speed = int(v)
	}
	gapMm := 2.0
	if v, ok := jsonMap["gap_mm"].(float64); ok {
		gapMm = v
	}
	usePrinterDefaults := true
	if v, ok := jsonMap["use_printer_defaults"].(bool); ok {
		usePrinterDefaults = v
	}
	sizeOnly := false
	if v, ok := jsonMap["size_only"].(bool); ok {
		sizeOnly = v
	}

	var definition json.RawMessage
	if raw, ok := jsonMap["definition"]; ok && raw != nil {
		definition, err = json.Marshal(raw)
		if err != nil {
			s.Utils.SendError(c, err, "LabelTemplateCreate: definition", "")
			return
		}
	}

	userID := c.GetInt("user_id")
	id, err := s.Store.Repo().LabelTemplateCreate(
		name, lineID, widthMm, heightMm, dpi, printRotationDeg,
		density, speed, gapMm, usePrinterDefaults, sizeOnly, definition, userID,
	)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateCreate", "")
		return
	}

	data, err := s.Store.Repo().LabelTemplateGetByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateCreate: get", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LabelTemplateDuplicate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateDuplicate: ReadBody", "")
		return
	}

	sourceID := 0
	if v, ok := jsonMap["id"].(float64); ok {
		sourceID = int(v)
	}
	if sourceID <= 0 {
		s.Utils.SendError(c, errors.New("id is required"), "LabelTemplateDuplicate", "")
		return
	}

	newName := ""
	if v, ok := jsonMap["name"].(string); ok {
		newName = strings.TrimSpace(v)
	}

	userID := c.GetInt("user_id")
	newID, err := s.Store.Repo().LabelTemplateDuplicate(sourceID, newName, userID)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateDuplicate", "")
		return
	}

	data, err := s.Store.Repo().LabelTemplateGetByID(newID)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateDuplicate: get", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LabelTemplateUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateUpdate: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	name, ok := jsonMap["name"].(string)
	if !ok || name == "" {
		s.Utils.SendError(c, errors.New("name is required"), "LabelTemplateUpdate", "")
		return
	}

	lineID := int(jsonMap["line_id"].(float64))
	widthMm := 100.0
	heightMm := 50.0
	dpi := 203

	if v, ok := jsonMap["width_mm"].(float64); ok {
		widthMm = v
	}
	if v, ok := jsonMap["height_mm"].(float64); ok {
		heightMm = v
	}
	if v, ok := jsonMap["dpi"].(float64); ok {
		dpi = int(v)
	}
	printRotationDeg := 0
	if v, ok := jsonMap["print_rotation_deg"].(float64); ok {
		printRotationDeg = int(v)
	}
	density := 8
	if v, ok := jsonMap["density"].(float64); ok {
		density = int(v)
	}
	speed := 4
	if v, ok := jsonMap["speed"].(float64); ok {
		speed = int(v)
	}
	gapMm := 2.0
	if v, ok := jsonMap["gap_mm"].(float64); ok {
		gapMm = v
	}
	usePrinterDefaults := true
	if v, ok := jsonMap["use_printer_defaults"].(bool); ok {
		usePrinterDefaults = v
	}
	sizeOnly := false
	if v, ok := jsonMap["size_only"].(bool); ok {
		sizeOnly = v
	}

	var definition json.RawMessage
	if raw, ok := jsonMap["definition"]; ok && raw != nil {
		definition, err = json.Marshal(raw)
		if err != nil {
			s.Utils.SendError(c, err, "LabelTemplateUpdate: definition", "")
			return
		}
	}

	err = s.Store.Repo().LabelTemplateUpdate(
		id, name, lineID, widthMm, heightMm, dpi, printRotationDeg,
		density, speed, gapMm, usePrinterDefaults, sizeOnly, definition,
	)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateUpdate", "")
		return
	}

	data, err := s.Store.Repo().LabelTemplateGetByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateUpdate: get", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LabelTemplateDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateDelete: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	err = s.Store.Repo().LabelTemplateDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) LabelTemplateImageUpload(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateImageUpload: ReadBody", "")
		return
	}

	templateID := int(jsonMap["template_id"].(float64))
	if templateID == 0 {
		s.Utils.SendError(c, errors.New("template_id is required"), "LabelTemplateImageUpload", "")
		return
	}

	_, err = s.Store.Repo().LabelTemplateGetByID(templateID)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateImageUpload: LabelTemplateGetByID", "")
		return
	}

	file64, ok := jsonMap["file64"].(string)
	if !ok || file64 == "" {
		s.Utils.SendError(c, errors.New("file64 is required"), "LabelTemplateImageUpload", "")
		return
	}

	data, extension, err := decodeComponentPhoto(file64)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplateImageUpload: decode", "")
		return
	}
	if len(data) > 5*1024*1024 {
		s.Utils.SendError(c, errors.New("photo size must be less than 5MB"), "LabelTemplateImageUpload", "")
		return
	}

	dir := filepath.Join("uploads", "labels")
	if err := os.MkdirAll(dir, 0755); err != nil {
		s.Utils.SendError(c, err, "LabelTemplateImageUpload: MkdirAll", "")
		return
	}

	filename := fmt.Sprintf("%d_%d%s", templateID, time.Now().UnixNano(), extension)
	fullPath := filepath.Join(dir, filename)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		s.Utils.SendError(c, err, "LabelTemplateImageUpload: WriteFile", "")
		return
	}

	s.Utils.SendOK(c, "/uploads/labels/"+filename)
}

func (s *ServerModel) LabelTemplatePreview(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplatePreview: ReadBody", "")
		return
	}

	templateID := 0
	if v, ok := jsonMap["id"].(float64); ok {
		templateID = int(v)
	}

	printerV2ID := 0
	if v, ok := jsonMap["printer_v2_id"].(float64); ok {
		printerV2ID = int(v)
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer_v2_id is required"), "LabelTemplatePreview", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplatePreview: PrinterV2GetByID", "")
		return
	}
	if templateID > 0 && printerV2.LabelTemplateID != templateID {
		s.Utils.SendError(c, errors.New("printer ushbu shablonga bog'lanmagan"), "LabelTemplatePreview", "")
		return
	}

	widthMm := 100.0
	heightMm := 50.0
	dpi := 203
	if v, ok := jsonMap["width_mm"].(float64); ok {
		widthMm = v
	}
	if v, ok := jsonMap["height_mm"].(float64); ok {
		heightMm = v
	}
	if v, ok := jsonMap["dpi"].(float64); ok {
		dpi = int(v)
	}
	printRotationDeg := 0
	if v, ok := jsonMap["print_rotation_deg"].(float64); ok {
		printRotationDeg = int(v)
	}

	var definition json.RawMessage
	if raw, ok := jsonMap["definition"]; ok && raw != nil {
		definition, err = json.Marshal(raw)
		if err != nil {
			s.Utils.SendError(c, err, "LabelTemplatePreview: definition", "")
			return
		}
	} else if templateID > 0 {
		stored, getErr := s.Store.Repo().LabelTemplateGetByID(templateID)
		if getErr != nil {
			s.Utils.SendError(c, getErr, "LabelTemplatePreview: LabelTemplateGetByID", "")
			return
		}
		definition = stored.Definition
		if widthMm == 100 && heightMm == 50 {
			widthMm = stored.WidthMm
			heightMm = stored.HeightMm
			dpi = stored.DPI
		}
		if _, ok := jsonMap["print_rotation_deg"]; !ok {
			printRotationDeg = stored.PrintRotationDeg
		}
	}

	template := models.LabelTemplate{
		ID:               templateID,
		WidthMm:          widthMm,
		HeightMm:         heightMm,
		DPI:              dpi,
		PrintRotationDeg: printRotationDeg,
		Definition:       definition,
	}

	overrides := map[string]any{}
	if v, ok := jsonMap["serial"].(string); ok {
		overrides["serial"] = v
	}
	if v, ok := jsonMap["acc_serial"].(string); ok {
		overrides["acc_serial"] = v
	}
	sampleData := utils.MergeLabelPreviewSampleData(overrides)
	previewBrand := ""
	if modelRaw, ok := sampleData["model"].(map[string]any); ok {
		if v, ok := modelRaw["brend"]; ok {
			previewBrand = strings.TrimSpace(fmt.Sprint(v))
		}
	}
	if err := s.applyBrandLogoData(previewBrand, sampleData); err != nil {
		s.Utils.SendError(c, err, "LabelTemplatePreview: BrandLogosAll", "")
		return
	}

	dataURL, widthPx, heightPx, err := s.Utils.RenderLabelV2PreviewDataURL(template, sampleData)
	if err != nil {
		s.Utils.SendError(c, err, "LabelTemplatePreview", "")
		return
	}

	s.Utils.SendOK(c, gin.H{
		"preview_png":  dataURL,
		"width_px":     widthPx,
		"height_px":    heightPx,
		"printer_name": printerV2.PrinterName,
		"dpi":          dpi,
		"width_mm":     widthMm,
		"height_mm":    heightMm,
	})
}
