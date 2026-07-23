package api

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/utils"
	"github.com/xuri/excelize/v2"
)

type modelUploadRow struct {
	Row   int
	Model models.ModelInfo
}

type modelUploadConflict struct {
	Row        int    `json:"row"`
	Field      string `json:"field"`
	Value      string `json:"value"`
	ExistingID int    `json:"existing_id"`
	Message    string `json:"message"`
}

func (s *ServerModel) ModelsAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsAdd: ReadBody", "")
		return
	}

	file64 := jsonMap["file64"].(string)
	user_id := c.GetInt("user_id")

	data, err := utils.Base64Decode(file64)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsAdd, Base64Decode", "")
		return
	}

	err = os.WriteFile("models_upload.xlsx", []byte(data), 0644)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsAdd, WriteFile", "")
		return
	}

	f, err := excelize.OpenFile("models_upload.xlsx")
	if err != nil {
		s.Utils.SendError(c, err, "ModelsAdd, OpenFile", "")
		return
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			s.Utils.SendError(c, err, "ModelsAdd, CloseFile", "")
		}
	}()

	sh := f.GetSheetName(0)
	rows, err := f.GetRows(sh)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsAdd, GetRows", "")
		return
	}

	if allData, ok := modelsFromHeaderRows(rows); ok {
		conflicts, err := s.modelUploadConflicts(allData)
		if err != nil {
			s.Utils.SendError(c, err, "ModelsAdd: duplicate check", "")
			return
		}
		if len(conflicts) > 0 {
			s.Utils.SendError(c, errors.New("model exists but row has no id"), "ModelsAdd: duplicate rows", conflicts)
			return
		}

		inserted := 0
		updated := 0
		for _, item := range allData {
			if strings.TrimSpace(item.Model.Seriya_raqami) == "" && strings.TrimSpace(item.Model.Modeli) == "" {
				continue
			}
			action := "inserted"
			if item.Model.ID > 0 {
				action, err = s.Store.Repo().ModelsUpsert(item.Model, user_id)
			} else {
				err = s.Store.Repo().ModelsAdd(item.Model, user_id)
			}
			if err != nil {
				s.Utils.SendError(c, err, "ModelsAdd: upsert", item.Model)
				return
			}
			if action == "updated" {
				updated++
			} else {
				inserted++
			}
		}
		s.Utils.SendOK(c, map[string]int{"inserted": inserted, "updated": updated})
		return
	}

	var allData []modelUploadRow

	for i, row := range rows {
		if i == 0 {
			continue
		}
		if i == 1 && looksLikeModelFieldKeysRow(row) {
			continue
		}

		allData = append(allData, modelUploadRow{Row: i + 1, Model: modelFromListExportRow(row)})
	}

	// var file []models.ModelInfo

	// for i := 0; i < dataTemp.Cols.Len; i++ {
	// 	s.Utils.DebugLogAny(dataTemp.Col(i))
	// }

	// x, _ := xlsx.New(xlsx.WithInputFile("models_upload.xlsx"))
	// defer x.Close()

	// if err := x.Read(&file); err != nil {
	// 	s.Utils.SendError(c, err, "ModelsAdd, Read", "")
	// 	return
	// }

	conflicts, err := s.modelUploadConflicts(allData)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsAdd: duplicate check", "")
		return
	}
	if len(conflicts) > 0 {
		s.Utils.SendError(c, errors.New("model exists but row has no id"), "ModelsAdd: duplicate rows", conflicts)
		return
	}

	for _, v := range allData {
		if v.Model.Seriya_raqami == "" {
			continue
		}
		err := s.Store.Repo().ModelsAdd(v.Model, user_id)

		if err != nil {
			// s.Utils.SendError(c, err, "ModelsAdd", "")
			s.Utils.DebugLogAny("ModelsAdd error: ", err)
			continue
		}
	}

	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) modelUploadConflicts(rows []modelUploadRow) ([]modelUploadConflict, error) {
	conflicts := []modelUploadConflict{}
	for _, row := range rows {
		if row.Model.ID > 0 {
			continue
		}
		existingID, field, value, err := s.Store.Repo().ModelsFindDuplicateKey(row.Model)
		if err != nil {
			return nil, err
		}
		if existingID == 0 {
			continue
		}
		conflicts = append(conflicts, modelUploadConflict{
			Row:        row.Row,
			Field:      field,
			Value:      value,
			ExistingID: existingID,
			Message:    "id yo'q, lekin bazada shu ma'lumot bor",
		})
	}
	return conflicts, nil
}

