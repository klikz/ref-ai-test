package api

import (
	"bytes"
	"database/sql"
	"errors"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
	"github.com/xuri/excelize/v2"
)

type consumptionNormUploadConflict struct {
	Row     int    `json:"row"`
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

type consumptionNormParsedRow struct {
	Row            int
	GroupLevel     int
	FactoryCode    string
	Quantity       float64
	ConsumeLine    string
	ReceiveLine    string
	ComponentID    int
	ConsumeLineID  int
	ReceiveLineID  int
}

func (s *ServerModel) ConsumptionNormModels(c *gin.Context) {
	data, err := s.Store.Repo().ConsumptionNormModelsSummary()
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormModels", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ConsumptionNormLines(c *gin.Context) {
	data, err := s.Store.Repo().LinesListForLookup()
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormLines", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ConsumptionNormAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormAdd: ReadBody", "")
		return
	}

	modelID := getProductionInt(jsonMap, "model_id", "id")
	componentID := getProductionInt(jsonMap, "component_id")
	parentItemID := getProductionInt(jsonMap, "parent_id", "parent_item_id")
	quantity := getProductionFloat(jsonMap, "quantity")
	consumeLineID := getProductionInt(jsonMap, "consume_line_id")
	receiveLineID := getProductionInt(jsonMap, "receive_line_id")

	if modelID == 0 {
		s.Utils.SendError(c, errors.New("model_id is required"), "ConsumptionNormAdd", "")
		return
	}
	if componentID == 0 {
		s.Utils.SendError(c, errors.New("component_id is required"), "ConsumptionNormAdd", "")
		return
	}
	if quantity <= 0 {
		s.Utils.SendError(c, errors.New("miqdor 0 dan katta bo'lishi kerak"), "ConsumptionNormAdd", "")
		return
	}

	ok, err := s.Store.Repo().ComponentExistsByID(componentID)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormAdd: ComponentExistsByID", "")
		return
	}
	if !ok {
		s.Utils.SendError(c, errors.New("komponent topilmadi"), "ConsumptionNormAdd", "")
		return
	}

	if parentItemID > 0 {
		parent, err := s.Store.Repo().ConsumptionNormItemByID(parentItemID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				s.Utils.SendError(c, errors.New("ota qator topilmadi"), "ConsumptionNormAdd", "")
				return
			}
			s.Utils.SendError(c, err, "ConsumptionNormAdd: parent", "")
			return
		}
		if parent.ModelID != modelID {
			s.Utils.SendError(c, errors.New("ota qator boshqa modelga tegishli"), "ConsumptionNormAdd", "")
			return
		}
	}

	if consumeLineID > 0 {
		ok, err := s.Store.Repo().LineExistsByID(consumeLineID)
		if err != nil {
			s.Utils.SendError(c, err, "ConsumptionNormAdd: consume line", "")
			return
		}
		if !ok {
			s.Utils.SendError(c, errors.New("ishlatilish joyi topilmadi"), "ConsumptionNormAdd", "")
			return
		}
	}
	if receiveLineID > 0 {
		ok, err := s.Store.Repo().LineExistsByID(receiveLineID)
		if err != nil {
			s.Utils.SendError(c, err, "ConsumptionNormAdd: receive line", "")
			return
		}
		if !ok {
			s.Utils.SendError(c, errors.New("i/ch joyi topilmadi"), "ConsumptionNormAdd", "")
			return
		}
	}

	consume := sql.NullInt64{}
	if consumeLineID > 0 {
		consume = sql.NullInt64{Int64: int64(consumeLineID), Valid: true}
	}
	receive := sql.NullInt64{}
	if receiveLineID > 0 {
		receive = sql.NullInt64{Int64: int64(receiveLineID), Valid: true}
	}

	item, err := s.Store.Repo().ConsumptionNormInsertItem(
		modelID,
		c.GetInt("user_id"),
		parentItemID,
		componentID,
		quantity,
		consume,
		receive,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.Utils.SendError(c, errors.New("ota qator topilmadi"), "ConsumptionNormAdd", "")
			return
		}
		s.Utils.SendError(c, err, "ConsumptionNormAdd", "")
		return
	}
	s.Utils.SendOK(c, item)
}

func (s *ServerModel) ConsumptionNormDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormDelete: ReadBody", "")
		return
	}

	itemID := getProductionInt(jsonMap, "id", "item_id")
	if itemID == 0 {
		s.Utils.SendError(c, errors.New("id is required"), "ConsumptionNormDelete", "")
		return
	}

	deleted, err := s.Store.Repo().ConsumptionNormDeleteItem(itemID, c.GetInt("user_id"))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.Utils.SendError(c, errors.New("qator topilmadi"), "ConsumptionNormDelete", "")
			return
		}
		s.Utils.SendError(c, err, "ConsumptionNormDelete", "")
		return
	}
	s.Utils.SendOK(c, map[string]any{"deleted": deleted})
}

