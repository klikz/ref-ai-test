package api

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
)

func parseRequestIntSlice(raw any) []int {
	switch v := raw.(type) {
	case []any:
		out := make([]int, 0, len(v))
		seen := map[int]bool{}
		for _, item := range v {
			num, ok := item.(float64)
			if !ok {
				continue
			}
			id := int(num)
			if id <= 0 || seen[id] {
				continue
			}
			seen[id] = true
			out = append(out, id)
		}
		return out
	case float64:
		if int(v) > 0 {
			return []int{int(v)}
		}
	}
	return nil
}

type DashboardReport struct {
	T1               []models.ModelsCount `json:"t1"`
	T2               []models.ModelsCount `json:"t2"`
	T3               []models.ModelsCount `json:"t3"`
	ServerTime       string               `json:"server_time"`
	Plan             int                  `json:"plan"`
	Done             int                  `json:"done"`
	Progress         int                  `json:"progress"`
	ExpectedNow      int                  `json:"expected_now"`
	CycleTimeSeconds float64              `json:"cycle_time_seconds"`
	CycleTime        string               `json:"cycle_time"`
	IsBehindPlan     bool                 `json:"is_behind_plan"`
	PlanDiff         int                  `json:"plan_diff"`
	PlanStatusText   string               `json:"plan_status_text"`
	WorkTimeText     string               `json:"work_time_text"`
	CurrentShiftNo   int                  `json:"current_shift_no"`
	CurrentShiftLabel string              `json:"current_shift_label"`
	PlanDate         string               `json:"plan_date"`
}

func (s *ServerModel) LinesGetAll(c *gin.Context) {
	data, err := s.Store.Repo().LinesGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetAll", "")
		return
	}
	s.Utils.DebugLogAny("LinesGetAll: ", data)
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesGetGPComponents(c *gin.Context) {
	data, err := s.Store.Repo().LinesGetGPComponents()
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetGPComponents", "")
		return
	}
	s.Utils.DebugLogAny("LinesGetGPComponents: ", data)
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesGetGPComponentsDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddGpComponent: ReadBody", "")
		return
	}

	component_id := int(jsonMap["component_id"].(float64))
	line_id := int(jsonMap["line_id"].(float64))
	// model_id := int(jsonMap["model_id"].(float64))

	err = s.Store.Repo().LinesGpComponentDelete(line_id, component_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddGpComponent: LinesGpComponentDelete", "")
		return
	}

	s.Utils.SendOK(c, "ok")
}

func GetMonthValueAsString() string {
	monthValue := time.Now().Month()

	if monthValue <= 9 {
		return fmt.Sprintf("%d", monthValue)
	}

	alphabetChar := 'A' + rune(monthValue-10)
	return string(alphabetChar)
}

func (s *ServerModel) GenerateSerial(prefix string, count int) (string, error) {
	return utils.GenerateSerial(prefix, count), nil
}