func modelsFromHeaderRows(rows [][]string) ([]modelUploadRow, bool) {
	headerRowIndex, dataStartIndex, ok := resolveModelHeaderLayout(rows)
	if !ok || dataStartIndex >= len(rows) {
		return nil, false
	}

	headers := buildModelHeadersMap(rows[headerRowIndex])
	if !hasModelHeader(headers, "seriya_raqami") && !hasModelHeader(headers, "qisqa_nomi") && !hasModelHeader(headers, "modeli") {
		return nil, false
	}

	items := make([]modelUploadRow, 0, len(rows)-dataStartIndex)
	for rowIndex, row := range rows[dataStartIndex:] {
		payload := map[string]any{}
		for index, key := range headers {
			value := " "
			if index < len(row) && strings.TrimSpace(row[index]) != "" {
				value = strings.TrimSpace(row[index])
			}
			switch key {
			case "id":
				id, _ := strconv.Atoi(value)
				payload[key] = id
			case "status", "gscode_count":
				continue
			default:
				payload[key] = value
			}
		}

		raw, err := json.Marshal(payload)
		if err != nil {
			return nil, false
		}
		item := models.ModelInfo{}
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, false
		}
		items = append(items, modelUploadRow{Row: rowIndex + dataStartIndex + 1, Model: item})
	}
	return items, true
}

func resolveModelHeaderLayout(rows [][]string) (headerRowIndex int, dataStartIndex int, ok bool) {
	if len(rows) < 2 {
		return 0, 0, false
	}
	if len(rows) >= 3 && looksLikeModelFieldKeysRow(rows[1]) {
		return 1, 2, true
	}
	if looksLikeModelFieldKeysRow(rows[0]) {
		return 0, 1, true
	}
	headers := buildModelHeadersMap(rows[0])
	if hasModelHeader(headers, "seriya_raqami") || hasModelHeader(headers, "qisqa_nomi") || hasModelHeader(headers, "modeli") {
		return 0, 1, true
	}
	return 0, 0, false
}

func looksLikeModelFieldKeysRow(row []string) bool {
	for _, cell := range row {
		key := strings.TrimSpace(strings.ToLower(cell))
		switch key {
		case "id", "seriya_raqami", "acc_serial", "compressor_serial", "kompressor_serial", "qisqa_nomi", "modeli", "sovutgich_turi", "door_code":
			return true
		}
	}
	return false
}

func buildModelHeadersMap(row []string) map[int]string {
	headers := map[int]string{}
	for index, header := range row {
		key := normalizeModelHeaderKey(header)
		if key == "" {
			continue
		}
		headers[index] = key
	}
	return headers
}

func hasModelHeader(headers map[int]string, key string) bool {
	for _, header := range headers {
		if header == key {
			return true
		}
	}
	return false
}