func (s *ServerModel) ConsumptionNormUpdateLines(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpdateLines: ReadBody", "")
		return
	}

	itemID := getProductionInt(jsonMap, "id", "item_id")
	if itemID == 0 {
		s.Utils.SendError(c, errors.New("id is required"), "ConsumptionNormUpdateLines", "")
		return
	}

	exists, err := s.Store.Repo().ConsumptionNormItemExists(itemID)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpdateLines: ConsumptionNormItemExists", "")
		return
	}
	if !exists {
		s.Utils.SendError(c, errors.New("qator topilmadi"), "ConsumptionNormUpdateLines", "")
		return
	}

	consumeLineID := getProductionInt(jsonMap, "consume_line_id")
	receiveLineID := getProductionInt(jsonMap, "receive_line_id")

	if consumeLineID > 0 {
		ok, err := s.Store.Repo().LineExistsByID(consumeLineID)
		if err != nil {
			s.Utils.SendError(c, err, "ConsumptionNormUpdateLines: consume line", "")
			return
		}
		if !ok {
			s.Utils.SendError(c, errors.New("ishlatilish joyi topilmadi"), "ConsumptionNormUpdateLines", "")
			return
		}
	}
	if receiveLineID > 0 {
		ok, err := s.Store.Repo().LineExistsByID(receiveLineID)
		if err != nil {
			s.Utils.SendError(c, err, "ConsumptionNormUpdateLines: receive line", "")
			return
		}
		if !ok {
			s.Utils.SendError(c, errors.New("i/ch joyi topilmadi"), "ConsumptionNormUpdateLines", "")
			return
		}
	}

	consume := sql.NullInt64{}
	if consumeLineID > 0 {
		consume = sql.NullInt64{Int64: int64(consumeLineID), Valid: true}
	}
	receive := sql.NullInt64{}
	if receiveLineID > 0 {
		receive = sql.NullInt64{Int64: int64(receiveLineID), Valid: true}
	}

	if err := s.Store.Repo().ConsumptionNormUpdateItemLines(itemID, c.GetInt("user_id"), consume, receive); err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpdateLines", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ConsumptionNormItems(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormItems: ReadBody", "")
		return
	}

	modelID := getProductionInt(jsonMap, "model_id", "id")
	if modelID == 0 {
		s.Utils.SendError(c, errors.New("model_id is required"), "ConsumptionNormItems", "")
		return
	}

	data, err := s.Store.Repo().ConsumptionNormItemsByModel(modelID)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormItems", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ConsumptionNormTemplate(c *gin.Context) {
	lines, err := s.Store.Repo().LinesListForLookup()
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormTemplate: LinesListForLookup", "")
		return
	}

	f := excelize.NewFile()
	defer f.Close()
	sheetName := "BOM list"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormTemplate: SetSheetName", "")
		return
	}

	_ = f.SetCellValue(sheetName, "A1", "Modeli")
	_ = f.SetCellValue(sheetName, "C1", "ID")
	headers := []string{
		"",
		"Factory product code",
		"Miqdori\nQuantity",
		"O`lchov birligi\nUnit of measurement",
		"ishlatilish joyi",
		"i/ch joyi",
	}
	for i, header := range headers {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		_ = f.SetCellValue(sheetName, cell, header)
	}

	_ = f.SetCellValue(sheetName, "J2", "liniyalar nomi")
	for i, line := range lines {
		_ = f.SetCellValue(sheetName, "J"+strconv.Itoa(i+3), line.Name)
	}

	buffer, err := f.WriteToBuffer()
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormTemplate: WriteToBuffer", "")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="bom_list_template.xlsx"`)
	c.Data(
		200,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		buffer.Bytes(),
	)
}