func (s *ServerModel) LinesAddProduct(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddProduct: ReadBody", "")
		return
	}

	line_id := int(jsonMap["line_id"].(float64))
	gp_id := int(jsonMap["gp_id"].(float64))
	printer_id := int(jsonMap["printer_id"].(float64))
	user_id := c.GetInt("user_id")
	// serial := jsonMap["serial"].(string)

	s.Utils.DebugLogAny("line_id: ", line_id)
	s.Utils.DebugLogAny("gp_id: ", gp_id)

	printerInfo, err := s.Store.Repo().PrinterInfoById(printer_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddProduct: PrinterInfoById", "")
		return
	}

	s.Utils.DebugLogAny("printerInfo: ", printerInfo)

	_, err = s.Store.Repo().LinesIncreaseCount(gp_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddProduct: LinesIncreaseCount", "")
		return
	}

	gpComponent, err := s.Store.Repo().LinesGetGPComponentById(gp_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddProduct: LinesGetGPComponentById", "")
		return
	}
	s.Utils.DebugLogAny("gpComponent: ", gpComponent.ComponentID)
	s.Utils.DebugLogAny("gp_id: ", gp_id)

	serialData, err := s.GenerateSerial(gpComponent.Serial, gpComponent.Count)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddProduct: GenerateSerial", "")
		return
	}

	s.Utils.DebugLogAny("serialData: ", serialData)

	id, err := s.Store.Repo().LinesAddProduct(line_id, gpComponent.ComponentID, user_id, gpComponent.ID, serialData, "")
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddProduct", "")
		return
	}

	// folderName, err := s.Store.Repo().LinesGetFolderName(line_id)
	// if err != nil {
	// 	s.Utils.SendError(c, err, "LinesAddProduct: LinesGetFolderName", "")
	// 	return
	// }

	printData := []byte(fmt.Sprintf(`
	{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": 0,
		"serialNumbers": 0,
		"dataEntryControls": {
				"serialInput": "%s",
				"index1Input": "%s",
				"index2Input": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName, serialData, gpComponent.Index_1, gpComponent.Index_2))

	s.Utils.DebugLogAny("printData: ", string(printData))
	s.Utils.DebugLogAny("id: ", id)

	errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

	s.Utils.DebugLogAny("errString: ", errString)

	if errString != "ok" {
		err = s.Store.Repo().LinesDeleteProduct(id)
		if err != nil {
			s.Utils.SendError(c, err, "LinesAddProduct: LinesDeleteProduct", "")
			return
		}

		count, err := s.Store.Repo().LinesDecreaseCount(line_id, gpComponent.ComponentID)
		if err != nil {
			s.Utils.SendError(c, err, "LinesAddProduct: LinesDecreaseCount", "")
			return
		}

		s.Utils.DebugLogAny("deleted count: ", count)
		s.Utils.SendError(c, errors.New(errString), "LinesAddProduct", "")
		return
	}

	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) LinesRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRePrint: ReadBody", "")
		return
	}

	index_1 := jsonMap["index_1"].(string)
	index_2 := jsonMap["index_2"].(string)
	serial := jsonMap["serial"].(string)
	printer_id := int(jsonMap["printer_id"].(float64))

	printerInfo, err := s.Store.Repo().PrinterInfoById(printer_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRePrint: PrinterInfoById", "")
		return
	}

	printData := []byte(fmt.Sprintf(`
	{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": 0,
		"serialNumbers": 0,
		"dataEntryControls": {
				"serialInput": "%s",
				"index1Input": "%s",
				"index2Input": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName, serial, index_1, index_2))

	s.Utils.DebugLogAny("printData: ", string(printData))

	errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

	if errString != "ok" {
		s.Utils.SendError(c, errors.New(errString), "LinesRePrint", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) LinesGetLastComponents(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddProduct: ReadBody", "")
		return
	}

	line_id := int(jsonMap["line_id"].(float64))
	data, err := s.Store.Repo().LinesGetLastComponents(line_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetLastComponents", "")
		return
	}
	s.Utils.DebugLogAny("LinesGetLastComponents: ", data)
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesGetPrinters(c *gin.Context) {
	data, err := s.Store.Repo().LinesGetPrinters()
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetPrinters", "")
		return
	}
	s.Utils.DebugLogAny("LinesGetPrinters: ", data)
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesFinPressPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesFinPressPrint: ReadBody", "")
		return
	}

	count := jsonMap["count"].(float64)
	gp_id := int(jsonMap["gp_id"].(float64))
	component_id := int(jsonMap["component_id"].(float64))
	printer_id := int(jsonMap["printer_id"].(float64))

	s.Utils.DebugLogAny("count: ", count)
	s.Utils.DebugLogAny("gp_id: ", gp_id)
	s.Utils.DebugLogAny("component_id: ", component_id)
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	gpComponent, err := s.Store.Repo().LinesGetGPComponentById(gp_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesFinPressPrint: LinesGetGPComponentById", "")
		return
	}

	printerInfo, err := s.Store.Repo().PrinterInfoById(printer_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesFinPressPrint: PrinterInfoById", "")
		return
	}

	qrData := fmt.Sprintf("%d&%d", gpComponent.ID, int(count))

	printData := []byte(fmt.Sprintf(`
	{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": 0,
		"serialNumbers": 0,
		"dataEntryControls": {
				"index1Input": "%s",
				"index2Input": "%s"
				"inputCount": "%d",
				"inputData": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName, gpComponent.Index_1, gpComponent.Index_2, int(count), qrData))

	s.Utils.DebugLogAny("printData: ", string(printData))

	errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

	if errString != "ok" {
		s.Utils.SendError(c, errors.New(errString), "LinesFinPressPrint", "")
		return
	}

	finPressLineID, err := s.Store.Repo().LinesFinPressLineID()
	if err != nil {
		s.Utils.SendError(c, err, "LinesFinPressPrint: LinesFinPressLineID", "")
		return
	}

	err = s.Store.Repo().LinesBalanceUpdate(count, finPressLineID, component_id, c.GetInt("user_id"), "fin_press_print", "Fin press chop etish")
	if err != nil {
		s.Utils.SendError(c, err, "LinesFinPressPrint: LinesBalanceUpdate", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) LinesBalanceAdjust(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesBalanceAdjust: ReadBody", "")
		return
	}

	lineID := int(jsonMap["line_id"].(float64))
	componentID := int(jsonMap["component_id"].(float64))
	quantityChange := jsonMap["quantity_change"].(float64)
	comment := ""
	if rawComment, ok := jsonMap["comment"].(string); ok {
		comment = rawComment
	}

	quantityAfter, err := s.Store.Repo().LinesBalanceApplyChange(store.BalanceChangeParams{
		LineID:         lineID,
		ComponentID:    componentID,
		QuantityChange: quantityChange,
		UserID:         c.GetInt("user_id"),
		Source:         "manual",
		Comment:        comment,
	})
	if err != nil {
		s.Utils.SendError(c, err, "LinesBalanceAdjust", "")
		return
	}
	s.Utils.SendOK(c, map[string]float64{"quantity_after": quantityAfter})
}

func (s *ServerModel) LinesBalanceTransactions(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesBalanceTransactions: ReadBody", "")
		return
	}

	filter := store.BalanceTransactionFilter{}
	if rawLineID, ok := jsonMap["line_id"].(float64); ok {
		filter.LineID = int(rawLineID)
	}
	if rawComponentID, ok := jsonMap["component_id"].(float64); ok {
		filter.ComponentID = int(rawComponentID)
	}
	if rawDateFrom, ok := jsonMap["date_from"].(string); ok {
		filter.DateFrom = rawDateFrom
	}
	if rawDateTo, ok := jsonMap["date_to"].(string); ok {
		filter.DateTo = rawDateTo
	}

	data, err := s.Store.Repo().LinesBalanceTransactionsGet(filter)
	if err != nil {
		s.Utils.SendError(c, err, "LinesBalanceTransactions", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesGetBalance(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetBalance: ReadBody", "")
		return
	}
	line_id := int(jsonMap["line_id"].(float64))

	data, err := s.Store.Repo().LinesGetBalance(line_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetBalance", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesT1SerialPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetBalance: ReadBody", "")
		return
	}
	model_id := int(jsonMap["model_id"].(float64))
	s.Utils.DebugLogAny("model_id: ", model_id)

	copy := int(jsonMap["copy"].(float64))
	s.Utils.DebugLogAny("copy: ", copy)

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	user_id := c.GetInt("user_id")
	s.Utils.DebugLogAny("user_id: ", user_id)

	if model_id == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesT1PrintSerial", "")
		return
	}

	modelsAll, err := s.Store.Repo().ModelsGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1PrintSerial: ModelsGetAll", "")
		return
	}
	selectedModel := models.ModelInfo{}

	for _, model := range modelsAll {
		if model.ID == model_id {
			selectedModel = model
			break
		}
	}
	if selectedModel.ID == 0 {
		s.Utils.SendError(c, errors.New("model not found"), "LinesT1PrintSerial", "")
		return
	}

	printerInfo, err := s.Store.Repo().PrinterInfoById(printer_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1PrintSerial: PrinterInfoById", "")
		return
	}

	s.Utils.DebugLogAny("printer info: ", printerInfo)

	count, err := s.Store.Repo().ModelsUpdateCount(model_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1PrintSerial: ModelsUpdateCount", "")
		return
	}

	fullSerial, err := s.GenerateSerial(selectedModel.Seriya_raqami, count)
	if err != nil {
		if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(model_id); rollbackErr != nil {
			s.Utils.SendError(c, rollbackErr, "LinesT1PrintSerial: ModelsUpdateCountMinus", "")
			return
		}
		s.Utils.SendError(c, err, "LinesT1PrintSerial: GenerateSerial", "")
		return
	}

	printData := []byte(fmt.Sprintf(`
	{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": %d,
		"serialNumbers": 0,
		"dataEntryControls": {
				"serialInput": "%s",
				"modeliInput": "%s",
				"modelNomiInput": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName, copy, fullSerial, selectedModel.Modeli, selectedModel.Qisqa_nomi))

	s.Utils.DebugLogAny("printData: ", string(printData))

	if gin.Mode() == gin.ReleaseMode {
		errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)
		if errString != "ok" {
			if _, rollbackErr := s.Store.Repo().ModelsUpdateCountMinus(model_id); rollbackErr != nil {
				s.Utils.SendError(c, rollbackErr, "LinesT1PrintSerial: ModelsUpdateCountMinus", "")
				return
			}
			s.Utils.SendError(c, errors.New(errString), "LinesT1PrintSerial", "")
			return
		}
	}

	s.Utils.DebugLogAny("fullSerial: ", fullSerial)

	err = s.Store.Repo().LinesT1InsertProduct(fullSerial, selectedModel.ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1PrintSerial: LinesT1InsertProduct", "")
		return
	}

	_, err = s.Store.Repo().LinesAddProduct(4, 1, user_id, selectedModel.ID, fullSerial, "")
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1PrintSerial: LinesAddProduct", "")
		return
	}

	s.Utils.SendOK(c, selectedModel)
}

func (s *ServerModel) LinesT1SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1SerialRePrint: ReadBody", "")
		return
	}

	serial := jsonMap["serial"].(string)
	s.Utils.DebugLogAny("serial: ", serial)

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	isSerialExists, err := s.Store.Repo().ProductSerialCheck(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1SerialRePrint: ProductSerialCheck", "")
		return
	}
	if !isSerialExists {
		s.Utils.SendError(c, errors.New("Serial Xato"), "LinesT1SerialRePrint: ProductSerialCheck", "")
		return
	}

	modelShortInfo, err := s.Store.Repo().ModelsShortInfoBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, errors.New("Serial Xato"), "LinesT1SerialRePrint: ModelsShortInfoBySerial", "")
		return
	}

	selectedModel := models.ModelInfo{}

	allModels, err := s.Utils.Store.Repo().ModelsGetAll()

	for _, d := range allModels {
		if d.ID == modelShortInfo.ModelId {
			selectedModel = d
			break
		}
	}

	printerInfo, err := s.Store.Repo().PrinterInfoById(printer_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT1SerialRePrint: PrinterInfoById", "")
		return
	}
	s.Utils.DebugLogAny("printer info: ", printerInfo)

	printData := []byte(fmt.Sprintf(`
	{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": 0,
		"serialNumbers": 0,
		"dataEntryControls": {
				"serialInput": "%s",
				"modeliInput": "%s",
				"modelNomiInput": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName, serial, selectedModel.Modeli, selectedModel.Qisqa_nomi))

	s.Utils.DebugLogAny("printData: ", string(printData))

	if gin.Mode() == gin.ReleaseMode {
		errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

		if errString != "ok" {
			s.Utils.SendError(c, errors.New(errString), "LinesT1SerialRePrint", "")
			return
		}
	}

	s.Utils.SendOK(c, selectedModel)

}

func (s *ServerModel) LinesT1Last(c *gin.Context) {
	data, err := s.Store.Repo().LinesGetT1Last()
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetT1Last", "")
		return
	}
	s.Utils.DebugLogAny("LinesGetT1Last: ", data)
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) linesT2ExecutePrint(printerID int, serial string, selectedModel models.ModelInfo) error {
	printerInfo, err := s.Store.Repo().PrinterInfoById(printerID)
	if err != nil {
		return err
	}

	s.Utils.DebugLogAny("printer info: ", printerInfo)

	printData := []byte(fmt.Sprintf(`
	{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"dataEntryControls": {
				"serialInput": "%s",
				"modeliInput": "%s",
				"modelNomiInput": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName, serial, selectedModel.Modeli, selectedModel.Qisqa_nomi))

	s.Utils.ProdLogAny("printData: ", string(printData))

	if gin.Mode() == gin.ReleaseMode {
		errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

		if errString != "ok" {
			return errors.New(errString)
		}
	}

	return nil
}

func (s *ServerModel) LinesT2SerialPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2SerialPrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	s.Utils.DebugLogAny("serial: ", serial)

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	user_id := c.GetInt("user_id")
	s.Utils.DebugLogAny("user_id: ", user_id)

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT2SerialPrint: serial", "")
		return
	}

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2SerialPrint: ProductSerialCheck", "")
		return
	}

	productID, err := s.Store.Repo().LinesAddProduct(5, 0, user_id, selectedModel.ID, serial, "")
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2SerialPrint: ProductInsert", "")
		return
	}

	if err := s.linesT2ExecutePrint(printer_id, serial, selectedModel); err != nil {
		if deleteErr := s.Store.Repo().LinesDeleteProduct(productID); deleteErr != nil {
			s.Utils.SendError(c, deleteErr, "LinesT2SerialPrint: LinesDeleteProduct", "")
			return
		}
		s.Utils.SendError(c, err, "LinesT2SerialPrint", "")
		return
	}

	s.Utils.SendOK(c, selectedModel)
}

