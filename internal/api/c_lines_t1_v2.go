package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
)

func (s *ServerModel) linesT1V2ExecutePrint(
	printerV2 models.PrinterV2,
	copyCount int,
	serial string,
	selectedModel models.ModelInfo,
	gsCode string,
) error {
	if printerV2.LineID != 4 {
		return errors.New("printer T1 liniyasiga tegishli emas")
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

func (s *ServerModel) LinesT1V2SerialPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: ReadBody", "")
		return
	}

	modelID := int(jsonMap["model_id"].(float64))
	copyCount := int(jsonMap["copy"].(float64))
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	compressorSerial, _ := jsonMap["compressor_serial"].(string)
	compressorSerial = strings.TrimSpace(compressorSerial)
	userID := c.GetInt("user_id")

	if modelID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesT1V2SerialPrint", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesT1V2SerialPrint", "")
		return
	}
	if compressorSerial == "" {
		s.Utils.SendError(c, errors.New("Kompressor nomer bo'sh"), "LinesT1V2SerialPrint", "")
		return
	}

	selectedModel, err := s.Store.Repo().ModelsGetByID(modelID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: ModelsGetByID", "")
		return
	}
	if selectedModel.ID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesT1V2SerialPrint", "")
		return
	}

	if err := s.enforceProductionPlan(4, selectedModel.ID, 0, 1); err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: plan", "")
		return
	}

	if !store.ModelCompressorSerialMatches(compressorSerial, selectedModel.Compressor_serial) {
		s.Utils.SendError(c, errors.New("Kompressor nomer mos emas"), "LinesT1V2SerialPrint", "")
		return
	}

	exists, err := s.Store.Repo().ProductParamsCompressorExists(compressorSerial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: ProductParamsCompressorExists", "")
		return
	}
	if exists {
		s.Utils.SendError(c, errors.New("bu kompressor nomer allaqachon kiritilgan"), "LinesT1V2SerialPrint", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: PrinterV2GetByID", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: LabelTemplateGetByID", "")
		return
	}

	needsGSCode, err := utils.LabelTemplateNeedsGSCode(template)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: LabelTemplateNeedsGSCode", "")
		return
	}

	count, err := s.Store.Repo().ModelsUpdateCount(modelID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: ModelsUpdateCount", "")
		return
	}

	fullSerial, err := s.GenerateSerial(selectedModel.Seriya_raqami, count)
	if err != nil {
		if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
			s.Utils.SendError(c, rollbackErr, "LinesT1V2SerialPrint: ModelsUpdateCountMinus", "")
			return
		}
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: GenerateSerial", "")
		return
	}

	gsCode := ""
	var productID, gsID int

	if needsGSCode {
		productID, err = s.Store.Repo().LinesAddProduct(4, 1, userID, selectedModel.ID, fullSerial, "")
		if err != nil {
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesT1V2SerialPrint: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesT1V2SerialPrint: LinesAddProduct", "")
			return
		}

		gsID, err = s.Store.Repo().GsCodeUpdate(modelID, productID, userID)
		if err != nil {
			_ = s.Store.Repo().LinesDeleteProduct(productID)
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesT1V2SerialPrint: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesT1V2SerialPrint: GsCodeUpdate", "")
			return
		}

		productInfo, err := s.Store.Repo().LinesGetInfoByProductId(productID)
		if err != nil {
			_ = s.Store.Repo().GsCodeUpdateUndo(gsID)
			_ = s.Store.Repo().LinesDeleteProduct(productID)
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesT1V2SerialPrint: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesT1V2SerialPrint: LinesGetInfoByProductId", "")
			return
		}
		gsCode = productInfo.GS1Data
	}

	if err := s.linesT1V2ExecutePrint(printerV2, copyCount, fullSerial, selectedModel, gsCode); err != nil {
		if needsGSCode {
			if gsID > 0 {
				_ = s.Store.Repo().GsCodeUpdateUndo(gsID)
			}
			if productID > 0 {
				_ = s.Store.Repo().LinesDeleteProduct(productID)
			}
		}
		if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
			s.Utils.SendError(c, rollbackErr, "LinesT1V2SerialPrint: ModelsUpdateCountMinus", "")
			return
		}
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint", "")
		return
	}

	err = s.Store.Repo().LinesT1InsertProduct(fullSerial, selectedModel.ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: LinesT1InsertProduct", "")
		return
	}

	if !needsGSCode {
		_, err = s.Store.Repo().LinesAddProduct(4, 1, userID, selectedModel.ID, fullSerial, "")
		if err != nil {
			s.Utils.SendError(c, err, "LinesT1V2SerialPrint: LinesAddProduct", "")
			return
		}
	}

	if err := s.Store.Repo().ProductParamsInsertT1(fullSerial, compressorSerial, selectedModel.ID, userID); err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialPrint: ProductParamsInsertT1", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"model":             selectedModel,
		"serial":            fullSerial,
		"compressor_serial": compressorSerial,
	})
}

func (s *ServerModel) linesT1V2ResolveGSCode(serial string, template models.LabelTemplate) (string, error) {
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

func (s *ServerModel) linesT1V2ModelBySerial(serial string) (models.ModelInfo, error) {
	modelShortInfo, err := s.Store.Repo().ModelsShortInfoBySerial(serial)
	if err != nil {
		return models.ModelInfo{}, errors.New("Serial Xato")
	}

	selectedModel, err := s.Store.Repo().ModelsGetByID(modelShortInfo.ModelId)
	if err != nil {
		return models.ModelInfo{}, err
	}
	if selectedModel.ID == 0 {
		return models.ModelInfo{}, errors.New("model not found")
	}
	return selectedModel, nil
}

func (s *ServerModel) LinesT1V2SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialRePrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	const copyCount = 1

	if serial == "" {
		s.Utils.SendError(c, errors.New("serial bo'sh"), "LinesT1V2SerialRePrint", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesT1V2SerialRePrint", "")
		return
	}

	isSerialExists, err := s.Store.Repo().ProductSerialCheck(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialRePrint: ProductSerialCheck", "")
		return
	}
	if !isSerialExists {
		s.Utils.SendError(c, errors.New("Serial Xato"), "LinesT1V2SerialRePrint: ProductSerialCheck", "")
		return
	}

	selectedModel, err := s.linesT1V2ModelBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialRePrint", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialRePrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != 4 {
		s.Utils.SendError(c, errors.New("printer T1 liniyasiga tegishli emas"), "LinesT1V2SerialRePrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesT1V2SerialRePrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialRePrint: LabelTemplateGetByID", "")
		return
	}

	gsCode, err := s.linesT1V2ResolveGSCode(serial, template)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialRePrint: GsCodeBySerial", "")
		return
	}

	if err := s.linesT1V2ExecutePrint(printerV2, copyCount, serial, selectedModel, gsCode); err != nil {
		s.Utils.SendError(c, err, "LinesT1V2SerialRePrint", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"model":  selectedModel,
		"serial": serial,
	})
}