func (s *ServerModel) ConsumptionNormUpload(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpload: ReadBody", "")
		return
	}

	file64 := getProductionString(jsonMap, "file64")
	if file64 == "" {
		s.Utils.SendError(c, errors.New("file64 is required"), "ConsumptionNormUpload", "")
		return
	}

	requestModelID := getProductionInt(jsonMap, "model_id", "id")

	data, err := utils.Base64Decode(file64)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpload: Base64Decode", "")
		return
	}

	f, err := excelize.OpenReader(bytes.NewReader([]byte(data)))
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpload: OpenReader", "")
		return
	}
	defer func() {
		_ = f.Close()
	}()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpload: GetRows", "")
		return
	}
	rows = consumptionNormEnhanceQuantityColumn(f, sheetName, rows)

	modelName := strings.TrimSpace(consumptionNormCellValue(f, sheetName, "B1"))
	modelIDText := strings.TrimSpace(consumptionNormCellValue(f, sheetName, "D1"))
	fileModelID := parseProductionInt(modelIDText)

	resolveModelID := requestModelID
	if resolveModelID == 0 {
		resolveModelID = fileModelID
	}
	model, err := s.Store.Repo().ModelResolveForConsumptionNorm(resolveModelID, modelName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			s.Utils.SendError(c, errors.New("model topilmadi"), "ConsumptionNormUpload: model", map[string]any{
				"model_name": modelName,
				"model_id":   resolveModelID,
			})
			return
		}
		s.Utils.SendError(c, err, "ConsumptionNormUpload: ModelResolveForConsumptionNorm", "")
		return
	}
	if requestModelID > 0 && fileModelID > 0 && requestModelID != fileModelID {
		s.Utils.SendError(c, errors.New("fayldagi model ID sahifadagi model bilan mos kelmaydi"), "ConsumptionNormUpload: model mismatch", nil)
		return
	}

	dataStart := consumptionNormDetectDataStart(rows)
	lineMap, err := s.consumptionNormLineMap()
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpload: line map", "")
		return
	}

	parsedRows := make([]consumptionNormParsedRow, 0)
	conflicts := make([]consumptionNormUploadConflict, 0)
	codes := make([]string, 0)

	for rowIndex := dataStart; rowIndex < len(rows); rowIndex++ {
		row := rows[rowIndex]
		if consumptionNormRowIsEmpty(row) {
			continue
		}

		excelRow := rowIndex + 1
		rawFactoryCode := strings.TrimSpace(getRowValue(row, 1))
		if rawFactoryCode == "" || rawFactoryCode == "-" {
			continue
		}

		item := consumptionNormParsedRow{Row: excelRow}
		item.GroupLevel = parseProductionInt(getRowValue(row, 0))
		item.FactoryCode = cleanProductionCell(rawFactoryCode)
		item.Quantity = parseProductionFloat(getRowValue(row, 2))
		item.ConsumeLine = strings.TrimSpace(getRowValue(row, 4))
		item.ReceiveLine = strings.TrimSpace(getRowValue(row, 5))

		if item.FactoryCode == "" {
			continue
		}
		codes = append(codes, item.FactoryCode)

		if item.ConsumeLine != "" {
			lineID, ok := lookupConsumptionLine(lineMap, item.ConsumeLine)
			if !ok {
				conflicts = append(conflicts, consumptionNormUploadConflict{
					Row: excelRow, Field: "consume_line", Value: item.ConsumeLine,
					Message: "ishlatilish joyi bazada topilmadi",
				})
				continue
			}
			item.ConsumeLineID = lineID
		}
		if item.ReceiveLine != "" {
			lineID, ok := lookupConsumptionLine(lineMap, item.ReceiveLine)
			if !ok {
				conflicts = append(conflicts, consumptionNormUploadConflict{
					Row: excelRow, Field: "receive_line", Value: item.ReceiveLine,
					Message: "i/ch joyi bazada topilmadi",
				})
				continue
			}
			item.ReceiveLineID = lineID
		}

		parsedRows = append(parsedRows, item)
	}

	if len(parsedRows) == 0 && len(conflicts) == 0 {
		s.Utils.SendError(c, errors.New("faylda import qilinadigan qatorlar topilmadi (ma'lumotlar 4-qatordan boshlanadi)"), "ConsumptionNormUpload: empty file", nil)
		return
	}

	componentMap, err := s.Store.Repo().ComponentsIDsByCodes(uniqueStrings(codes))
	if err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpload: ComponentsIDsByCodes", "")
		return
	}

	validRows := make([]consumptionNormParsedRow, 0, len(parsedRows))
	for _, item := range parsedRows {
		componentID, ok := componentMap[item.FactoryCode]
		if !ok {
			conflicts = append(conflicts, consumptionNormUploadConflict{
				Row: item.Row, Field: "factory_code", Value: item.FactoryCode,
				Message: "komponent bazada topilmadi",
			})
			continue
		}
		item.ComponentID = componentID
		validRows = append(validRows, item)
	}

	if len(validRows) == 0 {
		s.Utils.SendError(c, errors.New("import validation failed"), "ConsumptionNormUpload: no valid rows", conflicts)
		return
	}

	writeItems := make([]store.ConsumptionNormWriteItem, 0, len(validRows))
	for index, item := range validRows {
		writeItem := store.ConsumptionNormWriteItem{
			SortOrder:   index + 1,
			GroupLevel:  item.GroupLevel,
			ComponentID: item.ComponentID,
			Quantity:    item.Quantity,
		}
		if item.ConsumeLineID > 0 {
			writeItem.ConsumeLineID = sql.NullInt64{Int64: int64(item.ConsumeLineID), Valid: true}
		}
		if item.ReceiveLineID > 0 {
			writeItem.ReceiveLineID = sql.NullInt64{Int64: int64(item.ReceiveLineID), Valid: true}
		}
		writeItems = append(writeItems, writeItem)
	}

	if err := s.Store.Repo().ConsumptionNormReplace(model.ID, c.GetInt("user_id"), writeItems); err != nil {
		s.Utils.SendError(c, err, "ConsumptionNormUpload: ConsumptionNormReplace", "")
		return
	}

	response := map[string]any{
		"model_id": model.ID,
		"imported": len(validRows),
	}
	if len(conflicts) > 0 {
		response["conflicts"] = conflicts
	}
	s.Utils.SendOK(c, response)
}

