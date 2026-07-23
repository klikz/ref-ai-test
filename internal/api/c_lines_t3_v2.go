package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
)

const t3V2LineID = 6

func (s *ServerModel) linesT3V2ExecutePrint(
	printerV2 models.PrinterV2,
	copyCount int,
	serial string,
	accSerial string,
	selectedModel models.ModelInfo,
	gsCode string,
) error {
	if printerV2.LineID != t3V2LineID {
		return errors.New("printer T3 liniyasiga tegishli emas")
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

	printData := utils.BuildLabelPrintData(serial, accSerial, selectedModel, gsCode)
	if err := s.applyBrandLogoData(selectedModel.Brend, printData); err != nil {
		return err
	}
	return s.Utils.PrintLabelV2(template, printerV2, copyCount, printData)
}

func (s *ServerModel) linesT3V2ResolveGSCode(serial string, template models.LabelTemplate) (string, error) {
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

func (s *ServerModel) LinesT3V2SerialPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	accSerial, _ := jsonMap["acc_serial"].(string)
	accSerial = strings.TrimSpace(accSerial)
	userID := c.GetInt("user_id")
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	copyCount := 1
	if rawCopy, ok := jsonMap["copy"].(float64); ok && int(rawCopy) > 0 {
		copyCount = int(rawCopy)
	}

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT3V2SerialPrint: serial", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesT3V2SerialPrint", "")
		return
	}

	s.publishT3V2Serial(serial)

	modelShortInfo, err := s.Store.Repo().ModelsShortInfoBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, errors.New("Serial Xato"), "LinesT3V2SerialPrint: ModelsShortInfoBySerial", "")
		return
	}

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint: linesT3SelectedModel", "")
		return
	}

	planWarning, err := s.checkProductionPlanAllowMissing(t3V2LineID, selectedModel.ID, 0, 1)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint: plan", "")
		return
	}

	if accSerial == "" {
		s.Utils.SendError(c, errors.New("Aksessuar nomer bo'sh"), "LinesT3V2SerialPrint: acc_serial", "")
		return
	}
	if !store.ModelAccSerialMatches(accSerial, modelShortInfo.AccSerial) {
		s.Utils.SendError(c, errors.New("Aksessuar nomer mos emas"), "LinesT3V2SerialPrint: acc_serial", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != t3V2LineID {
		s.Utils.SendError(c, errors.New("printer T3 liniyasiga tegishli emas"), "LinesT3V2SerialPrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesT3V2SerialPrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint: LabelTemplateGetByID", "")
		return
	}

	transfer, err := s.Store.Repo().ProductLineTransfer(5, t3V2LineID, 0, userID, selectedModel.ID, serial, accSerial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint: ProductLineTransfer", "")
		return
	}

	cfg := utils.LoadCameraConfigFromEnv()
	var photoDone <-chan t3ParallelPhotoResult
	if cfg.Enabled {
		photoDone = s.startParallelT3ScanPhoto(c.Request.Context(), transfer.ToProductID, userID, serial, cfg)
	}

	gsCode, err := s.linesT3V2ResolveGSCode(serial, template)
	if err != nil {
		_ = s.Store.Repo().ProductLineTransferRollback(transfer)
		if photoDone != nil {
			s.discardParallelT3Photo(photoDone)
		}
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint: GsCodeBySerial", "")
		return
	}

	if err := s.linesT3V2ExecutePrint(printerV2, copyCount, serial, accSerial, selectedModel, gsCode); err != nil {
		if rollbackErr := s.Store.Repo().ProductLineTransferRollback(transfer); rollbackErr != nil {
			if photoDone != nil {
				s.discardParallelT3Photo(photoDone)
			}
			s.Utils.SendError(c, rollbackErr, "LinesT3V2SerialPrint: ProductLineTransferRollback", "")
			return
		}
		if photoDone != nil {
			s.discardParallelT3Photo(photoDone)
		}
		s.Utils.SendError(c, err, "LinesT3V2SerialPrint", "")
		return
	}

	resp := map[string]any{
		"plan_warning": planWarning,
	}

	if photoDone != nil {
		s.attachParallelT3PhotoResponse(resp, serial, <-photoDone)
	}

	s.Utils.SendOK(c, resp)
}

func (s *ServerModel) LinesT3V2SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialRePrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	const copyCount = 1

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT3V2SerialRePrint: serial", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesT3V2SerialRePrint", "")
		return
	}

	s.publishT3V2Serial(serial)

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialRePrint: linesT3SelectedModel", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialRePrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != t3V2LineID {
		s.Utils.SendError(c, errors.New("printer T3 liniyasiga tegishli emas"), "LinesT3V2SerialRePrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesT3V2SerialRePrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialRePrint: LabelTemplateGetByID", "")
		return
	}

	gsCode, err := s.linesT3V2ResolveGSCode(serial, template)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialRePrint: GsCodeBySerial", "")
		return
	}

	accSerial, err := s.Store.Repo().ProductAccSerialBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialRePrint: ProductAccSerialBySerial", "")
		return
	}

	if err := s.linesT3V2ExecutePrint(printerV2, copyCount, serial, accSerial, selectedModel, gsCode); err != nil {
		s.Utils.SendError(c, err, "LinesT3V2SerialRePrint", "")
		return
	}

	s.Utils.SendOK(c, selectedModel)
}

func (s *ServerModel) LinesT3V2SerialCurrent(c *gin.Context) {
	s.Utils.SendOK(c, s.CurrentT3V2Serial)
}