func (s *ServerModel) LinesT2SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2SerialRePrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	s.Utils.DebugLogAny("serial: ", serial)

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT2SerialRePrint: serial", "")
		return
	}

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT2SerialRePrint: ProductSerialCheck", "")
		return
	}

	if err := s.linesT2ExecutePrint(printer_id, serial, selectedModel); err != nil {
		s.Utils.SendError(c, err, "LinesT2SerialRePrint", "")
		return
	}

	s.Utils.SendOK(c, selectedModel)
}

func (s *ServerModel) LinesLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetBalance: ReadBody", "")
		return
	}
	line_id := int(jsonMap["line_id"].(float64))
	s.Utils.DebugLogAny("line_id: ", line_id)

	data, err := s.Store.Repo().LinesGetLast(line_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesGetLast", "")
		return
	}
	s.Utils.DebugLogAny("LinesGetLast: ", data)
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesReport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesReport: ReadBody", "")
		return
	}
	var lineIDs []int
	if val, exists := jsonMap["line_ids"]; exists && val != nil {
		lineIDs = parseRequestIntSlice(val)
	}
	if len(lineIDs) == 0 {
		if val, exists := jsonMap["line_id"]; exists && val != nil {
			if num, ok := val.(float64); ok && int(num) > 0 {
				lineIDs = []int{int(num)}
			}
		}
	}
	s.Utils.DebugLogAny("line_ids: ", lineIDs)

	var modelIDs []int
	if val, exists := jsonMap["model_ids"]; exists && val != nil {
		modelIDs = parseRequestIntSlice(val)
	}
	if len(modelIDs) == 0 {
		if val, exists := jsonMap["model_id"]; exists && val != nil {
			if num, ok := val.(float64); ok && int(num) > 0 {
				modelIDs = []int{int(num)}
			}
		}
	}
	s.Utils.DebugLogAny("model_ids: ", modelIDs)

	var date1 string
	if val, exists := jsonMap["date1"]; exists && val != nil {
		if date, ok := val.(string); ok {
			date1 = date
		}
	}
	s.Utils.DebugLogAny("date1: ", date1)

	var date2 string
	if val, exists := jsonMap["date2"]; exists && val != nil {
		if date, ok := val.(string); ok {
			date2 = date
		}
	}
	s.Utils.DebugLogAny("date2: ", date2)

	exportAll := false
	if val, exists := jsonMap["export_all"]; exists {
		if b, ok := val.(bool); ok {
			exportAll = b
		}
	}

	page := 1
	pageSize := 500
	if exportAll {
		pageSize = 0
	} else {
		if val, exists := jsonMap["page"]; exists {
			if num, ok := val.(float64); ok && int(num) > 0 {
				page = int(num)
			}
		}
		if val, exists := jsonMap["page_size"]; exists {
			if num, ok := val.(float64); ok && int(num) > 0 {
				pageSize = int(num)
				if pageSize > 5000 {
					pageSize = 5000
				}
			}
		}
	}

	type LineReports struct {
		Count      int                  `json:"count"`
		ShortTable []models.ModelsCount `json:"short_table"`
		Detailed   []models.LReport     `json:"detailed"`
		Page       int                  `json:"page"`
		PageSize   int                  `json:"page_size"`
		TotalPages int                  `json:"total_pages"`
	}
	data := LineReports{Page: page, PageSize: pageSize}

	total, err := s.Store.Repo().LinesReportCount(lineIDs, modelIDs, date1, date2)
	if err != nil {
		s.Utils.SendError(c, err, "LinesReportCount", "")
		return
	}
	data.Count = total

	offset := 0
	if pageSize > 0 {
		offset = (page - 1) * pageSize
		data.TotalPages = (total + pageSize - 1) / pageSize
	} else {
		data.TotalPages = 1
	}

	lreport, err := s.Store.Repo().LinesReport(lineIDs, modelIDs, date1, date2, pageSize, offset)
	if err != nil {
		s.Utils.SendError(c, err, "LinesReport", "")
		return
	}
	modelsCount, err := s.Utils.Store.Repo().ProductionCountModels(date1, date2, lineIDs, modelIDs)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionCountModels", "")
		return
	}
	data.Detailed = lreport
	data.ShortTable = modelsCount
	s.Utils.SendOK(c, data)
}

