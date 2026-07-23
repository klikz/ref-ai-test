package api

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/utils"
)

const YigishLineID = 1

func (s *ServerModel) linesYigishV2ExecutePrint(
	printerV2 models.PrinterV2,
	copyCount int,
	serial string,
	selectedModel models.ModelInfo,
	gsCode string,
) error {
	if printerV2.LineID != YigishLineID {
		return errors.New("printer Yi'g'ish liniyasiga tegishli emas")
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

type yigishPrintAttempt struct {
	modelID        int
	countAllocated bool
	productID      int
	gsID           int
	serial         string
}

func (s *ServerModel) rollbackYigishPrintAttempt(attempt yigishPrintAttempt) error {
	if attempt.gsID > 0 {
		if err := s.Store.Repo().GsCodeUpdateUndo(attempt.gsID); err != nil {
			return err
		}
	}
	if attempt.productID > 0 {
		if err := s.Store.Repo().LinesDeleteProduct(attempt.productID); err != nil {
			return err
		}
	}
	if attempt.countAllocated {
		if _, err := s.Store.Repo().ModelsUpdateCountMinus(attempt.modelID); err != nil {
			return err
		}
	}
	return nil
}

func (s *ServerModel) linesYigishV2PrintOne(
	printerV2 models.PrinterV2,
	selectedModel models.ModelInfo,
	userID int,
) (serial string, gsCode string, err error) {
	attempt := yigishPrintAttempt{modelID: selectedModel.ID}

	count, err := s.Store.Repo().ModelsUpdateCount(selectedModel.ID)
	if err != nil {
		return "", "", err
	}
	attempt.countAllocated = true

	fullSerial, err := s.GenerateSerial(selectedModel.Seriya_raqami, count)
	if err != nil {
		_ = s.rollbackYigishPrintAttempt(attempt)
		return "", "", err
	}
	attempt.serial = fullSerial

	productID, err := s.Store.Repo().LinesAddProduct(YigishLineID, 1, userID, selectedModel.ID, fullSerial, "")
	if err != nil {
		_ = s.rollbackYigishPrintAttempt(attempt)
		return "", "", err
	}
	attempt.productID = productID

	gsID, err := s.Store.Repo().GsCodeUpdate(selectedModel.ID, productID, userID)
	if err != nil {
		_ = s.rollbackYigishPrintAttempt(attempt)
		return "", "", err
	}
	attempt.gsID = gsID

	productInfo, err := s.Store.Repo().LinesGetInfoByProductId(productID)
	if err != nil {
		_ = s.rollbackYigishPrintAttempt(attempt)
		return "", "", err
	}

	gsCode = productInfo.GS1Data
	if err := s.linesYigishV2ExecutePrint(printerV2, 1, fullSerial, selectedModel, gsCode); err != nil {
		_ = s.rollbackYigishPrintAttempt(attempt)
		return "", "", err
	}

	if err := s.Store.Repo().ProductParamsInsertYigish(fullSerial, gsCode, selectedModel.ID, userID); err != nil {
		_ = s.rollbackYigishPrintAttempt(attempt)
		return "", "", err
	}

	return fullSerial, gsCode, nil
}

func (s *ServerModel) linesYigishV2ModelByProduct(productID int) (models.ModelInfo, error) {
	modelID, err := s.Store.Repo().LinesGetModelIDByProductID(productID)
	if err != nil {
		return models.ModelInfo{}, err
	}
	if modelID == 0 {
		return models.ModelInfo{}, errors.New("model topilmadi")
	}

	selectedModel, err := s.Store.Repo().ModelsGetByID(modelID)
	if err != nil {
		return models.ModelInfo{}, err
	}
	if selectedModel.ID == 0 {
		return models.ModelInfo{}, errors.New("model topilmadi")
	}
	return selectedModel, nil
}

func (s *ServerModel) linesYigishV2ResolveGSCodeForReprint(
	serial string,
	template models.LabelTemplate,
	modelID int,
	assignGsCode bool,
	userID int,
) (string, int, error) {
	needsGSCode, err := utils.LabelTemplateNeedsGSCode(template)
	if err != nil {
		return "", 0, err
	}
	if !needsGSCode {
		return "", 0, nil
	}

	gsCode, err := s.Store.Repo().GsCodeBySerial(serial)
	if err != nil {
		return "", 0, err
	}
	if strings.TrimSpace(gsCode) != "" {
		return gsCode, 0, nil
	}

	if !assignGsCode {
		return "", 0, errors.New("gscode biriktirilmagan")
	}

	productID, err := s.Store.Repo().LinesGetProductIdBySerial(YigishLineID, serial)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, errors.New("ushbu serial Yi'g'ish liniyasida chiqmagan")
		}
		return "", 0, err
	}
	if productID == 0 {
		return "", 0, errors.New("ushbu serial Yi'g'ish liniyasida chiqmagan")
	}

	gsID, err := s.Store.Repo().GsCodeUpdate(modelID, productID, userID)
	if err != nil {
		return "", 0, err
	}

	productInfo, err := s.Store.Repo().LinesGetInfoByProductId(productID)
	if err != nil {
		_ = s.Store.Repo().GsCodeUpdateUndo(gsID)
		return "", 0, err
	}

	return productInfo.GS1Data, gsID, nil
}

