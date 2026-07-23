package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/utils"
)

const t2V2LineID = 5

func (s *ServerModel) linesT2V2ExecutePrint(
	printerV2 models.PrinterV2,
	copyCount int,
	serial string,
	selectedModel models.ModelInfo,
	gsCode string,
) error {
	if printerV2.LineID != t2V2LineID {
		return errors.New("printer T2 liniyasiga tegishli emas")
	}
	if printerV2.LabelTemplateID <= 0 {
		return errors.New("etiketka shablon tanlanmagan")
	}
	if strings.TrimSpace(printerV2.PrinterName) == "" {
		return errors.New("printer nomi bo'sh")
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		return err
	}

	printData := utils.BuildLabelPrintData(serial, "", selectedModel, gsCode)
	if err := s.applyBrandLogoData(selectedModel.Brend, printData); err != nil {
		return err
	}
	return s.Utils.PrintLabelV2(template, printerV2, copyCount, printData)
}

func (s *ServerModel) linesT2V2ResolveGSCode(serial string, template models.LabelTemplate) (string, error) {
	needsGSCode, err := utils.LabelTemplateNeedsGSCode(template)
	if err != nil {
		return "", err
	}
	if !needsGSCode {
		return "", nil
	}

	gsCode, err := s.Store.Repo().GsCodeBySerial(serial)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(gsCode) == "" {
		return "", errors.New("ushbu serial uchun gs_code topilmadi")
	}
	return gsCode, nil
}

func (s *ServerModel) LinesT2V2SerialPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	userID := c.GetInt("user_id")
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	copyCount := 1
	if rawCopy, ok := jsonMap["copy"].(float64); ok && int(rawCopy) > 0 {
		copyCount = int(rawCopy)
	}

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT2V2SerialPrint: serial", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesT2V2SerialPrint", "")
		return
	}

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint: linesT3SelectedModel", "")
		return
	}

	planWarning, err := s.checkProductionPlanAllowMissing(t2V2LineID, selectedModel.ID, 0, 1)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint: plan", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != t2V2LineID {
		s.Utils.SendError(c, errors.New("printer T2 liniyasiga tegishli emas"), "LinesT2V2SerialPrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesT2V2SerialPrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint: LabelTemplateGetByID", "")
		return
	}

	transfer, err := s.Store.Repo().ProductLineTransfer(4, t2V2LineID, 0, userID, selectedModel.ID, serial, "")
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint: ProductLineTransfer", "")
		return
	}

	gsCode, err := s.linesT2V2ResolveGSCode(serial, template)
	if err != nil {
		_ = s.Store.Repo().ProductLineTransferRollback(transfer)
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint: GsCodeBySerial", "")
		return
	}

	if err := s.linesT2V2ExecutePrint(printerV2, copyCount, serial, selectedModel, gsCode); err != nil {
		if rollbackErr := s.Store.Repo().ProductLineTransferRollback(transfer); rollbackErr != nil {
			s.Utils.SendError(c, rollbackErr, "LinesT2V2SerialPrint: ProductLineTransferRollback", "")
			return
		}
		s.Utils.SendError(c, err, "LinesT2V2SerialPrint", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"plan_warning": planWarning,
	})
}

func (s *ServerModel) LinesT2V2SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialRePrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	const copyCount = 1

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT2V2SerialRePrint: serial", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesT2V2SerialRePrint", "")
		return
	}

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialRePrint: linesT3SelectedModel", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialRePrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != t2V2LineID {
		s.Utils.SendError(c, errors.New("printer T2 liniyasiga tegishli emas"), "LinesT2V2SerialRePrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesT2V2SerialRePrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialRePrint: LabelTemplateGetByID", "")
		return
	}

	gsCode, err := s.linesT2V2ResolveGSCode(serial, template)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialRePrint: GsCodeBySerial", "")
		return
	}

	if err := s.linesT2V2ExecutePrint(printerV2, copyCount, serial, selectedModel, gsCode); err != nil {
		s.Utils.SendError(c, err, "LinesT2V2SerialRePrint", "")
		return
	}

	s.Utils.SendOK(c, selectedModel)
}