func filterAuxiliaryLineIDs(lineIDs []int) []int {
	if len(lineIDs) == 0 {
		return nil
	}
	filtered := make([]int, 0, len(lineIDs))
	for _, id := range lineIDs {
		if id == store.FinPressLineID || id == store.RadiatorLineID || id == store.KlapanLineID || id == store.EshikLineID {
			filtered = append(filtered, id)
		}
	}
	return filtered
}

func (s *ServerModel) LinesAuxiliaryReport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAuxiliaryReport: ReadBody", "")
		return
	}

	lineIDs := filterAuxiliaryLineIDs(parseRequestLineIDs(jsonMap))

	var date1, date2 string
	if val, exists := jsonMap["date1"]; exists && val != nil {
		if date, ok := val.(string); ok {
			date1 = date
		}
	}
	if val, exists := jsonMap["date2"]; exists && val != nil {
		if date, ok := val.(string); ok {
			date2 = date
		}
	}

	exportAll := false
	if val, exists := jsonMap["export_all"]; exists {
		if b, ok := val.(bool); ok {
			exportAll = b
		}
	}

	page := 1
	pageSize := 500
	if exportAll {
		pageSize = 0
	} else {
		if val, exists := jsonMap["page"]; exists {
			if num, ok := val.(float64); ok && int(num) > 0 {
				page = int(num)
			}
		}
		if val, exists := jsonMap["page_size"]; exists {
			if num, ok := val.(float64); ok && int(num) > 0 {
				pageSize = int(num)
				if pageSize > 5000 {
					pageSize = 5000
				}
			}
		}
	}

	type AuxiliaryReportResponse struct {
		Count      int                              `json:"count"`
		ShortTable []store.AuxiliaryReportSummaryRow `json:"short_table"`
		Detailed   []store.AuxiliaryReportDetailRow  `json:"detailed"`
		Page       int                              `json:"page"`
		PageSize   int                              `json:"page_size"`
		TotalPages int                              `json:"total_pages"`
	}

	data := AuxiliaryReportResponse{Page: page, PageSize: pageSize}

	total, err := s.Store.Repo().AuxiliaryReportCount(date1, date2, lineIDs)
	if err != nil {
		s.Utils.SendError(c, err, "AuxiliaryReportCount", "")
		return
	}
	data.Count = total

	offset := 0
	if pageSize > 0 {
		offset = (page - 1) * pageSize
		data.TotalPages = (total + pageSize - 1) / pageSize
	} else {
		data.TotalPages = 1
	}

	summary, err := s.Store.Repo().AuxiliaryReportSummary(date1, date2, lineIDs)
	if err != nil {
		s.Utils.SendError(c, err, "AuxiliaryReportSummary", "")
		return
	}
	data.ShortTable = summary

	detailed, err := s.Store.Repo().AuxiliaryReportDetail(date1, date2, lineIDs, pageSize, offset)
	if err != nil {
		s.Utils.SendError(c, err, "AuxiliaryReportDetail", "")
		return
	}
	data.Detailed = detailed

	s.Utils.SendOK(c, data)
}