func normalizeModelHeaderKey(header string) string {
	key := strings.TrimSpace(strings.ToLower(header))
	if key == "" {
		return ""
	}
	switch key {
	case "компрессор номер", "kompressor nomer", "компрессор_номер", "kompressor_nomer", "kompressor_serial":
		return "compressor_serial"
	case "аксессуар номер", "aksessuar nomer", "аксессуар_номер", "aksessuar_nomer", "acc serial":
		return "acc_serial"
	case "odoo code":
		return "odoo_code"
	case "gs1 ean13":
		return "gs1_ean13"
	case "sovutgich turi":
		return "sovutgich_turi"
	case "qisqa nomi":
		return "qisqa_nomi"
	case "ta'minot kuchlanishi (v)", "taminot kuchlanishi (v)":
		return "taminot_kuchlanishi_v"
	case "xladagent miqdori (g)":
		return "xladagent_miqdori_g"
	case "energiya samaradorlik sarfi (a+)":
		return "energiya_samaradorlik_sarfi"
	case "kompressor nomi":
		return "kompressor_nomi"
	case "local/export":
		return "local_export"
	case "elektr toki kuchlanishi va turi":
		return "elektr_toki_kuchlanishi_va_turi"
	case "yoritgich lampaning quvvati (vt)":
		return "yoritgich_lampaning_quvvati_vt"
	case "umumiy hajmi (l)":
		return "umumiy_hajmi_l"
	case "sovutgich kamera hajmi (l)":
		return "sovutgich_kamera_hajmi_l"
	case "muzlatgich kamera hajmi (l)":
		return "muzlatgich_kamera_hajmi_l"
	case "muzlatish quvvati", "muzlatish quvvati ":
		return "muzlatish_quvvati"
	case "nominal tok quvvati (w)":
		return "nominal_tok_quvvati_w"
	case "shovqin darajasi ichki blok db":
		return "shovqin_darajasi_db"
	}
	if strings.Contains(key, "_") {
		return key
	}
	return strings.ReplaceAll(key, " ", "_")
}