func (s *ServerModel) LinesYigishV2Print(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2Print: ReadBody", "")
		return
	}

	userID := c.GetInt("user_id")
	modelID := 0
	if v, ok := jsonMap["model_id"].(float64); ok {
		modelID = int(v)
	}
	printerV2ID := 0
	if v, ok := jsonMap["printer_v2_id"].(float64); ok {
		printerV2ID = int(v)
	}
	quantity := 1
	if v, ok := jsonMap["quantity"].(float64); ok && int(v) > 0 {
		quantity = int(v)
	}

	if modelID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesYigishV2Print", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesYigishV2Print", "")
		return
	}

	selectedModel, err := s.Store.Repo().ModelsGetByID(modelID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2Print: ModelsGetByID", "")
		return
	}
	if selectedModel.ID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesYigishV2Print", "")
		return
	}

	if err := s.enforceProductionPlan(YigishLineID, selectedModel.ID, 0, quantity); err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2Print: plan", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2Print: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != YigishLineID {
		s.Utils.SendError(c, errors.New("printer Yi'g'ish liniyasiga tegishli emas"), "LinesYigishV2Print", "")
		return
	}

	lastGSCode := ""
	lastSerial := ""
	for i := 0; i < quantity; i++ {
		fullSerial, gsCode, err := s.linesYigishV2PrintOne(printerV2, selectedModel, userID)
		if err != nil {
			s.Utils.SendError(c, err, "LinesYigishV2Print", "")
			return
		}
		lastGSCode = gsCode
		lastSerial = fullSerial
	}

	s.Utils.SendOK(c, map[string]any{
		"model":  selectedModel,
		"serial": lastSerial,
		"gscode": lastGSCode,
	})
}

func (s *ServerModel) LinesYigishV2RePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2RePrint: ReadBody", "")
		return
	}

	userID := c.GetInt("user_id")
	serial, _ := jsonMap["serial"].(string)
	printerV2ID := 0
	if v, ok := jsonMap["printer_v2_id"].(float64); ok {
		printerV2ID = int(v)
	}
	assignGsCode := false
	if v, ok := jsonMap["assign_gscode"].(bool); ok {
		assignGsCode = v
	}
	const copyCount = 1

	if strings.TrimSpace(serial) == "" {
		s.Utils.SendError(c, errors.New("serial bo'sh"), "LinesYigishV2RePrint", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesYigishV2RePrint", "")
		return
	}

	productID, err := s.Store.Repo().LinesGetProductIdBySerial(YigishLineID, serial)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.Utils.SendError(c, errors.New("ushbu serial Yi'g'ish liniyasida chiqmagan"), "LinesYigishV2RePrint", "")
			return
		}
		s.Utils.SendError(c, err, "LinesYigishV2RePrint: LinesGetProductIdBySerial", "")
		return
	}
	if productID == 0 {
		s.Utils.SendError(c, errors.New("ushbu serial Yi'g'ish liniyasida chiqmagan"), "LinesYigishV2RePrint", "")
		return
	}

	selectedModel, err := s.linesYigishV2ModelByProduct(productID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2RePrint: model", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2RePrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != YigishLineID {
		s.Utils.SendError(c, errors.New("printer Yi'g'ish liniyasiga tegishli emas"), "LinesYigishV2RePrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesYigishV2RePrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2RePrint: LabelTemplateGetByID", "")
		return
	}

	gsCode, assignedGsID, err := s.linesYigishV2ResolveGSCodeForReprint(
		serial,
		template,
		selectedModel.ID,
		assignGsCode,
		userID,
	)
	if err != nil {
		s.Utils.SendError(c, err, "LinesYigishV2RePrint", "")
		return
	}

	if err := s.linesYigishV2ExecutePrint(printerV2, copyCount, serial, selectedModel, gsCode); err != nil {
		if assignedGsID > 0 {
			_ = s.Store.Repo().GsCodeUpdateUndo(assignedGsID)
		}
		s.Utils.SendError(c, err, "LinesYigishV2RePrint", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"model":  selectedModel,
		"serial": serial,
		"gscode": gsCode,
	})
}