func parseRequestLineIDs(jsonMap map[string]interface{}) []int {
	var lineIDs []int
	if val, exists := jsonMap["line_ids"]; exists && val != nil {
		lineIDs = parseRequestIntSlice(val)
	}
	if len(lineIDs) == 0 {
		if val, exists := jsonMap["line_id"]; exists && val != nil {
			if num, ok := val.(float64); ok && int(num) > 0 {
				lineIDs = []int{int(num)}
			}
		}
	}
	return lineIDs
}

func (s *ServerModel) linesT3SelectedModel(serial string) (models.ModelInfo, error) {
	isSerialExists, err := s.Store.Repo().ProductSerialCheck(serial)
	if err != nil {
		return models.ModelInfo{}, err
	}
	if !isSerialExists {
		return models.ModelInfo{}, errors.New("Serial Xato")
	}

	modelShortInfo, err := s.Store.Repo().ModelsShortInfoBySerial(serial)
	if err != nil {
		return models.ModelInfo{}, errors.New("Serial Xato")
	}

	allModels, err := s.Store.Repo().ModelsGetAll()
	if err != nil {
		return models.ModelInfo{}, err
	}

	for _, model := range allModels {
		if model.ID == modelShortInfo.ModelId {
			return model, nil
		}
	}

	return models.ModelInfo{}, errors.New("model not found")
}