func modelCell(row []string, index int) string {
	if index < 0 || index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

// modelFromListExportRow — ustun tartibi /models export (modelExportFields) bilan mos.
func modelFromListExportRow(row []string) models.ModelInfo {
	model := models.ModelInfo{}
	model.ID, _ = strconv.Atoi(modelCell(row, 0))
	model.Seriya_raqami = modelCell(row, 1)
	model.Acc_serial = modelCell(row, 2)
	model.Modeli = modelCell(row, 3)
	model.Sovutgich_turi = modelCell(row, 4)
	model.Qisqa_nomi = modelCell(row, 5)
	model.Rangi = modelCell(row, 6)
	model.Sotuv_turi = modelCell(row, 7)
	model.GS1_EAN13 = modelCell(row, 8)
	model.GOST = modelCell(row, 9)
	model.Taminot_kuchlanishi_v = modelCell(row, 10)
	model.Xladagent_miqdori_g = modelCell(row, 11)
	model.Energiya_samaradorlik_sarfi = modelCell(row, 12)
	model.Kompressor_nomi = modelCell(row, 13)
	model.Maxalliy_sertifikat = modelCell(row, 14)
	model.EAC_Sertifikati = modelCell(row, 15)
	model.CE_Sertifikat = modelCell(row, 16)
	model.Ishlab_chiqaruvchi_mamlakat = modelCell(row, 17)
	model.Korxon_nomi = modelCell(row, 18)
	model.Manzil = modelCell(row, 19)
	model.Brend = modelCell(row, 20)
	model.Local_export = modelCell(row, 21)
	model.Netto = modelCell(row, 22)
	model.Brutto = modelCell(row, 23)
	model.Qadoq_hajmi = modelCell(row, 24)
	model.Mahsulot_hajmi = modelCell(row, 25)
	model.Iqlim_sharoitlari = modelCell(row, 26)
	model.Elektr_toki_kuchlanishi_va_turi = modelCell(row, 27)
	model.Yoritgich_lampaning_quvvati_vt = modelCell(row, 28)
	model.Umumiy_hajmi_l = modelCell(row, 29)
	model.Sovutgich_kamera_hajmi_l = modelCell(row, 30)
	model.Muzlatgich_kamera_hajmi_l = modelCell(row, 31)
	model.Muzlatish_quvvati = modelCell(row, 32)
	model.Nominal_tok_quvvati_w = modelCell(row, 33)
	model.Freon = modelCell(row, 34)
	model.Shovqin_darajasi_db = modelCell(row, 35)
	model.OdooCode = modelCell(row, 36)
	model.Door_code = modelCell(row, 37)
	model.Compressor_serial = modelCell(row, 38)
	model.Comment = modelCell(row, 39)
	return model
}
func (s *ServerModel) ModelsGetAll(c *gin.Context) {
	data, err := s.Store.Repo().ModelsGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "ModelsGetAll", "")
		return
	}
	// s.Utils.DebugLogAny("ModelsGetAll: ", data)
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ModelsUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsUpdate: ReadBody", "")
		return
	}

	user_id := c.GetInt("user_id")
	if modelPayload, ok := jsonMap["model"]; ok && modelPayload != nil {
		modelPayload = replaceNilWithSpace(modelPayload)
		modelBytes, err := json.Marshal(modelPayload)
		if err != nil {
			s.Utils.SendError(c, err, "ModelsUpdate: MarshalModel", "")
			return
		}
		model := models.ModelInfo{}
		if err := json.Unmarshal(modelBytes, &model); err != nil {
			s.Utils.SendError(c, err, "ModelsUpdate: UnmarshalModel", "")
			return
		}
		if model.ID == 0 {
			s.Utils.SendError(c, errors.New("model id is required"), "ModelsUpdate: model", "")
			return
		}
		if err := s.Store.Repo().ModelsUpdateWithStatus(model, user_id); err != nil {
			s.Utils.SendError(c, err, "ModelsUpdate: model", "")
			return
		}
		s.Utils.SendOK(c, "ok")
		return
	}

	file64, ok := jsonMap["file64"].(string)
	if !ok || file64 == "" {
		s.Utils.SendError(c, errors.New("file64 or model is required"), "ModelsUpdate", "")
		return
	}

	id := int(jsonMap["id"].(float64))

	data, err := utils.Base64Decode(file64)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsUpdate, Base64Decode", "")
		return
	}

	err = os.WriteFile("models_upload.xlsx", []byte(data), 0644)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsUpdate, WriteFile", "")
		return
	}

	f, err := excelize.OpenFile("models_upload.xlsx")
	if err != nil {
		s.Utils.SendError(c, err, "ModelsUpdate, OpenFile", "")
		return
	}
	defer func() {
		// Close the spreadsheet.
		if err := f.Close(); err != nil {
			s.Utils.SendError(c, err, "ModelsUpdate, CloseFile", "")
		}
	}()

	sh := f.GetSheetName(0)
	rows, err := f.GetRows(sh)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsUpdate, GetRows", "")
		return
	}

	if items, ok := modelsFromHeaderRows(rows); ok && len(items) > 0 {
		model := items[0].Model
		model.ID = id
		if err := s.Store.Repo().ModelsUpdate(model, user_id); err != nil {
			s.Utils.SendError(c, err, "ModelsUpdate: header row", model)
			return
		}
		s.Utils.SendOK(c, "ok")
		return
	}

	var allData []models.ModelInfo

	for i, row := range rows {
		if i == 0 {
			continue
		}

		// s.Utils.DebugLogAny("i: ", i)
		// s.Utils.DebugLogAny("row: ", len(row))
		model := models.ModelInfo{}

		model.ID = id
		model.Seriya_raqami = modelCell(row, 0)
		model.Acc_serial = modelCell(row, 1)
		model.Modeli = modelCell(row, 2)
		model.Sovutgich_turi = modelCell(row, 3)
		model.Qisqa_nomi = modelCell(row, 4)
		model.Rangi = modelCell(row, 5)
		model.Sotuv_turi = modelCell(row, 6)
		model.GS1_EAN13 = modelCell(row, 7)
		model.GOST = modelCell(row, 8)
		model.Taminot_kuchlanishi_v = modelCell(row, 9)
		model.Xladagent_miqdori_g = modelCell(row, 10)
		model.Energiya_samaradorlik_sarfi = modelCell(row, 11)
		model.Kompressor_nomi = modelCell(row, 12)
		model.Maxalliy_sertifikat = modelCell(row, 13)
		model.EAC_Sertifikati = modelCell(row, 14)
		model.CE_Sertifikat = modelCell(row, 15)
		model.Ishlab_chiqaruvchi_mamlakat = modelCell(row, 16)
		model.Korxon_nomi = modelCell(row, 17)
		model.Manzil = modelCell(row, 18)
		model.Brend = modelCell(row, 19)
		model.Local_export = modelCell(row, 20)
		model.Netto = modelCell(row, 21)
		model.Brutto = modelCell(row, 22)
		model.Qadoq_hajmi = modelCell(row, 23)
		model.Mahsulot_hajmi = modelCell(row, 24)
		model.Iqlim_sharoitlari = modelCell(row, 25)
		model.Elektr_toki_kuchlanishi_va_turi = modelCell(row, 26)
		model.Yoritgich_lampaning_quvvati_vt = modelCell(row, 27)
		model.Umumiy_hajmi_l = modelCell(row, 28)
		model.Sovutgich_kamera_hajmi_l = modelCell(row, 29)
		model.Muzlatgich_kamera_hajmi_l = modelCell(row, 30)
		model.Muzlatish_quvvati = modelCell(row, 31)
		model.Nominal_tok_quvvati_w = modelCell(row, 32)
		model.Freon = modelCell(row, 33)
		model.Shovqin_darajasi_db = modelCell(row, 34)
		model.OdooCode = modelCell(row, 35)
		model.Door_code = modelCell(row, 36)
		model.Compressor_serial = modelCell(row, 37)
		model.Comment = modelCell(row, 38)

		allData = append(allData, model)
	}

	s.Utils.DebugLogAny("allData: ", allData[0])

	err = s.Store.Repo().ModelsUpdate(allData[0], user_id)

	if err != nil {
		s.Utils.SendError(c, err, "ModelsUpdate", "")
		return
	}

	s.Utils.SendOK(c, "ok")
}