func (s *ServerModel) consumptionNormLineMap() (map[string]int, error) {
	lines, err := s.Store.Repo().LinesListForLookup()
	if err != nil {
		return nil, err
	}
	result := make(map[string]int, len(lines))
	for _, line := range lines {
		result[normalizeConsumptionLineName(line.Name)] = line.LineID
	}
	return result, nil
}

func consumptionNormCellValue(f *excelize.File, sheet, cell string) string {
	value, err := f.GetCellValue(sheet, cell)
	if err != nil {
		return ""
	}
	return value
}

func consumptionNormDetectDataStart(rows [][]string) int {
	if len(rows) > 0 {
		a1 := strings.ToLower(strings.TrimSpace(getRowValue(rows[0], 0)))
		if a1 == "model nomi" || a1 == "modeli" {
			return 3
		}
	}
	if len(rows) > 1 {
		header := strings.ToLower(strings.TrimSpace(getRowValue(rows[1], 1)))
		if strings.Contains(header, "odoo") || strings.Contains(header, "factory") || strings.Contains(header, "kod") {
			return 3
		}
	}
	if len(rows) > 0 {
		header := strings.ToLower(strings.TrimSpace(getRowValue(rows[0], 1)))
		if strings.Contains(header, "odoo") || strings.Contains(header, "factory") || strings.Contains(header, "kod") {
			return 1
		}
	}
	return 3
}

func consumptionNormRowIsEmpty(row []string) bool {
	rawFactoryCode := strings.TrimSpace(getRowValue(row, 1))
	if rawFactoryCode != "" && rawFactoryCode != "-" {
		return false
	}
	for i := 0; i < 6; i++ {
		if i == 1 {
			continue
		}
		if strings.TrimSpace(getRowValue(row, i)) != "" {
			return false
		}
	}
	return true
}

func consumptionNormEnhanceQuantityColumn(f *excelize.File, sheet string, rows [][]string) [][]string {
	if len(rows) == 0 {
		return rows
	}
	dataStart := consumptionNormDetectDataStart(rows)
	result := make([][]string, len(rows))
	for i, row := range rows {
		copied := make([]string, len(row))
		copy(copied, row)
		result[i] = copied
	}
	for rowIndex := dataStart; rowIndex < len(result); rowIndex++ {
		if consumptionNormRowIsEmpty(result[rowIndex]) {
			continue
		}
		excelRow := rowIndex + 1
		cell, err := excelize.CoordinatesToCellName(3, excelRow)
		if err != nil {
			continue
		}
		value, err := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
		if err != nil || strings.TrimSpace(value) == "" {
			continue
		}
		for len(result[rowIndex]) <= 2 {
			result[rowIndex] = append(result[rowIndex], "")
		}
		result[rowIndex][2] = strings.TrimSpace(value)
	}
	return result
}

func normalizeConsumptionLineName(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "_", "")
	value = strings.ReplaceAll(value, "-", "")
	return value
}

func lookupConsumptionLine(lineMap map[string]int, raw string) (int, bool) {
	norm := normalizeConsumptionLineName(raw)
	if norm == "" {
		return 0, false
	}
	if id, ok := lineMap[norm]; ok {
		return id, true
	}
	if len(norm) < 2 {
		return 0, false
	}
	for key, id := range lineMap {
		if strings.HasPrefix(key, norm) || strings.HasPrefix(norm, key) {
			return id, true
		}
	}
	return 0, false
}

func uniqueStrings(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		result = append(result, value)
	}
	return result
}