func (s *ServerModel) linesT3ExecutePrint(printerID int, serial string, selectedModel models.ModelInfo) error {
	printerInfo, err := s.Store.Repo().PrinterInfoById(printerID)
	if err != nil {
		return err
	}

	s.Utils.DebugLogAny("selectedModel.Xladagent_miqdori_g: ", selectedModel.Xladagent_miqdori_g)

	printData := []byte(fmt.Sprintf(`
	{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": 1,
		"serialNumbers": 0,
		"dataEntryControls": {
				"agent_input": "%s",
				"brutto_input": "%s",
				"elektr_input": "%s",
				"gost_input": "%s",
				"istemol_input": "%s",
				"model_input": "%s",
				"model_short_input": "%s",
				"netto_input": "%s",
				"razmer_input": "%s",
				"reg_nomer_input": "%s",
				"serial_input": "%s",
				"shtrix_input": "%s",
				"sovutish_input": "%s",
				"hertz_input": "%s",
				"gramm_input": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName,
		selectedModel.Freon,
		selectedModel.Brutto,
		selectedModel.Nominal_tok_quvvati_w,
		selectedModel.GOST,
		selectedModel.Taminot_kuchlanishi_v,
		selectedModel.Modeli, selectedModel.Qisqa_nomi,
		selectedModel.Netto,
		selectedModel.Qadoq_hajmi,
		"",
		serial, selectedModel.GS1_EAN13,
		selectedModel.Umumiy_hajmi_l, selectedModel.Taminot_kuchlanishi_v, selectedModel.Xladagent_miqdori_g))

	s.Utils.DebugLogAny("printData: ", string(printData))

	if gin.Mode() == gin.ReleaseMode {
		errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

		if errString != "ok" {
			return errors.New(errString)
		}
	}

	return nil
}

func (s *ServerModel) LinesT3SerialPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3SerialPrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	accSerial, _ := jsonMap["acc_serial"].(string)
	accSerial = strings.TrimSpace(accSerial)
	s.Utils.DebugLogAny("serial: ", serial)
	s.Utils.DebugLogAny("accSerial: ", accSerial)

	user_id := c.GetInt("user_id")

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT3SerialPrint: serial", "")
		return
	}

	s.CurresntT3Serial = serial

	modelShortInfo, err := s.Store.Repo().ModelsShortInfoBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, errors.New("Serial Xato"), "LinesT3SerialPrint: ModelsShortInfoBySerial", "")
		return
	}

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3SerialPrint: ProductSerialCheck", "")
		return
	}

	if accSerial == "" {
		s.Utils.SendError(c, errors.New("Aksessuar nomer bo'sh"), "LinesT3SerialPrint: acc_serial", "")
		return
	}

	// Skanerlangan aksessuar nomer modeldagi prefixlardan biri bilan boshlanishi kerak.
	// Uzunroq bo'lishi mumkin, lekin boshlanishi mos kelishi shart.
	if !store.ModelAccSerialMatches(accSerial, modelShortInfo.AccSerial) {
		s.Utils.SendError(c, errors.New("Aksessuar nomer mos emas"), "LinesT3SerialPrint: acc_serial", "")
		return
	}
	
	productID, err := s.Store.Repo().LinesAddProduct(6, 0, user_id, selectedModel.ID, serial, accSerial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3SerialPrint: ProductInsert", "")
		return
	}

	if err := s.linesT3ExecutePrint(printer_id, serial, selectedModel); err != nil {
		if deleteErr := s.Store.Repo().LinesDeleteProduct(productID); deleteErr != nil {
			s.Utils.SendError(c, deleteErr, "LinesT3SerialPrint: LinesDeleteProduct", "")
			return
		}
		s.Utils.SendError(c, err, "LinesT3SerialPrint", "")
		return
	}

	s.Utils.SendOK(c, selectedModel)
}

func (s *ServerModel) LinesT3SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3SerialRePrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	s.Utils.DebugLogAny("serial: ", serial)

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	if serial == "" {
		s.Utils.SendError(c, errors.New("Serial nomer bo'sh"), "LinesT3SerialRePrint: serial", "")
		return
	}

	selectedModel, err := s.linesT3SelectedModel(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesT3SerialRePrint: ProductSerialCheck", "")
		return
	}

	if err := s.linesT3ExecutePrint(printer_id, serial, selectedModel); err != nil {
		s.Utils.SendError(c, err, "LinesT3SerialRePrint", "")
		return
	}

	s.Utils.SendOK(c, selectedModel)
}

func (s *ServerModel) LinesT3SerialCurrent(c *gin.Context) {
	s.Utils.SendOK(c, s.CurresntT3Serial)
}

func (s *ServerModel) LinesIchkiPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiPrint: ReadBody", "")
		return
	}

	user_id := c.GetInt("user_id")

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	quantity := int(jsonMap["quantity"].(float64))
	s.Utils.DebugLogAny("quantity: ", quantity)

	model_id := int(jsonMap["model_id"].(float64))
	s.Utils.DebugLogAny("model_id: ", model_id)

	selectedModel := models.ModelInfo{}
	gsCode := ""

	allModels, err := s.Utils.Store.Repo().ModelsGetAll()

	for _, model := range allModels {
		if model.ID == model_id {
			selectedModel = model
			break
		}
	}

	if selectedModel.ID == 0 {
		s.Utils.SendError(c, errors.New("Model not found"), "LinesIchkiPrint", "")
		return
	}

	printerInfo, err := s.Store.Repo().PrinterInfoById(printer_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiPrint: PrinterInfoById", "")
		return
	}

	s.Utils.DebugLogAny("printer info: ", printerInfo)

	for i := 0; i < quantity; i++ {

		count, err := s.Store.Repo().ModelsUpdateCount(model_id)
		if err != nil {
			s.Utils.SendError(c, err, "LinesIchkiPrint: ModelsUpdateCount", "")
			return
		}

		fullSerial, err := s.GenerateSerial(selectedModel.Seriya_raqami, count)

		product_id, err := s.Store.Repo().LinesAddProduct(7, 1, user_id, selectedModel.ID, fullSerial, "")
		if err != nil {
			s.Utils.SendError(c, err, "LinesIchkiPrint: LinesAddProduct", "")
			return
		}
		s.Utils.DebugLogAny("model_id: ", model_id)
		s.Utils.DebugLogAny("product_id: ", product_id)
		s.Utils.DebugLogAny("user_id: ", user_id)
		id, err := s.Store.Repo().GsCodeUpdate(model_id, product_id, user_id)
		if err != nil {
			s.Utils.Store.Repo().ModelsUpdateCountMinus(model_id)
			s.Store.Repo().LinesDeleteProduct(product_id)
			s.Utils.SendError(c, err, "LinesIchkiPrint: GsCodeUpdate", "")
			return
		}

		s.Utils.DebugLogAny("GSCode Id: ", id)

		productInfo, err := s.Store.Repo().LinesGetInfoByProductId(product_id)
		if err != nil {
			s.Utils.SendError(c, err, "LinesIchkiPrint: LinesGetInfoByProductId", "")
			return
		}

		gsCode = productInfo.GS1Data

		file, err := os.OpenFile("gscode.csv", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			s.Utils.SendError(c, err, "LinesIchkiPrint: OpenFile", "")
			return
		}

		_, err = file.WriteString(productInfo.GS1Data)
		if err != nil {
			s.Utils.SendError(c, err, "LinesIchkiPrint: WriteString", "")
			file.Close()
			return
		}
		file.Close()

		printData := []byte(fmt.Sprintf(`{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": 1,
		"serialNumbers": 0,
		"dataEntryControls": {
			"bruttoInput": "%s",
			"chastotaInput": "%s",
			"gostInput": "%s",
			"gs1EanInput": "%s",
			"hajmiInput": "%s",
			"korxonaNomiInput": "%s",
			"istemolQuvvatiInput": "%s",
			"mamlakatInput": "%s",
			"manzilInput": "%s",
			"modeliInput": "%s",
			"modelNomiInput": "%s",
			"rangiInput": "%s",
			"regNomerInput": "%s",
			"serialInput": "%s",
			"sovutishQuvvatiInput": "%s",
			"tokQuvvatiInput": "%s",
			"brendInput": "%s"
			}
		}`, printerInfo.FolderName, printerInfo.PrinterName,
			selectedModel.Brutto,
			selectedModel.Taminot_kuchlanishi_v,
			selectedModel.GOST,
			selectedModel.GS1_EAN13,
			selectedModel.Qadoq_hajmi,
			selectedModel.Korxon_nomi,
			selectedModel.Nominal_tok_quvvati_w,
			selectedModel.Ishlab_chiqaruvchi_mamlakat,
			selectedModel.Manzil,
			selectedModel.Modeli,
			selectedModel.Qisqa_nomi,
			selectedModel.Rangi,
			"",
			fullSerial,
			selectedModel.Umumiy_hajmi_l,
			selectedModel.Nominal_tok_quvvati_w,
			selectedModel.Brend))

		s.Utils.DebugLogAny("printData: ", string(printData))

		if gin.Mode() == gin.ReleaseMode {
			errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

			if errString != "ok" {
				err = s.Store.Repo().GsCodeUpdateUndo(id)
				if err != nil {
					s.Utils.SendError(c, err, "LinesIchkiPrint: GsCodeUpdateUndo", "")
					return
				}
				err = s.Store.Repo().LinesDeleteProduct(product_id)
				if err != nil {
					s.Utils.SendError(c, err, "LinesIchkiPrint: LinesDeleteProduct", "")
					return
				}
				_, err = s.Utils.Store.Repo().ModelsUpdateCountMinus(model_id)
				if err != nil {
					s.Utils.SendError(c, err, "LinesIchkiPrint: ModelsUpdateCountMinus", "")
					return
				}
				fmt.Println("Print error:", errString)
				s.Utils.SendError(c, errors.New(errString), "LinesIchkiPrint", "")
				return
			}

		}

		// s.Utils.DebugLogAny("fullSerial: ", productInfo.GS1Data)

	}

	s.Utils.SendOK(c, gsCode)

}

func (s *ServerModel) LinesIchkiRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiRePrint: ReadBody", "")
		return
	}

	// user_id := c.GetInt("user_id")

	printer_id := int(jsonMap["printer_id"].(float64))
	s.Utils.DebugLogAny("printer_id: ", printer_id)

	serial := jsonMap["serial"].(string)
	s.Utils.DebugLogAny("serial: ", serial)

	isSerialExists, err := s.Store.Repo().ProductSerialCheck(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiRePrint: ProductSerialCheck", "")
		return
	}

	if !isSerialExists {
		s.Utils.SendError(c, errors.New("Serial not found"), "LinesIchkiRePrint", "")
		return
	}

	// gsCode := ""

	info, err := s.Store.Repo().ModelsShortInfoBySerial(serial)
	selectedModel := models.ModelInfo{}
	allModels, err := s.Utils.Store.Repo().ModelsGetAll()

	for _, model := range allModels {
		if model.ID == info.ModelId {
			selectedModel = model
			break
		}
	}

	if selectedModel.ID == 0 {
		s.Utils.SendError(c, errors.New("Model not found"), "LinesIchkiPrint", "")
		return
	}

	printerInfo, err := s.Store.Repo().PrinterInfoById(printer_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiPrint: PrinterInfoById", "")
		return
	}

	s.Utils.DebugLogAny("printer info: ", printerInfo)

	product_id, err := s.Store.Repo().LinesGetProductIdBySerial(7, serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiPrint: LinesGetProductIdBySerial", "")
		return
	}

	s.Utils.DebugLogAny("product id: ", product_id)

	productInfo, err := s.Store.Repo().LinesGetInfoByProductId(product_id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiPrint: LinesGetInfoByProductId", "")
		return
	}

	// gsCode = productInfo.GS1Data

	file, err := os.OpenFile("gscode.csv", os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiPrint: OpenFile", "")
		return
	}

	_, err = file.WriteString(productInfo.GS1Data)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiPrint: WriteString", "")
		file.Close()
		return
	}
	file.Close()

	printData := []byte(fmt.Sprintf(`{
		"libraryID": "986278f7-755f-4412-940f-a89e893947de",
		"absolutePath": "C:/inetpub/wwwroot/BarTender/wwwroot/Templates/Premier/%s/label.btw",
		"printRequestID": "fe80480e-1f94-4A2f-8947-e492800623aa",
		"printer": "%s",
		"startingPosition": 0,
		"copies": 1,
		"serialNumbers": 0,
		"dataEntryControls": {
			"bruttoInput": "%s",
			"chastotaInput": "%s",
			"gostInput": "%s",
			"gs1EanInput": "%s",
			"hajmiInput": "%s",
			"korxonaNomiInput": "%s",
			"istemolQuvvatiInput": "%s",
			"mamlakatInput": "%s",
			"manzilInput": "%s",
			"modeliInput": "%s",
			"modelNomiInput": "%s",
			"rangiInput": "%s",
			"regNomerInput": "%s",
			"serialInput": "%s",
			"sovutishQuvvatiInput": "%s",
			"tokQuvvatiInput": "%s",
			"brendInput": "%s"
			}
		}`, printerInfo.FolderName, printerInfo.PrinterName,
		selectedModel.Brutto,
		selectedModel.Taminot_kuchlanishi_v,
		selectedModel.GOST,
		selectedModel.GS1_EAN13,
		selectedModel.Qadoq_hajmi,
		selectedModel.Korxon_nomi,
		selectedModel.Nominal_tok_quvvati_w,
		selectedModel.Ishlab_chiqaruvchi_mamlakat,
		selectedModel.Manzil,
		selectedModel.Modeli,
		selectedModel.Qisqa_nomi,
		selectedModel.Rangi,
		"",
		serial,
		selectedModel.Umumiy_hajmi_l,
		selectedModel.Nominal_tok_quvvati_w,
		selectedModel.Brend))

	s.Utils.DebugLogAny("printData: ", string(printData))

	if gin.Mode() == gin.ReleaseMode {
		errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)

		if errString != "ok" {
			fmt.Println("Print error:", errString)
			s.Utils.SendError(c, errors.New(errString), "LinesIchkiPrint", "")
			return
		}

	}

	// s.Utils.DebugLogAny("fullSerial: ", productInfo.GS1Data)

	s.Utils.SendOK(c, productInfo.GS1Data)

}

func (s *ServerModel) LinesDashboard(c *gin.Context) {
	now := time.Now()
	current, err := s.Store.Repo().ProductionCurrentShift(now)
	if err != nil {
		s.Utils.SendError(c, err, "LinesDashboard: ProductionCurrentShift", "")
		return
	}

	allData := DashboardReport{
		T1:                []models.ModelsCount{},
		T2:                []models.ModelsCount{},
		T3:                []models.ModelsCount{},
		ServerTime:        now.Format(time.RFC3339),
		CurrentShiftNo:    current.ShiftNo,
		CurrentShiftLabel: current.Label,
		PlanDate:          current.PlanDate,
	}

	if current.ShiftNo > 0 && current.PlanDate != "" {
		shiftStart, shiftEnd, err := s.Store.Repo().ProductionShiftWindow(current.PlanDate, current.ShiftNo)
		if err != nil {
			s.Utils.SendError(c, err, "LinesDashboard: ProductionShiftWindow", "")
			return
		}

		allCounts, err := s.Store.Repo().ProductionCountModelsByTimeRange(shiftStart, shiftEnd, []int{4, 5, 6}, []int{})
		if err != nil {
			s.Utils.SendError(c, err, "LinesDashboard: ProductionCountModelsByTimeRange", "")
			return
		}
		plan, err := s.Store.Repo().ProductionPlanLineShiftPlannedTotal(current.PlanDate, store.T3LineID, current.ShiftNo)
		if err != nil {
			s.Utils.SendError(c, err, "LinesDashboard: ProductionPlanLineShiftPlannedTotal", "")
			return
		}

		allData.Plan = plan
		allData.WorkTimeText = current.StartTime + " - " + current.EndTime
		for _, row := range allCounts {
			switch row.LineID {
			case 4:
				allData.T1 = append(allData.T1, row)
			case 5:
				allData.T2 = append(allData.T2, row)
			case 6:
				allData.T3 = append(allData.T3, row)
				allData.Done += row.Count
			}
		}
		fillDashboardShiftPlanMetrics(&allData, shiftStart, shiftEnd, now)
	} else {
		allData.WorkTimeText = "Smena vaqti emas"
		allData.PlanStatusText = "smena yo'q"
		allData.CycleTime = "00:00"
	}

	s.Utils.SendOK(c, allData)
}

func fillDashboardShiftPlanMetrics(data *DashboardReport, shiftStart, shiftEnd, now time.Time) {
	if data.Plan <= 0 {
		data.CycleTime = "00:00"
		data.PlanStatusText = "олдинда 0"
		return
	}

	shiftSeconds := int(shiftEnd.Sub(shiftStart).Seconds())
	if shiftSeconds <= 0 {
		data.CycleTime = "00:00"
		data.PlanStatusText = "олдинда 0"
		return
	}

	elapsedSeconds := 0
	if now.After(shiftStart) {
		effectiveEnd := now
		if effectiveEnd.After(shiftEnd) {
			effectiveEnd = shiftEnd
		}
		elapsedSeconds = int(effectiveEnd.Sub(shiftStart).Seconds())
	}

	data.CycleTimeSeconds = float64(shiftSeconds) / float64(data.Plan)
	if now.After(shiftEnd) || now.Equal(shiftEnd) {
		data.ExpectedNow = data.Plan
	} else {
		data.ExpectedNow = int(float64(elapsedSeconds) / data.CycleTimeSeconds)
		if data.ExpectedNow > data.Plan {
			data.ExpectedNow = data.Plan
		}
	}
	data.Progress = int(float64(data.Done) / float64(data.Plan) * 100)
	if data.Progress > 100 {
		data.Progress = 100
	}
	if data.Progress < 0 {
		data.Progress = 0
	}
	data.IsBehindPlan = data.Done < data.ExpectedNow
	if data.Done >= data.ExpectedNow {
		data.PlanDiff = data.Done - data.ExpectedNow
		data.PlanStatusText = fmt.Sprintf("олдинда %d", data.PlanDiff)
	} else {
		data.PlanDiff = data.ExpectedNow - data.Done
		data.PlanStatusText = fmt.Sprintf("ортда қолиш %d", data.PlanDiff)
	}
	data.CycleTime = formatDashboardCycleTime(data.CycleTimeSeconds)
}

func formatDashboardCycleTime(totalSeconds float64) string {
	if totalSeconds <= 0 {
		return "00:00"
	}
	roundedSeconds := int(totalSeconds + 0.5)
	minutes := roundedSeconds / 60
	seconds := roundedSeconds % 60
	return fmt.Sprintf("%02d:%02d", minutes, seconds)
}

func (s *ServerModel) LinesRejaGet(c *gin.Context) {
	count, err := s.Store.Repo().LinesRejaGet()
	if err != nil {
		s.Utils.SendError(c, err, "LinesRejaGet", "")
		return
	}
	s.Utils.SendOK(c, count)
}

func (s *ServerModel) LinesRejaUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesIchkiRePrint: ReadBody", "")
		return
	}

	// user_id := c.GetInt("user_id")

	count := int(jsonMap["count"].(float64))

	err = s.Store.Repo().LinesRejaUpdate(count)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRejaGet", "")
		return
	}
	s.Utils.SendOK(c, count)
}

func (s *ServerModel) LinesRadiatorReceive(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorReceive: ReadBody", "")
		return
	}

	rawScan, ok := jsonMap["scan"].(string)
	if !ok || rawScan == "" {
		if rawSessionID, ok := jsonMap["session_id"].(float64); ok && int64(rawSessionID) > 0 {
			rawScan = fmt.Sprintf("FP:%d", int64(rawSessionID))
		} else {
			s.Utils.SendError(c, errors.New("sessiya kodi kiritilmagan"), "LinesRadiatorReceive", "")
			return
		}
	}

	sessionID, err := store.ParseFinPressSessionScan(rawScan)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorReceive", "")
		return
	}

	radiatorLineID, err := s.Store.Repo().LinesRadiatorLineID()
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorReceive: LinesRadiatorLineID", "")
		return
	}

	data, err := s.Store.Repo().FinPressSessionReceive(sessionID, radiatorLineID, c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorReceive", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesRadiatorTransactionsLast(c *gin.Context) {
	radiatorLineID, err := s.Store.Repo().LinesRadiatorLineID()
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorTransactionsLast: LinesRadiatorLineID", "")
		return
	}

	data, err := s.Store.Repo().LinesBalanceTransactionsGet(store.BalanceTransactionFilter{
		LineID: radiatorLineID,
		Limit:  store.LastRecordsLimit,
	})
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorTransactionsLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesBrigadirItems(c *gin.Context) {
	data, err := s.Store.Repo().BrigadirDeliveryItemsList(c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "LinesBrigadirItems", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesBrigadirConfirm(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesBrigadirConfirm: ReadBody", "")
		return
	}

	itemID := int64(getProductionInt(jsonMap, "item_id", "id"))
	if err := s.Store.Repo().BrigadirDeliveryItemConfirm(itemID, c.GetInt("user_id")); err != nil {
		var insufficient store.WareInsufficientStockError
		if errors.As(err, &insufficient) {
			s.Utils.SendError(c, err, "LinesBrigadirConfirm", insufficient.Shortages)
			return
		}
		s.Utils.SendError(c, err, "LinesBrigadirConfirm", "")
		return
	}
	s.Utils.SendOK(c, map[string]bool{"is_received": true})
}

func (s *ServerModel) LinesBrigadirConfirmAll(c *gin.Context) {
	count, err := s.Store.Repo().BrigadirDeliveryItemsConfirmAll(c.GetInt("user_id"))
	if err != nil {
		var insufficient store.WareInsufficientStockError
		if errors.As(err, &insufficient) {
			s.Utils.SendError(c, err, "LinesBrigadirConfirmAll", insufficient.Shortages)
			return
		}
		s.Utils.SendError(c, err, "LinesBrigadirConfirmAll", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"confirmed_count": count})
}