func replaceNilWithSpace(value any) any {
	switch data := value.(type) {
	case map[string]any:
		for key, item := range data {
			if item == nil {
				data[key] = " "
				continue
			}
			data[key] = replaceNilWithSpace(item)
		}
		return data
	case []any:
		for index, item := range data {
			if item == nil {
				data[index] = " "
				continue
			}
			data[index] = replaceNilWithSpace(item)
		}
		return data
	default:
		return value
	}
}

func (s *ServerModel) ModelsChangeStatus(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsChangeStatus: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	user_id := c.GetInt("user_id")

	err = s.Store.Repo().ModelsChangeStatus(id, user_id)
	if err != nil {
		s.Utils.SendError(c, err, "ModelsChangeStatus", "")
		return
	}

	s.Utils.SendOK(c, "ok")
}

func parseGsCodeUploadLines(raw string) []string {
	normalized := strings.ReplaceAll(raw, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	lines := strings.Split(normalized, "\n")
	result := make([]string, 0, len(lines))
	for index, line := range lines {
		line = strings.TrimSpace(line)
		if index == 0 {
			line = strings.TrimPrefix(line, "\ufeff")
		}
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func (s *ServerModel) GsCodeUpload(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "GsCodeUpload: ReadBody", "")
		return
	}
	file64 := getProductionString(jsonMap, "file64")
	if file64 == "" {
		s.Utils.SendError(c, errors.New("file64 is required"), "GsCodeUpload", "")
		return
	}
	user_id := c.GetInt("user_id")
	model_id := getProductionInt(jsonMap, "model_id")
	if model_id == 0 {
		s.Utils.SendError(c, errors.New("model_id is required"), "GsCodeUpload", "")
		return
	}

	data, err := utils.Base64Decode(file64)
	if err != nil {
		s.Utils.SendError(c, err, "GsCodeUpload, Base64Decode", "")
		return
	}

	stringArray := parseGsCodeUploadLines(data)
	if len(stringArray) == 0 {
		s.Utils.SendError(c, errors.New("faylda kodlar topilmadi"), "GsCodeUpload", nil)
		return
	}

	gs1EAN, err := s.Store.Repo().ModelsGetGS1ByID(model_id)
	if err != nil {
		s.Utils.SendError(c, err, "GsCodeUpload, ModelsGetGS1ByID", "")
		return
	}

	gscodeLen := len(gs1EAN)
	validCodes := make([]string, 0, len(stringArray))
	skippedShort := 0
	errGscode := ""

	for _, d := range stringArray {
		if len(d) < gscodeLen+3 {
			skippedShort++
			continue
		}
		if d[3:gscodeLen+3] != gs1EAN {
			errGscode = d[3 : gscodeLen+3]
			break
		}
		validCodes = append(validCodes, d)
	}

	if errGscode != "" {
		s.Utils.SendError(c, errors.New(errGscode+" - "+gs1EAN), "GsCodeUpload", errGscode)
		return
	}

	if len(validCodes) == 0 {
		s.Utils.SendError(c, errors.New("mos GS kodlar topilmadi. Model GS1: "+gs1EAN), "GsCodeUpload", map[string]any{
			"gs1_prefix":    gs1EAN,
			"skipped_short": skippedShort,
			"total_lines":   len(stringArray),
		})
		return
	}

	inserted, err := s.Store.Repo().GsCodesBulkAdd(validCodes, model_id, user_id)
	if err != nil {
		s.Utils.SendError(c, err, "GsCodeUpload, GsCodesBulkAdd", "")
		return
	}

	duplicates := len(validCodes) - inserted
	response := map[string]any{
		"inserted":    inserted,
		"duplicates":  duplicates,
		"valid":       len(validCodes),
		"total_lines": len(stringArray),
		"gs1_prefix":  gs1EAN,
	}
	if skippedShort > 0 {
		response["skipped_short"] = skippedShort
	}

	if inserted == 0 {
		s.Utils.SendError(c, errors.New("yangi kod qo'shilmadi — barcha kodlar allaqachon mavjud"), "GsCodeUpload", response)
		return
	}

	s.Utils.SendOK(c, response)
}

func (s *ServerModel) GsCodesGetCount(c *gin.Context) {
	data, err := s.Store.Repo().GsCodesGetCount()
	s.Utils.DebugLogAny(data)
	if err != nil {
		s.Utils.SendError(c, err, "GsCodesGetCount", "")
		return
	}
	s.Utils.SendOK(c, data)
}
func (s *ServerModel) GsCodesReport(c *gin.Context) {

	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "GsCodeUpload: ReadBody", "")
		return
	}

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

	data, err := s.Store.Repo().GsCodesReport(date1, date2)
	s.Utils.DebugLogAny(data)
	if err != nil {
		s.Utils.SendError(c, err, "GsCodesReport", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) LinesAddGpComponent(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddGpComponent: ReadBody", "")
		return
	}

	component_id := int(jsonMap["component_id"].(float64))
	line_id := int(jsonMap["line_id"].(float64))
	// model_id := int(jsonMap["model_id"].(float64))
	user_id := c.GetInt("user_id")
	index_1 := jsonMap["index_1"].(string)
	index_2 := jsonMap["index_2"].(string)
	serial := jsonMap["serial"].(string)

	err = s.Store.Repo().LinesAddGpComponent(component_id, line_id, user_id, index_1, index_2, serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddGpComponent", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) LinesAddPrinter(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddPrinter: ReadBody", "")
		return
	}

	line_id := int(jsonMap["line_id"].(float64))
	address := jsonMap["address"].(string)
	printer_name := jsonMap["printer_name"].(string)

	_, err = s.Store.Repo().PrintersAdd(line_id, address, printer_name)
	if err != nil {
		s.Utils.SendError(c, err, "LinesAddPrinter", "")
		return
	}

	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) LinesDeletePrinter(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesDeletePrinter: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	err = s.Store.Repo().PrintersDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "LinesDeletePrinter", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}
