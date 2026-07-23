package api

import (
	"database/sql"
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/utils"
)

const ichkiV2LineID = 7

func (s *ServerModel) linesIchkiV2ExecutePrint(
	printerV2 models.PrinterV2,
	copyCount int,
	serial string,
	selectedModel models.ModelInfo,
	gsCode string,
) error {
	if printerV2.LineID != ichkiV2LineID {
		return errors.New("printer I1 liniyasiga tegishli emas")
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

func (s *ServerModel) linesIchkiV2ResolveGSCodeForReprint(
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

	productID, err := s.Store.Repo().LinesGetProductIdBySerial(ichkiV2LineID, serial)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", 0, errors.New("ushbu serial I1 liniyasida chiqmagan")
		}
		return "", 0, err
	}
	if productID == 0 {
		return "", 0, errors.New("ushbu serial I1 liniyasida chiqmagan")
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

func (s *ServerModel) LinesIchkiV2Print(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2Print: ReadBody", "")
		return
	}

	userID := c.GetInt("user_id")
	modelID := int(jsonMap["model_id"].(float64))
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	quantity := int(jsonMap["quantity"].(float64))

	if modelID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesIchkiV2Print", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesIchkiV2Print", "")
		return
	}
	if quantity < 1 {
		quantity = 1
	}

	selectedModel, err := s.Store.Repo().ModelsGetByID(modelID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2Print: ModelsGetByID", "")
		return
	}
	if selectedModel.ID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesIchkiV2Print", "")
		return
	}

	if err := s.enforceProductionPlan(ichkiV2LineID, selectedModel.ID, 0, quantity); err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2Print: plan", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2Print: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != ichkiV2LineID {
		s.Utils.SendError(c, errors.New("printer I1 liniyasiga tegishli emas"), "LinesIchkiV2Print", "")
		return
	}

	lastGSCode := ""
	lastSerial := ""
	for i := 0; i < quantity; i++ {
		count, err := s.Store.Repo().ModelsUpdateCount(modelID)
		if err != nil {
			s.Utils.SendError(c, err, "LinesIchkiV2Print: ModelsUpdateCount", "")
			return
		}

		fullSerial, err := s.GenerateSerial(selectedModel.Seriya_raqami, count)
		if err != nil {
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesIchkiV2Print: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesIchkiV2Print: GenerateSerial", "")
			return
		}

		productID, err := s.Store.Repo().LinesAddProduct(ichkiV2LineID, 1, userID, selectedModel.ID, fullSerial, "")
		if err != nil {
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesIchkiV2Print: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesIchkiV2Print: LinesAddProduct", "")
			return
		}

		gsID, err := s.Store.Repo().GsCodeUpdate(modelID, productID, userID)
		if err != nil {
			_ = s.Store.Repo().LinesDeleteProduct(productID)
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesIchkiV2Print: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesIchkiV2Print: GsCodeUpdate", "")
			return
		}

		productInfo, err := s.Store.Repo().LinesGetInfoByProductId(productID)
		if err != nil {
			_ = s.Store.Repo().GsCodeUpdateUndo(gsID)
			_ = s.Store.Repo().LinesDeleteProduct(productID)
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesIchkiV2Print: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesIchkiV2Print: LinesGetInfoByProductId", "")
			return
		}

		gsCode := productInfo.GS1Data
		if err := s.linesIchkiV2ExecutePrint(printerV2, 1, fullSerial, selectedModel, gsCode); err != nil {
			_ = s.Store.Repo().GsCodeUpdateUndo(gsID)
			_ = s.Store.Repo().LinesDeleteProduct(productID)
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(modelID); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesIchkiV2Print: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, err, "LinesIchkiV2Print", "")
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

func (s *ServerModel) LinesIchkiV2RePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2RePrint: ReadBody", "")
		return
	}

	userID := c.GetInt("user_id")
	serial, _ := jsonMap["serial"].(string)
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	assignGsCode := false
	if v, ok := jsonMap["assign_gscode"].(bool); ok {
		assignGsCode = v
	}
	const copyCount = 1

	if serial == "" {
		s.Utils.SendError(c, errors.New("serial bo'sh"), "LinesIchkiV2RePrint", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesIchkiV2RePrint", "")
		return
	}

	isSerialExists, err := s.Store.Repo().ProductSerialCheck(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2RePrint: ProductSerialCheck", "")
		return
	}
	if !isSerialExists {
		s.Utils.SendError(c, errors.New("Serial Xato"), "LinesIchkiV2RePrint: ProductSerialCheck", "")
		return
	}

	selectedModel, err := s.linesT1V2ModelBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2RePrint", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2RePrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != ichkiV2LineID {
		s.Utils.SendError(c, errors.New("printer I1 liniyasiga tegishli emas"), "LinesIchkiV2RePrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesIchkiV2RePrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2RePrint: LabelTemplateGetByID", "")
		return
	}

	gsCode, assignedGsID, err := s.linesIchkiV2ResolveGSCodeForReprint(
		serial,
		template,
		selectedModel.ID,
		assignGsCode,
		userID,
	)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2RePrint", "")
		return
	}

	if err := s.linesIchkiV2ExecutePrint(printerV2, copyCount, serial, selectedModel, gsCode); err != nil {
		if assignedGsID > 0 {
			_ = s.Store.Repo().GsCodeUpdateUndo(assignedGsID)
		}
		s.Utils.SendError(c, err, "LinesIchkiV2RePrint", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"model":  selectedModel,
		"serial": serial,
		"gscode": gsCode,
	})
}

func (s *ServerModel) LinesIchkiV2TestPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2TestPrint: ReadBody", "")
		return
	}

	modelID := int(jsonMap["model_id"].(float64))
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))

	if modelID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesIchkiV2TestPrint", "")
		return
	}
	if printerV2ID == 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesIchkiV2TestPrint", "")
		return
	}

	selectedModel, err := s.Store.Repo().ModelsGetByID(modelID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2TestPrint: ModelsGetByID", "")
		return
	}
	if selectedModel.ID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesIchkiV2TestPrint", "")
		return
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2TestPrint: PrinterV2GetByID", "")
		return
	}
	if printerV2.LineID != ichkiV2LineID {
		s.Utils.SendError(c, errors.New("printer I1 liniyasiga tegishli emas"), "LinesIchkiV2TestPrint", "")
		return
	}
	if printerV2.LabelTemplateID <= 0 {
		s.Utils.SendError(c, errors.New("etiketka shablon tanlanmagan"), "LinesIchkiV2TestPrint", "")
		return
	}

	serial, err := s.Store.Repo().LinesGetLastSerialByLineModel(ichkiV2LineID, modelID)
	if err != nil {
		s.Utils.SendError(c, errors.New("ushbu model uchun avval chop etilgan serial topilmadi"), "LinesIchkiV2TestPrint", "")
		return
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2TestPrint: LabelTemplateGetByID", "")
		return
	}

	gsCode, err := s.linesT1V2ResolveGSCode(serial, template)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2TestPrint: GsCodeBySerial", "")
		return
	}

	if err := s.linesIchkiV2ExecutePrint(printerV2, 1, serial, selectedModel, gsCode); err != nil {
		s.Utils.SendError(c, err, "LinesIchkiV2TestPrint", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"model":  selectedModel,
		"serial": serial,
		"gscode": gsCode,
	})
}
