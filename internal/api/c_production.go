package api

import (
	"bytes"
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
	"github.com/xuri/excelize/v2"
)

type ProductionInfo struct {
	Count         int                      `json:"count"`
	CountByModels []models.ModelsCount     `json:"count_by_models"`
	Serials       []store.ReportWithSerial `json:"serials"`
}

type productionComponentUploadRow struct {
	row  int
	item models.TechComponent
}

type productionComponentUploadConflict struct {
	Row        int    `json:"row"`
	Field      string `json:"field"`
	Value      string `json:"value"`
	Message    string `json:"message"`
	ExistingID int    `json:"existing_id"`
}

func (s *ServerModel) ProductionComponentsGetAll(c *gin.Context) {
	data, err := s.Store.Repo().ComponentsGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsGetAll", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionComponentTypesGet(c *gin.Context) {
	data, err := s.Store.Repo().TypesGet()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentTypesGet", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionComponentUnitsGet(c *gin.Context) {
	data, err := s.Store.Repo().UnitsGet()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentUnitsGet", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionComponentsAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsAdd: ReadBody", "")
		return
	}

	data, err := s.productionComponentFromJSON(jsonMap)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsAdd: parse", "")
		return
	}

	if err := s.Store.Repo().TechComponentsInsert(data, c.GetInt("user_id")); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsAdd", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionComponentsUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpdate: ReadBody", "")
		return
	}

	data, err := s.productionComponentFromJSON(jsonMap)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpdate: parse", "")
		return
	}
	if data.ID == 0 {
		s.Utils.SendError(c, errors.New("component id is required"), "ProductionComponentsUpdate", "")
		return
	}

	if err := s.Store.Repo().ComponentsUpdate(data, c.GetInt("user_id")); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpdate", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionComponentsDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsDelete: ReadBody", "")
		return
	}

	componentID := getProductionInt(jsonMap, "id", "component_id")
	if componentID == 0 {
		factoryCode := cleanProductionCell(getProductionString(jsonMap, "factory_code", "value"))
		if factoryCode == "" {
			s.Utils.SendError(c, errors.New("component id yoki factory code kerak"), "ProductionComponentsDelete", "")
			return
		}
		item, lookupErr := s.Store.Repo().ComponentsGetByPrimaryKey(factoryCode)
		if lookupErr != nil {
			s.Utils.SendError(c, lookupErr, "ProductionComponentsDelete: lookup", "")
			return
		}
		componentID = item.ID
	}

	if err := s.Store.Repo().TechComponentsDelete(componentID, c.GetInt("user_id")); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsDelete", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"id": componentID})
}

func (s *ServerModel) ProductionComponentPhotoUpload(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentPhotoUpload: ReadBody", "")
		return
	}

	componentID := getProductionInt(jsonMap, "component_id", "id")
	if componentID == 0 {
		s.Utils.SendError(c, errors.New("component id is required"), "ProductionComponentPhotoUpload", "")
		return
	}

	file64 := getProductionString(jsonMap, "file64")
	if file64 == "" {
		s.Utils.SendError(c, errors.New("file64 is required"), "ProductionComponentPhotoUpload", "")
		return
	}

	data, extension, err := decodeComponentPhoto(file64)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentPhotoUpload: decode", "")
		return
	}
	if len(data) > 5*1024*1024 {
		s.Utils.SendError(c, errors.New("photo size must be less than 5MB"), "ProductionComponentPhotoUpload", "")
		return
	}

	if err := os.MkdirAll(filepath.Join("uploads", "components"), 0755); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentPhotoUpload: MkdirAll", "")
		return
	}

	filename := strconv.Itoa(componentID) + extension
	fullPath := filepath.Join("uploads", "components", filename)
	if err := os.WriteFile(fullPath, data, 0644); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentPhotoUpload: WriteFile", "")
		return
	}

	photoPath := "/uploads/components/" + filename
	if err := s.Store.Repo().ComponentsUpdatePhoto(componentID, photoPath, c.GetInt("user_id")); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentPhotoUpload: ComponentsUpdatePhoto", "")
		return
	}

	s.Utils.SendOK(c, photoPath)
}

func decodeComponentPhoto(file64 string) ([]byte, string, error) {
	commaIndex := strings.IndexByte(file64, ',')
	if commaIndex == -1 {
		return nil, "", errors.New("invalid image data")
	}

	meta := strings.ToLower(file64[:commaIndex])
	raw, err := base64.StdEncoding.DecodeString(file64[commaIndex+1:])
	if err != nil {
		return nil, "", err
	}

	mimeType := http.DetectContentType(raw)
	switch {
	case strings.Contains(meta, "image/jpeg") || mimeType == "image/jpeg":
		return raw, ".jpg", nil
	case strings.Contains(meta, "image/png") || mimeType == "image/png":
		return raw, ".png", nil
	case strings.Contains(meta, "image/webp") || mimeType == "image/webp":
		return raw, ".webp", nil
	default:
		return nil, "", errors.New("only jpg, png and webp images are allowed")
	}
}

func (s *ServerModel) ProductionComponentsUpload(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpload: ReadBody", "")
		return
	}

	file64 := getProductionString(jsonMap, "file64")
	if file64 == "" {
		s.Utils.SendError(c, errors.New("file64 is required"), "ProductionComponentsUpload", "")
		return
	}

	data, err := utils.Base64Decode(file64)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpload: Base64Decode", "")
		return
	}

	f, err := excelize.OpenReader(bytes.NewReader([]byte(data)))
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpload: OpenReader", "")
		return
	}
	defer func() {
		if err := f.Close(); err != nil {
			s.Utils.SendError(c, err, "ProductionComponentsUpload: CloseFile", "")
		}
	}()

	sheetName := f.GetSheetName(0)
	rows, err := f.GetRows(sheetName)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpload: GetRows", "")
		return
	}
	rows = productionEnhanceXLSXNumericRows(f, sheetName, rows)

	typeMap, err := s.productionTypeMap()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpload: TypesGet", "")
		return
	}
	unitMap, err := s.productionUnitMap()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpload: UnitsGet", "")
		return
	}

	parsedRows := make([]productionComponentUploadRow, 0, len(rows))
	var parseErrors []string
	if headerRows, headerErrors, ok := productionComponentsFromHeaderRows(rows, typeMap, unitMap); ok {
		parsedRows = headerRows
		parseErrors = headerErrors
	} else if isProductionBOMSheet(rows) {
		parsedRows, parseErrors = productionComponentsFromBOMRows(rows, typeMap, unitMap)
	} else {
		for i, row := range rows {
			if i == 0 || productionComponentRowIsEmpty(row, 0) {
				continue
			}
			rowNumber := i + 1
			item, err := productionComponentFromXLSXRow(row, typeMap, unitMap)
			if err != nil {
				parseErrors = append(parseErrors, "row "+strconv.Itoa(rowNumber)+": "+err.Error())
				continue
			}
			parsedRows = append(parsedRows, productionComponentUploadRow{row: rowNumber, item: item})
		}
	}

	uploadConflicts := productionParseErrorsToConflicts(parseErrors)
	if len(parsedRows) == 0 && len(uploadConflicts) == 0 {
		s.Utils.SendError(c, errors.New("faylda import qilinadigan qatorlar topilmadi"), "ProductionComponentsUpload: empty file", nil)
		return
	}
	if len(parsedRows) == 0 {
		s.Utils.SendError(c, errors.New("import validation failed"), "ProductionComponentsUpload: parse rows", uploadConflicts)
		return
	}

	preparedRows, prepareConflicts, err := s.productionComponentUploadPrepare(parsedRows)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsUpload: duplicate check", "")
		return
	}
	uploadConflicts = append(uploadConflicts, prepareConflicts...)

	if len(preparedRows) == 0 {
		s.Utils.SendError(c, errors.New("import validation failed"), "ProductionComponentsUpload: no valid rows", uploadConflicts)
		return
	}

	inserted := 0
	updated := 0
	for _, row := range preparedRows {
		if row.item.ID > 0 {
			if err := s.Store.Repo().ComponentsUpdate(row.item, c.GetInt("user_id")); err != nil {
				s.Utils.SendError(c, err, "ProductionComponentsUpload: ComponentsUpdate", row.row)
				return
			}
			updated++
			continue
		}
		if err := s.Store.Repo().TechComponentsInsert(row.item, c.GetInt("user_id")); err != nil {
			s.Utils.SendError(c, err, "ProductionComponentsUpload: TechComponentsInsert", row.row)
			return
		}
		inserted++
	}

	response := map[string]any{
		"inserted": inserted,
		"updated":  updated,
	}
	if len(uploadConflicts) > 0 {
		response["conflicts"] = uploadConflicts
	}
	s.Utils.SendOK(c, response)
}

func (s *ServerModel) productionComponentUploadPrepare(rows []productionComponentUploadRow) ([]productionComponentUploadRow, []productionComponentUploadConflict, error) {
	conflicts := []productionComponentUploadConflict{}
	factoryRows := make(map[string]int, len(rows))
	factoryCodes := make([]string, 0, len(rows))
	validRows := make([]productionComponentUploadRow, 0, len(rows))

	for _, row := range rows {
		if productionComponentItemIsEmpty(row.item) {
			continue
		}
		factoryCode := strings.TrimSpace(row.item.FactoryCode)
		if factoryCode == "" {
			conflicts = append(conflicts, productionComponentUploadConflict{
				Row:     row.row,
				Field:   "Factory product code",
				Value:   "",
				Message: "Korxona kodi (Factory product code) majburiy — ODOO kod o'rniga ishlatilmaydi",
			})
			continue
		}
		if firstRow, ok := factoryRows[factoryCode]; ok {
			conflicts = append(conflicts, productionComponentUploadConflict{
				Row:        row.row,
				Field:      "Factory product code",
				Value:      factoryCode,
				Message:    "duplicate in file, first row " + strconv.Itoa(firstRow),
				ExistingID: s.productionConflictExistingID(factoryCode, 0),
			})
			continue
		}
		factoryRows[factoryCode] = row.row
		factoryCodes = append(factoryCodes, factoryCode)
		validRows = append(validRows, row)
	}

	if len(validRows) == 0 {
		return nil, conflicts, nil
	}

	existing, err := s.Store.Repo().ComponentsFindExistingKeys(factoryCodes, []string{}, []string{})
	if err != nil {
		return nil, nil, err
	}

	existingByFactory := make(map[string]int, len(existing))
	for _, item := range existing {
		if code := strings.TrimSpace(item.FactoryCode); code != "" {
			existingByFactory[code] = item.ID
		}
	}

	importRows := make([]productionComponentUploadRow, 0, len(validRows))
	for _, row := range validRows {
		item := row.item
		if item.ID == 0 {
			item.ID = existingByFactory[strings.TrimSpace(item.FactoryCode)]
		}
		if item.ID > 0 {
			if factoryCode := strings.TrimSpace(item.FactoryCode); factoryCode != "" {
				if existingID, ok := existingByFactory[factoryCode]; ok && existingID != item.ID {
					conflicts = append(conflicts, productionComponentUploadConflict{
						Row:        row.row,
						Field:      "Factory product code",
						Value:      factoryCode,
						Message:    "belongs to another component, id " + strconv.Itoa(existingID),
						ExistingID: existingID,
					})
					continue
				}
			}
		}
		importRows = append(importRows, productionComponentUploadRow{row: row.row, item: item})
	}

	return importRows, conflicts, nil
}

func (s *ServerModel) productionConflictExistingID(factoryCode string, preferredID int) int {
	if preferredID > 0 {
		return preferredID
	}
	factoryCode = strings.TrimSpace(factoryCode)
	if factoryCode == "" {
		return 0
	}
	item, err := s.Store.Repo().ComponentsGetByPrimaryKey(factoryCode)
	if err != nil || item.ID <= 0 {
		return 0
	}
	return item.ID
}

func productionParseErrorsToConflicts(parseErrors []string) []productionComponentUploadConflict {
	conflicts := make([]productionComponentUploadConflict, 0, len(parseErrors))
	for _, parseError := range parseErrors {
		if !strings.HasPrefix(parseError, "row ") {
			continue
		}
		parts := strings.SplitN(parseError, ": ", 2)
		if len(parts) != 2 {
			continue
		}
		rowNumber, err := strconv.Atoi(strings.TrimPrefix(parts[0], "row "))
		if err != nil || rowNumber <= 0 {
			continue
		}
		message := parts[1]
		field := "Row"
		value := ""
		switch {
		case strings.Contains(message, "factory product code"):
			field = "Factory product code"
			message = "Korxona kodi (Factory product code) majburiy — ODOO kod o'rniga ishlatilmaydi"
		case strings.Contains(message, "unknown unit"):
			field = "Unit of measurement"
		case strings.Contains(message, "unknown type"):
			field = "Type of procurement"
		}
		conflicts = append(conflicts, productionComponentUploadConflict{
			Row:     rowNumber,
			Field:   field,
			Value:   value,
			Message: message,
		})
	}
	return conflicts
}

func productionComponentItemIsEmpty(item models.TechComponent) bool {
	return strings.TrimSpace(item.FactoryCode) == "" &&
		strings.TrimSpace(item.OdooCode) == "" &&
		strings.TrimSpace(item.ManufacturerCode) == "" &&
		strings.TrimSpace(item.FullNameUz) == "" &&
		strings.TrimSpace(item.StandardNameUz) == ""
}

func resolveProductionComponentIdentity(item *models.TechComponent, requireFactory bool) error {
	item.OdooCode = cleanProductionCell(item.OdooCode)
	item.FactoryCode = cleanProductionCell(item.FactoryCode)
	item.ManufacturerCode = cleanProductionCell(item.ManufacturerCode)

	if requireFactory && item.FactoryCode == "" {
		return errors.New("factory product code is required")
	}
	if item.FactoryCode == "" && item.OdooCode == "" && item.ManufacturerCode == "" {
		return errors.New("factory product code is required")
	}
	if item.ManufacturerCode == "" && item.FactoryCode != "" {
		item.ManufacturerCode = item.FactoryCode
	}
	return nil
}

func (s *ServerModel) ProductionComponentsTemplate(c *gin.Context) {
	units, err := s.Store.Repo().UnitsGet()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsTemplate: UnitsGet", "")
		return
	}

	typeOptions := []string{"import", "local", "production"}
	unitOptions := make([]string, 0, len(units))
	for _, unit := range units {
		unitOptions = append(unitOptions, unit.Name)
	}
	if len(unitOptions) == 0 {
		unitOptions = []string{"pcs", "dona"}
	}

	f := excelize.NewFile()
	defer f.Close()
	sheetName := "Base for Bom list"
	if err := f.SetSheetName("Sheet1", sheetName); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsTemplate: SetSheetName", "")
		return
	}

	row1 := productionBOMHeaderRow1()
	for i, header := range row1 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		f.SetCellValue(sheetName, cell, header)
	}
	row2 := productionBOMHeaderRow2()
	for i, header := range row2 {
		cell, _ := excelize.CoordinatesToCellName(i+1, 2)
		f.SetCellValue(sheetName, cell, header)
	}
	for col := 1; col <= 13; col++ {
		start, _ := excelize.CoordinatesToCellName(col, 1)
		end, _ := excelize.CoordinatesToCellName(col, 2)
		_ = f.MergeCell(sheetName, start, end)
	}

	if _, err := f.NewSheet("Lists"); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsTemplate: NewSheet", "")
		return
	}
	f.SetCellValue("Lists", "A1", "Types")
	for i, value := range typeOptions {
		f.SetCellValue("Lists", "A"+strconv.Itoa(i+2), value)
	}
	f.SetCellValue("Lists", "B1", "Units")
	for i, value := range unitOptions {
		f.SetCellValue("Lists", "B"+strconv.Itoa(i+2), value)
	}

	typeValidation := excelize.NewDataValidation(false)
	typeValidation.Sqref = "I3:I1000"
	typeValidation.SetSqrefDropList("Lists!$A$2:$A$" + strconv.Itoa(len(typeOptions)+1))
	if err := f.AddDataValidation(sheetName, typeValidation); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsTemplate: TypeValidation", "")
		return
	}

	unitValidation := excelize.NewDataValidation(false)
	unitValidation.Sqref = "H3:H1000"
	unitValidation.SetSqrefDropList("Lists!$B$2:$B$" + strconv.Itoa(len(unitOptions)+1))
	if err := f.AddDataValidation(sheetName, unitValidation); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsTemplate: UnitValidation", "")
		return
	}

	if err := f.SetSheetVisible("Lists", false); err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsTemplate: SetSheetVisible", "")
		return
	}
	f.SetActiveSheet(0)

	buffer, err := f.WriteToBuffer()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionComponentsTemplate: WriteToBuffer", "")
		return
	}

	c.Header("Content-Disposition", `attachment; filename="components_list_template.xlsx"`)
	c.Data(
		200,
		"application/vnd.openxmlformats-officedocument.spreadsheetml.sheet",
		buffer.Bytes(),
	)
}

func (s *ServerModel) productionComponentFromJSON(jsonMap map[string]any) (models.TechComponent, error) {
	item := models.TechComponent{
		ID:                    getProductionInt(jsonMap, "id"),
		DetalTuriKodi:         getProductionString(jsonMap, "detal_turi_kodi"),
		FactoryCode:           cleanProductionCell(getProductionString(jsonMap, "factory_code")),
		ManufacturerCode:      cleanProductionCell(getProductionString(jsonMap, "manufacturer_code")),
		OdooCode:              cleanProductionCell(getProductionString(jsonMap, "odoo_code")),
		FullNameUz:            getProductionString(jsonMap, "full_name_uz"),
		StandardNameUz:        getProductionString(jsonMap, "standard_name_uz"),
		FullNameRu:            getProductionString(jsonMap, "full_name_ru"),
		StandardNameRu:        getProductionString(jsonMap, "standard_name_ru"),
		SpecificationUz:       getProductionString(jsonMap, "specification_uz"),
		TypeId:                getProductionInt(jsonMap, "type_id"),
		UnitId:                getProductionInt(jsonMap, "unit_id"),
		NetWeightPcs:          getProductionFloat(jsonMap, "net_weight_pcs", "net_weight_kg"),
		NetWeightSet:          getProductionFloat(jsonMap, "net_weight_set"),
		TechnologicalWastePcs: getProductionFloat(jsonMap, "technological_waste_pcs", "technological_waste"),
		TechnologicalWasteSet: getProductionFloat(jsonMap, "technological_waste_set"),
		Comment:               getProductionString(jsonMap, "comment"),
	}
	if err := resolveProductionComponentIdentity(&item, true); err != nil {
		return models.TechComponent{}, err
	}
	return item, nil
}

func productionBOMHeaderRow1() []string {
	return []string{
		"Detal Turi kodi",
		"ODOO kod",
		"Mahsulotning korxona kodi\nFactory product code",
		"Manufacturer code",
		"Mahsulotning to'liq nomi (O'zb)",
		"Mahsulotning standart nomi (O'zb)",
		"Mahsulotning standart nomi (Ru)",
		"O`lchov birligi",
		"Xarid turi (import, local yoki production)",
		"Xususiyatlari (O'zb)",
		"photo",
		"comment",
		"O'girligi (dona)",
		"O'girligi (Set)",
		"Texnologik chiqit (dona)",
		"Texnologik chiqit (Set)",
	}
}

func productionBOMHeaderRow2() []string {
	return []string{
		"", "", "", "", "", "", "", "", "", "", "", "",
		"dona",
		"Set",
		"dona",
		"Set",
	}
}

func isProductionBOMSheet(rows [][]string) bool {
	if len(rows) == 0 {
		return false
	}
	header := strings.ToLower(strings.Join(rows[0], " "))
	return strings.Contains(header, "factory product code") ||
		strings.Contains(header, "korxona kodi") ||
		strings.Contains(header, "detal turi kodi")
}

func productionXLSXMetricColumnIndexes(bomOffset int) []int {
	return []int{
		bomOffset + 12,
		bomOffset + 13,
		bomOffset + 14,
		bomOffset + 15,
	}
}

func productionEnhanceXLSXNumericRows(f *excelize.File, sheet string, rows [][]string) [][]string {
	if len(rows) == 0 {
		return rows
	}

	bomOffset := productionBOMColumnOffset(rows)
	startIndex := 1
	if len(rows) > 1 && isProductionBOMSubHeaderRow(rows[1], bomOffset) {
		startIndex = 2
	}

	metricCols := productionXLSXMetricColumnIndexes(bomOffset)
	result := make([][]string, len(rows))
	for i, row := range rows {
		copied := make([]string, len(row))
		copy(copied, row)
		result[i] = copied
	}

	for rowIndex := startIndex; rowIndex < len(result); rowIndex++ {
		if isProductionBOMSubHeaderRow(result[rowIndex], bomOffset) || productionComponentRowIsEmpty(result[rowIndex], bomOffset) {
			continue
		}
		excelRow := rowIndex + 1
		for _, colIndex := range metricCols {
			cell, err := excelize.CoordinatesToCellName(colIndex+1, excelRow)
			if err != nil {
				continue
			}
			value, err := f.GetCellValue(sheet, cell, excelize.Options{RawCellValue: true})
			if err != nil || strings.TrimSpace(value) == "" {
				continue
			}
			for len(result[rowIndex]) <= colIndex {
				result[rowIndex] = append(result[rowIndex], "")
			}
			result[rowIndex][colIndex] = strings.TrimSpace(value)
		}
	}

	return result
}

func productionBOMColumnOffset(rows [][]string) int {
	if len(rows) == 0 {
		return 0
	}
	for index, header := range rows[0] {
		key := normalizeProductionKey(productionComponentHeaderText(header))
		if key == "detal turi kodi" {
			return index
		}
	}
	// Legacy template/export: leading ID column shifts BOM data by one.
	key := normalizeProductionKey(productionComponentHeaderText(getRowValue(rows[0], 0)))
	if key == "id" {
		return 1
	}
	return 0
}

func isProductionBOMSubHeaderRow(row []string, bomOffset int) bool {
	joined := strings.ToLower(strings.Join(row, " "))
	if strings.Contains(joined, "1 piece") || strings.Contains(joined, "set.") {
		return true
	}
	// Metadata ends at comment (index 11 without ID, 12 with leading ID); metrics start after.
	metaEnd := 12 + bomOffset
	for i := bomOffset; i < metaEnd && i < len(row); i++ {
		if cleanProductionCell(getRowValue(row, i)) != "" {
			return false
		}
	}
	for i := metaEnd; i < len(row); i++ {
		cell := strings.ToLower(cleanProductionCell(getRowValue(row, i)))
		if cell == "" {
			continue
		}
		if cell == "dona" || cell == "set" || cell == "set." ||
			strings.Contains(cell, "piece") || strings.Contains(cell, "pcs") {
			return true
		}
		return false
	}
	return false
}

// productionComponentRowIsEmpty skips blank rows and metric-only sub-header leftovers.
func productionComponentRowIsEmpty(row []string, bomOffset int) bool {
	if len(row) == 0 {
		return true
	}
	for i := bomOffset + 1; i < len(row); i++ {
		cell := cleanProductionCell(getRowValue(row, i))
		if cell == "" {
			continue
		}
		lower := strings.ToLower(cell)
		if lower == "dona" || lower == "set" || lower == "set." ||
			strings.Contains(lower, "piece") || strings.Contains(lower, "pcs") {
			continue
		}
		return false
	}
	return true
}

func productionComponentsFromBOMRows(rows [][]string, typeMap map[string]int, unitMap map[string]int) ([]productionComponentUploadRow, []string) {
	bomOffset := productionBOMColumnOffset(rows)
	startIndex := 1
	if len(rows) > 1 && isProductionBOMSubHeaderRow(rows[1], bomOffset) {
		startIndex = 2
	}

	result := make([]productionComponentUploadRow, 0, len(rows)-startIndex)
	parseErrors := []string{}
	for i := startIndex; i < len(rows); i++ {
		row := rows[i]
		if isProductionBOMSubHeaderRow(row, bomOffset) || productionComponentRowIsEmpty(row, bomOffset) {
			continue
		}
		rowNumber := i + 1
		item, err := productionComponentFromBOMRow(row, bomOffset, typeMap, unitMap)
		if err != nil {
			parseErrors = append(parseErrors, "row "+strconv.Itoa(rowNumber)+": "+err.Error())
			continue
		}
		result = append(result, productionComponentUploadRow{row: rowNumber, item: item})
	}
	return result, parseErrors
}

func productionComponentFromBOMRow(row []string, bomOffset int, typeMap map[string]int, unitMap map[string]int) (models.TechComponent, error) {
	item := models.TechComponent{
		DetalTuriKodi:         getRowValue(row, bomOffset+0),
		OdooCode:              cleanProductionCell(getRowValue(row, bomOffset+1)),
		FactoryCode:           cleanProductionCell(getRowValue(row, bomOffset+2)),
		ManufacturerCode:      cleanProductionCell(getRowValue(row, bomOffset+3)),
		FullNameUz:            getRowValue(row, bomOffset+4),
		StandardNameUz:        getRowValue(row, bomOffset+5),
		StandardNameRu:        getRowValue(row, bomOffset+6),
		SpecificationUz:       getRowValue(row, bomOffset+9),
		Comment:               getRowValue(row, bomOffset+11),
		NetWeightPcs:          parseProductionFloat(getRowValue(row, bomOffset+12)),
		NetWeightSet:          parseProductionFloat(getRowValue(row, bomOffset+13)),
		TechnologicalWastePcs: parseProductionFloat(getRowValue(row, bomOffset+14)),
		TechnologicalWasteSet: parseProductionFloat(getRowValue(row, bomOffset+15)),
	}
	if bomOffset == 1 {
		item.ID = parseProductionInt(getRowValue(row, 0))
	}
	if err := resolveProductionComponentIdentity(&item, true); err != nil {
		return models.TechComponent{}, err
	}

	typeID := lookupProductionTypeID(typeMap, getRowValue(row, bomOffset+8))
	if typeID == 0 {
		return models.TechComponent{}, errors.New("unknown type " + getRowValue(row, bomOffset+8))
	}
	unitID := lookupProductionUnitID(unitMap, getRowValue(row, bomOffset+7))
	if unitID == 0 {
		return models.TechComponent{}, errors.New("unknown unit " + getRowValue(row, bomOffset+7))
	}

	item.UnitId = unitID
	item.TypeId = typeID
	return item, nil
}

func productionComponentFromXLSXRow(row []string, typeMap map[string]int, unitMap map[string]int) (models.TechComponent, error) {
	manufacturerCode := strings.TrimSpace(getRowValue(row, 0))
	if manufacturerCode == "" {
		return models.TechComponent{}, errors.New("manufacturer code is required")
	}

	typeCell := getRowValue(row, 6)
	dataOffset := 0
	typeID := lookupProductionID(typeMap, typeCell)
	if typeID == 0 {
		if lookupProductionID(unitMap, typeCell) == 0 {
			return models.TechComponent{}, errors.New("unknown type " + typeCell)
		}
		typeID = lookupProductionID(typeMap, "import")
		if typeID == 0 {
			return models.TechComponent{}, errors.New("unknown type " + typeCell + "; default type import is not configured")
		}
		dataOffset = -1
	}
	unitID := lookupProductionID(unitMap, getRowValue(row, 7+dataOffset))
	if unitID == 0 {
		return models.TechComponent{}, errors.New("unknown unit " + getRowValue(row, 7+dataOffset))
	}

	return models.TechComponent{
		ManufacturerCode:      manufacturerCode,
		FullNameUz:            getRowValue(row, 1),
		StandardNameUz:        getRowValue(row, 2),
		FullNameRu:            getRowValue(row, 3),
		SpecificationUz:       getRowValue(row, 4),
		TypeId:                typeID,
		UnitId:                unitID,
		NetWeightPcs:          parseProductionFloat(getRowValue(row, 13+dataOffset)),
		TechnologicalWastePcs: parseProductionFloat(getRowValue(row, 8+dataOffset)),
		Comment:               getRowValue(row, 10+dataOffset),
		OdooCode:              getRowValue(row, 11+dataOffset),
		StandardNameRu:        getRowValue(row, 12+dataOffset),
	}, nil
}

func productionComponentsApplyBOMSubHeaders(headers map[int]string, row0, row1 []string) {
	metricGroup := ""
	maxIndex := len(row0)
	if len(row1) > maxIndex {
		maxIndex = len(row1)
	}
	for index := 0; index < maxIndex; index++ {
		top := strings.ToLower(strings.TrimSpace(getRowValue(row0, index)))
		sub := strings.ToLower(strings.TrimSpace(getRowValue(row1, index)))

		switch {
		case strings.Contains(top, "net weight") || strings.Contains(top, "o'girligi") || strings.Contains(top, "ogirligi"):
			metricGroup = "net_weight"
		case strings.Contains(top, "technological waste") || strings.Contains(top, "texnologik chiqit"):
			metricGroup = "technological_waste"
		case strings.Contains(top, "irreversible"):
			metricGroup = ""
		}

		switch {
		case strings.Contains(sub, "piece") || sub == "1 pcs" || sub == "1pcs" || sub == "dona":
			if metricGroup != "" {
				headers[index] = metricGroup + "_pcs"
			}
		case strings.Contains(sub, "set"):
			if metricGroup != "" {
				headers[index] = metricGroup + "_set"
			}
		}
	}
}

func productionComponentsFromHeaderRows(rows [][]string, typeMap map[string]int, unitMap map[string]int) ([]productionComponentUploadRow, []string, bool) {
	if len(rows) < 2 {
		return nil, nil, false
	}

	headers := make(map[int]string, len(rows[0]))
	for index, header := range rows[0] {
		key := productionComponentHeaderKey(header)
		if key != "" && key != "metric_pcs" && key != "metric_set" {
			headers[index] = key
		}
	}
	bomOffset := productionBOMColumnOffset(rows)
	if len(rows) > 1 && isProductionBOMSubHeaderRow(rows[1], bomOffset) {
		productionComponentsApplyBOMSubHeaders(headers, rows[0], rows[1])
	}
	if !hasProductionComponentHeader(headers, "factory_code") &&
		!hasProductionComponentHeader(headers, "odoo_code") &&
		!hasProductionComponentHeader(headers, "manufacturer_code") {
		return nil, nil, false
	}

	startIndex := 1
	if len(rows) > 1 && isProductionBOMSubHeaderRow(rows[1], bomOffset) {
		startIndex = 2
	}

	result := make([]productionComponentUploadRow, 0, len(rows)-startIndex)
	parseErrors := []string{}
	for rowIndex, row := range rows[startIndex:] {
		if isProductionBOMSubHeaderRow(row, bomOffset) || productionComponentRowIsEmpty(row, bomOffset) {
			continue
		}
		rowNumber := rowIndex + startIndex + 1
		values := map[string]string{}
		for index, key := range headers {
			values[key] = getRowValue(row, index)
		}

		typeID := lookupProductionTypeID(typeMap, values["type"])
		if typeID == 0 {
			typeCell := cleanProductionCell(values["type"])
			if typeCell == "" {
				typeCell = "(bo'sh)"
			}
			parseErrors = append(parseErrors, "row "+strconv.Itoa(rowNumber)+": unknown type "+typeCell)
			continue
		}
		unitID := lookupProductionUnitID(unitMap, values["unit"])
		if unitID == 0 {
			unitCell := cleanProductionCell(values["unit"])
			if unitCell == "" {
				unitCell = "(bo'sh)"
			}
			parseErrors = append(parseErrors, "row "+strconv.Itoa(rowNumber)+": unknown unit "+unitCell)
			continue
		}
		item := models.TechComponent{
			ID:                    parseProductionInt(values["id"]),
			DetalTuriKodi:         values["detal_turi_kodi"],
			FactoryCode:           cleanProductionCell(values["factory_code"]),
			ManufacturerCode:      cleanProductionCell(values["manufacturer_code"]),
			OdooCode:              cleanProductionCell(values["odoo_code"]),
			FullNameUz:            values["full_name_uz"],
			StandardNameUz:        values["standard_name_uz"],
			FullNameRu:            values["full_name_ru"],
			StandardNameRu:        values["standard_name_ru"],
			SpecificationUz:       values["specification_uz"],
			TypeId:                typeID,
			UnitId:                unitID,
			NetWeightPcs:          parseProductionFloat(values["net_weight_pcs"]),
			NetWeightSet:          parseProductionFloat(values["net_weight_set"]),
			TechnologicalWastePcs: parseProductionFloat(values["technological_waste_pcs"]),
			TechnologicalWasteSet: parseProductionFloat(values["technological_waste_set"]),
			Comment:               values["comment"],
		}
		if err := resolveProductionComponentIdentity(&item, true); err != nil {
			parseErrors = append(parseErrors, "row "+strconv.Itoa(rowNumber)+": "+err.Error())
			continue
		}
		result = append(result, productionComponentUploadRow{row: rowNumber, item: item})
	}
	return result, parseErrors, true
}

func productionComponentHeaderKey(header string) string {
	key := normalizeProductionKey(productionComponentHeaderText(header))
	key = strings.ReplaceAll(key, "_", " ")
	key = strings.ReplaceAll(key, "`", "'")
	key = strings.Join(strings.Fields(key), " ")
	switch key {
	case "id":
		return "id"
	case "detal turi kodi":
		return "detal_turi_kodi"
	case "odoo kod", "odoo code", "odoo":
		return "odoo_code"
	case "mahsulotning korxona kodi factory product code", "factory product code", "factory code", "mahsulotning korxona kodi":
		return "factory_code"
	case "ishlab chiqaruvchining kodi manufacturer code", "manufacturer code":
		return "manufacturer_code"
	case "mahsulotning to'liq nomi (o'zb) full name of the product (o'zb)", "full name of the product uzbek", "full name uz", "mahsulotning to'liq nomi (o'zb)":
		return "full_name_uz"
	case "mahsulotning standart nomi (o'zb) standard name of the product (o'zb)", "standard name of the product uzbek", "standard uz", "mahsulotning standart nomi (o'zb)":
		return "standard_name_uz"
	case "mahsulotning to'liq nomi (ru)", "full name of the product (ru)", "full name ru":
		return "full_name_ru"
	case "mahsulotning standart nomi (ru) standard name of the product (ru)", "standard name of the product (rus)", "standard name of the product rus", "standard ru", "mahsulotning standart nomi (ru)":
		return "standard_name_ru"
	case "o'lchov birligi unit of measurement", "unit of measurement", "unit", "o'lchov birligi":
		return "unit"
	case "xarid turi type of procurement", "type of procurement", "type", "xarid turi", "xarid turi (import, local yoki production)":
		return "type"
	case "xususiyatlari (o'zb) specification (o'zb)", "specification uzbek", "specification uz", "xususiyatlari (o'zb)":
		return "specification_uz"
	case "rasm photo", "photo":
		return ""
	case "izohlar notes", "notes", "comment":
		return "comment"
	case "net weight [kg]", "o'girligi (dona)", "o'girligi (set)", "texnologik chiqit (dona)", "texnologik chiqit (set)":
		return ""
	case "1 piece", "1 pcs", "1pcs", "dona":
		return "metric_pcs"
	case "set.", "set":
		return "metric_set"
	case "technological waste":
		return ""
	case "irreversible waste", "irreversible waste pcs", "irreversible waste set":
		return ""
	case "technological waste pcs", "tech waste pcs", "tech waste", "technological waste 1 piece", "texnologik chiqit dona":
		return "technological_waste_pcs"
	case "technological waste set", "tech waste set", "texnologik chiqit set":
		return "technological_waste_set"
	case "net weight pcs", "net weight kg", "net kg", "net weight 1 piece", "o'girligi dona":
		return "net_weight_pcs"
	case "net weight set", "o'girligi set":
		return "net_weight_set"
	default:
		return ""
	}
}

func hasProductionComponentHeader(headers map[int]string, key string) bool {
	for _, header := range headers {
		if header == key {
			return true
		}
	}
	return false
}

func (s *ServerModel) productionTypeMap() (map[string]int, error) {
	items, err := s.Store.Repo().TypesGet()
	if err != nil {
		return nil, err
	}
	return productionIDMap(items), nil
}

func (s *ServerModel) productionUnitMap() (map[string]int, error) {
	items, err := s.Store.Repo().UnitsGet()
	if err != nil {
		return nil, err
	}
	return productionIDMap(items), nil
}

func productionIDMap(items []store.IdName) map[string]int {
	result := make(map[string]int, len(items)*2)
	for _, item := range items {
		key := normalizeProductionKey(item.Name)
		result[key] = item.Id
		result[strconv.Itoa(item.Id)] = item.Id
		for _, alias := range productionAliases(key) {
			result[alias] = item.Id
		}
	}
	return result
}

func lookupProductionID(items map[string]int, value string) int {
	return items[normalizeProductionKey(value)]
}

func lookupProductionTypeID(typeMap map[string]int, value string) int {
	value = cleanProductionCell(value)
	if value == "" {
		return lookupProductionID(typeMap, "import")
	}
	if id := lookupProductionID(typeMap, value); id != 0 {
		return id
	}
	normalized := normalizeProductionKey(value)
	for _, prefix := range []string{"production", "import", "local"} {
		if strings.HasPrefix(normalized, prefix) {
			if id := lookupProductionID(typeMap, prefix); id != 0 {
				return id
			}
		}
	}
	return 0
}

func lookupProductionUnitID(unitMap map[string]int, value string) int {
	value = cleanProductionCell(value)
	if value == "" {
		return lookupProductionID(unitMap, "pcs")
	}
	return lookupProductionID(unitMap, value)
}

func cleanProductionCell(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || value == "-" {
		return ""
	}
	return value
}

func productionComponentHeaderText(header string) string {
	parts := strings.Split(header, "\n")
	english := ""
	fallback := ""
	for _, part := range parts {
		line := strings.TrimSpace(part)
		if line == "" {
			continue
		}
		if fallback == "" {
			fallback = line
		}
		for _, r := range line {
			if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') {
				english = line
				break
			}
		}
	}
	if english != "" {
		return english
	}
	return fallback
}

func normalizeProductionKey(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

func productionAliases(key string) []string {
	switch key {
	case "dona", "шт", "sht", "piece", "pieces", "pc", "pcs":
		return []string{"dona", "шт", "sht", "piece", "pieces", "pc", "pcs"}
	case "gr", "gram", "gramm", "г", "g":
		return []string{"gr", "gram", "gramm", "г", "g"}
	case "kilogram", "kilogramm", "кг", "kg":
		return []string{"kilogram", "kilogramm", "кг", "kg"}
	case "meter", "metr", "м", "m":
		return []string{"meter", "metr", "м", "m"}
	case "m.kub":
		return []string{"m.kub", "m3", "m³", "kub", "kub.m", "м³", "м.куб", "cubic meter", "cubic metre"}
	case "litr":
		return []string{"litr", "l", "liter", "litre", "л", "литр"}
	case "import":
		return []string{"import", "imported", "импорт"}
	case "local":
		return []string{"local", "lokal", "местный"}
	case "production":
		return []string{"production", "ishlab chiqarish", "производство"}
	default:
		return nil
	}
}

func getRowValue(row []string, index int) string {
	if index >= len(row) {
		return ""
	}
	return strings.TrimSpace(row[index])
}

func getProductionString(jsonMap map[string]any, keys ...string) string {
	for _, key := range keys {
		if value, ok := jsonMap[key]; ok && value != nil {
			return strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(toProductionString(value)), "\ufeff"))
		}
	}
	return ""
}

func getProductionInt(jsonMap map[string]any, keys ...string) int {
	for _, key := range keys {
		if value, ok := jsonMap[key]; ok && value != nil {
			switch v := value.(type) {
			case float64:
				return int(v)
			case int:
				return v
			case string:
				parsed, _ := strconv.Atoi(strings.TrimSpace(v))
				return parsed
			}
		}
	}
	return 0
}

func getProductionFloat(jsonMap map[string]any, keys ...string) float64 {
	for _, key := range keys {
		if value, ok := jsonMap[key]; ok && value != nil {
			switch v := value.(type) {
			case float64:
				return v
			case int:
				return float64(v)
			case string:
				return parseProductionFloat(v)
			}
		}
	}
	return 0
}

func parseProductionFloat(value string) float64 {
	value = cleanProductionCell(value)
	if value == "" {
		return 0
	}
	parsed, _ := strconv.ParseFloat(strings.ReplaceAll(value, ",", "."), 64)
	return parsed
}

func parseProductionInt(value string) int {
	parsed, _ := strconv.Atoi(strings.TrimSpace(value))
	return parsed
}

func toProductionString(value any) string {
	switch v := value.(type) {
	case string:
		return v
	case float64:
		return strconv.FormatFloat(v, 'f', -1, 64)
	case int:
		return strconv.Itoa(v)
	default:
		return ""
	}
}

func (s *ServerModel) ProductionInfo(c *gin.Context) {

	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionInfo: ReadBody", "")
		return
	}

	date1, ok := jsonMap["date1"]
	date1String := ""
	if ok {
		if date1 != nil {
			date1String = date1.(string)
			if date1String != "" {
				date1String = date1String[:10]
			}
		}

	}
	date2, ok := jsonMap["date2"]
	s.Utils.DebugLogAny("ProductionInfo date2: ", date2)
	s.Utils.DebugLogAny("ok: ", ok)
	date2String := ""
	if ok {
		if date2 != nil {
			date2String = date2.(string)
			if date2String != "" {
				date2String = date2String[:10]
			}
		}

	}

	s.Utils.DebugLogAny("ProductionInfo date1: ", date1String)
	s.Utils.DebugLogAny("ProductionInfo date2: ", date2String)

	count, err := s.Store.Repo().ProductCount(date1String, date2String)
	if err != nil {
		s.Utils.SendError(c, err, "ProductCount", "")
		return
	}

	countByModels, err := s.Store.Repo().ProductionCountModels(date1String, date2String, nil, nil)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionCountInfo", "")
		return
	}

	data := ProductionInfo{
		Count:         count,
		CountByModels: countByModels,
	}

	serials, err := s.Store.Repo().ProductionReportWithSerials(date1String, date2String)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionReportWithSerials", "")
		return
	}
	data.Serials = serials

	s.Utils.DebugLogAny("ProductionInfo: ", data)

	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionReportXlsx(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionReportXlsx: ReadBody", "")
		return
	}
	s.Utils.DebugLogAny("get date1")
	date1, ok := jsonMap["date1"]
	date1String := ""
	if ok {
		if date1 != nil {
			date1String = date1.(string)
			if date1String != "" {
				date1String = date1String[:10]
			}
		}

	}

	s.Utils.DebugLogAny("get date2")
	date2, ok := jsonMap["date2"]
	date2String := ""
	if ok {
		if date2 != nil {
			date2String = date2.(string)
			if date2String != "" {
				date2String = date2String[:10]
			}
		}

	}

	serials, err := s.Store.Repo().ProductionReportWithSerials(date1String, date2String)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionReportXlsx, ProductionReportWithSerials", "")
		return
	}

	f, err := excelize.OpenFile("report_sample.xlsx")
	if err != nil {
		s.Utils.SendError(c, err, "ProductionReportXlsx, OpenFile", "")
		return
	}
	if err := f.Close(); err != nil {
		s.Utils.SendError(c, err, "ProductionReportXlsx, OpenFile", "")
	}
	defer func() {
		if err := f.Close(); err != nil {
			s.Utils.SendError(c, err, "ProductionReportXlsx, Close", "")
		}
	}()

	style, err := f.NewStyle(
		&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Fill: excelize.Fill{
				Type:    "pattern",
				Color:   []string{"#e5e5e5"},
				Pattern: 1,
			},
			Font: &excelize.Font{Bold: true, Color: "000000"},
			Border: []excelize.Border{
				{Type: "left", Color: "00000000", Style: 1},
				{Type: "right", Color: "00000000", Style: 1},
				{Type: "top", Color: "00000000", Style: 1},
				{Type: "bottom", Color: "00000000", Style: 1},
			},
		},
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionReportXlsx, NewStyle", "")
		return
	}
	style2, err := f.NewStyle(
		&excelize.Style{
			Alignment: &excelize.Alignment{Horizontal: "center"},
			Font:      &excelize.Font{Bold: true, Color: "000000"},
			Border: []excelize.Border{
				{Type: "left", Color: "00000000", Style: 1},
				{Type: "right", Color: "00000000", Style: 1},
				{Type: "top", Color: "00000000", Style: 1},
				{Type: "bottom", Color: "00000000", Style: 1},
			},
		},
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionReportXlsx, NewStyle2", "")
		return
	}
	for index, value := range serials {
		f.SetCellValue("serials", "A"+strconv.Itoa(index+2), index+1)
		f.SetCellValue("serials", "B"+strconv.Itoa(index+2), value.Serial1)
		f.SetCellValue("serials", "C"+strconv.Itoa(index+2), value.Serial2)
		f.SetCellValue("serials", "D"+strconv.Itoa(index+2), value.ModelName)
		f.SetCellValue("serials", "E"+strconv.Itoa(index+2), value.Time)
		if index%2 == 0 {
			f.SetCellStyle("serials", "A"+strconv.Itoa(index+2), "E"+strconv.Itoa(index+2), style)
		} else {
			f.SetCellStyle("serials", "A"+strconv.Itoa(index+2), "E"+strconv.Itoa(index+2), style2)
		}
	}

	if _, err := os.Stat("./web/build/report"); os.IsNotExist(err) {
		os.Mkdir("./web/build/report", 0755)
	}
	err = f.SaveAs("./web/build/report/" + date1String + "-" + date2String + ".xlsx")
	if err != nil {
		s.Utils.SendError(c, err, "WareInOutReportXlsx, SaveAs", "")
		return
	}

	s.Utils.SendOK(c, "/report/"+date1String+"-"+date2String+".xlsx")
}

func (s *ServerModel) ProductionFinPressGetAll(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressGetAll: ReadBody", "")
		return
	}

	data, err := s.Store.Repo().FinPressComponentsGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressGetAll", "")
		return
	}

	if rawID, ok := jsonMap["id"].(float64); ok {
		finPressID := int(rawID)
		if finPressID > 0 {
			for _, item := range data {
				if item.ID == finPressID {
					sessions, err := s.Store.Repo().FinPressPrintSessionsGetLast(0, store.LastRecordsLimit)
					if err != nil {
						s.Utils.SendError(c, err, "ProductionFinPressGetAll: FinPressPrintSessionsGetLast", "")
						return
					}
					s.Utils.SendOK(c, map[string]any{
						"item":          item,
						"last_sessions": sessions,
					})
					return
				}
			}
			s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionFinPressGetAll", "")
			return
		}
	}

	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionFinPressAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressAdd: ReadBody", "")
		return
	}

	componentID := int(jsonMap["component_id"].(float64))
	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	err = s.Store.Repo().FinPressComponentAdd(componentID, c.GetInt("user_id"), strings.TrimSpace(index1), strings.TrimSpace(index2))
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressAdd", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionFinPressUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressUpdate: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	if id <= 0 {
		s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionFinPressUpdate", "")
		return
	}

	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	err = s.Store.Repo().FinPressComponentUpdateIndexes(id, strings.TrimSpace(index1), strings.TrimSpace(index2))
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressUpdate", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionLineResponsiblesGetAll(c *gin.Context) {
	data, err := s.Store.Repo().LineResponsiblesGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionLineResponsiblesGetAll", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionLineResponsiblesAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionLineResponsiblesAdd: ReadBody", "")
		return
	}

	userID := int(jsonMap["user_id"].(float64))
	if userID <= 0 {
		s.Utils.SendError(c, errors.New("foydalanuvchi tanlanmagan"), "ProductionLineResponsiblesAdd", "")
		return
	}

	lineIDs := []int{}
	if rawLineIDs, ok := jsonMap["line_ids"].([]any); ok {
		for _, rawLineID := range rawLineIDs {
			if lineID, ok := rawLineID.(float64); ok && int(lineID) > 0 {
				lineIDs = append(lineIDs, int(lineID))
			}
		}
	}
	if rawLineID, ok := jsonMap["line_id"].(float64); ok && int(rawLineID) > 0 {
		lineIDs = append(lineIDs, int(rawLineID))
	}
	if len(lineIDs) == 0 {
		s.Utils.SendError(c, errors.New("kamida bitta liniya tanlang"), "ProductionLineResponsiblesAdd", "")
		return
	}

	added := 0
	var lastErr error
	for _, lineID := range lineIDs {
		err = s.Store.Repo().LineResponsibleAdd(lineID, userID, c.GetInt("user_id"))
		if err != nil {
			lastErr = err
			continue
		}
		added++
	}
	if added == 0 {
		s.Utils.SendError(c, lastErr, "ProductionLineResponsiblesAdd", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"added": added})
}

func (s *ServerModel) ProductionLineResponsiblesDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionLineResponsiblesDelete: ReadBody", "")
		return
	}

	id := int64(jsonMap["id"].(float64))
	err = s.Store.Repo().LineResponsibleDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionLineResponsiblesDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionFinPressDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressDelete: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	err = s.Store.Repo().FinPressComponentDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionRadiatorGetAll(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorGetAll: ReadBody", "")
		return
	}

	data, err := s.Store.Repo().RadiatorComponentsGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorGetAll", "")
		return
	}

	if rawID, ok := jsonMap["id"].(float64); ok {
		radiatorID := int(rawID)
		if radiatorID > 0 {
			for _, item := range data {
				if item.ID == radiatorID {
					sessions, err := s.Store.Repo().RadiatorPrintSessionsGetLast(radiatorID, store.LastRecordsLimit)
					if err != nil {
						s.Utils.SendError(c, err, "ProductionRadiatorGetAll: RadiatorPrintSessionsGetLast", "")
						return
					}
					s.Utils.SendOK(c, map[string]any{
						"item":          item,
						"last_sessions": sessions,
					})
					return
				}
			}
			s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionRadiatorGetAll", "")
			return
		}
	}

	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionRadiatorAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorAdd: ReadBody", "")
		return
	}

	finPressComponentID := int(jsonMap["fin_press_component_id"].(float64))
	componentID := int(jsonMap["component_id"].(float64))
	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	seriyaRaqami, _ := jsonMap["seriya_raqami"].(string)
	err = s.Store.Repo().RadiatorComponentAdd(
		componentID,
		finPressComponentID,
		c.GetInt("user_id"),
		strings.TrimSpace(seriyaRaqami),
		strings.TrimSpace(index1),
		strings.TrimSpace(index2),
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorAdd", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionRadiatorUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorUpdate: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	if id <= 0 {
		s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionRadiatorUpdate", "")
		return
	}

	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	seriyaRaqami, _ := jsonMap["seriya_raqami"].(string)
	err = s.Store.Repo().RadiatorComponentUpdateFields(
		id,
		strings.TrimSpace(seriyaRaqami),
		strings.TrimSpace(index1),
		strings.TrimSpace(index2),
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorUpdate", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionRadiatorDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorDelete: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	err = s.Store.Repo().RadiatorComponentDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionRadiatorDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func buildFinPressV2PrintData(sessionID int64, count int, component store.FinPressComponent) map[string]any {
	qrData := fmt.Sprintf("FP:%d", sessionID)
	return map[string]any{
		"serial":  qrData,
		"index_1": component.Index1,
		"index_2": component.Index2,
		"count":   count,
		"component": map[string]any{
			"factory_code":      component.FactoryCode,
			"full_name_uz":        component.FullNameUz,
			"manufacturer_code": component.ManufacturerCode,
			"standard_name_uz":  component.StandardNameUz,
			"odoo_code":         component.OdooCode,
		},
		"gscode": map[string]any{
			"data": qrData,
		},
	}
}

func (s *ServerModel) finPressSendPrintV2(
	printerV2ID int,
	sessionID int64,
	component store.FinPressComponent,
	count int,
) error {
	if printerV2ID <= 0 {
		return errors.New("printer not found")
	}
	if sessionID <= 0 {
		return errors.New("sessiya raqami noto'g'ri")
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		return err
	}
	if printerV2.LineID != store.FinPressLineID {
		return errors.New("printer fin press liniyasiga tegishli emas")
	}
	if printerV2.LabelTemplateID <= 0 {
		return errors.New("etiketka shablon tanlanmagan")
	}

	template, err := s.Store.Repo().LabelTemplateGetByID(printerV2.LabelTemplateID)
	if err != nil {
		return err
	}

	printData := buildFinPressV2PrintData(sessionID, count, component)
	return s.Utils.PrintLabelV2(template, printerV2, 1, printData)
}

func (s *ServerModel) productionFinPressPrinterIDs(jsonMap map[string]any) (printerID, printerV2ID int, err error) {
	if rawPrinterV2ID, ok := jsonMap["printer_v2_id"].(float64); ok {
		printerV2ID = int(rawPrinterV2ID)
	}
	if rawPrinterID, ok := jsonMap["printer_id"].(float64); ok {
		printerID = int(rawPrinterID)
	}
	if printerV2ID <= 0 && printerID <= 0 {
		return 0, 0, errors.New("printer tanlanmagan")
	}
	return printerID, printerV2ID, nil
}

func (s *ServerModel) finPressSendPrint(printerID int, sessionID int64) error {
	if printerID <= 0 {
		return errors.New("printer tanlanmagan")
	}
	if sessionID <= 0 {
		return errors.New("sessiya raqami noto'g'ri")
	}

	printerInfo, err := s.Store.Repo().PrinterInfoById(printerID)
	if err != nil {
		return err
	}
	if printerInfo.LineId != store.FinPressLineID {
		return errors.New("printer fin press liniyasiga tegishli emas")
	}

	qrData := fmt.Sprintf("FP:%d", sessionID)
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
				"data_input": "%s"
		}
	}`, printerInfo.FolderName, printerInfo.PrinterName, qrData))

	s.Utils.DebugLogAny("finPressSendPrint printData: ", string(printData))

	if gin.Mode() == gin.ReleaseMode {
		errString := s.Utils.PrintLabelAndWait(printData, printerInfo.Address)
		if errString != "ok" {
			return errors.New(errString)
		}
	}
	return nil
}

func (s *ServerModel) productionFinPressResolveLineID(jsonMap map[string]any) (int, error) {
	finPressLineID, err := s.Store.Repo().LinesFinPressLineID()
	if err != nil {
		return 0, err
	}
	if rawLineID, ok := jsonMap["line_id"].(float64); ok && int(rawLineID) > 0 {
		lineID := int(rawLineID)
		if lineID != finPressLineID {
			return 0, errors.New("noto'g'ri liniya")
		}
		return lineID, nil
	}
	return finPressLineID, nil
}

func (s *ServerModel) ProductionFinPressPrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint: ReadBody", "")
		return
	}

	componentID := int(jsonMap["component_id"].(float64))
	count := int(jsonMap["count"].(float64))
	printerID, printerV2ID, err := s.productionFinPressPrinterIDs(jsonMap)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint", "")
		return
	}
	if componentID <= 0 {
		s.Utils.SendError(c, errors.New("komponent tanlanmagan"), "ProductionFinPressPrint", "")
		return
	}
	if count <= 0 {
		s.Utils.SendError(c, errors.New("miqdor 0 dan katta bo'lishi kerak"), "ProductionFinPressPrint", "")
		return
	}

	finPressLineID, err := s.productionFinPressResolveLineID(jsonMap)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint", "")
		return
	}

	component, err := s.Store.Repo().FinPressComponentGetByComponentID(componentID)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint: FinPressComponentGetByComponentID", "")
		return
	}

	if err := s.enforceProductionPlan(finPressLineID, 0, componentID, count); err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint: plan", "")
		return
	}

	sessionID, err := s.Store.Repo().FinPressPrintSessionCreate(componentID, c.GetInt("user_id"), count)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint: FinPressPrintSessionCreate", "")
		return
	}

	if printerV2ID > 0 {
		err = s.finPressSendPrintV2(printerV2ID, sessionID, component, count)
	} else {
		err = s.finPressSendPrint(printerID, sessionID)
	}
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint: finPressSendPrint", "")
		return
	}

	err = s.Store.Repo().LinesBalanceUpdate(
		float64(count), finPressLineID, componentID, c.GetInt("user_id"),
		"fin_press_print", fmt.Sprintf("Fin press chop etish (sessiya #%d)", sessionID),
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressPrint: LinesBalanceUpdate", "")
		return
	}

	s.Utils.DebugLogAny("ProductionFinPressPrint session_id: ", sessionID)
	s.Utils.DebugLogAny("ProductionFinPressPrint component_id: ", componentID)
	s.Utils.DebugLogAny("ProductionFinPressPrint count: ", count)
	s.Utils.DebugLogAny("ProductionFinPressPrint printer_id: ", printerID)
	s.Utils.DebugLogAny("ProductionFinPressPrint printer_v2_id: ", printerV2ID)
	s.Utils.SendOK(c, map[string]any{
		"session_id":   sessionID,
		"component_id": componentID,
		"count":        count,
	})
}

func (s *ServerModel) ProductionFinPressSessionsLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressSessionsLast: ReadBody", "")
		return
	}

	componentID := 0
	if rawComponentID, ok := jsonMap["component_id"].(float64); ok && int(rawComponentID) > 0 {
		componentID = int(rawComponentID)
	}
	limit := store.LastRecordsLimit
	if rawLimit, ok := jsonMap["limit"].(float64); ok && int(rawLimit) > 0 {
		limit = int(rawLimit)
	}

	data, err := s.Store.Repo().FinPressPrintSessionsGetLast(componentID, limit)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressSessionsLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionFinPressReprint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressReprint: ReadBody", "")
		return
	}

	sessionID := int64(jsonMap["session_id"].(float64))
	if sessionID <= 0 {
		s.Utils.SendError(c, errors.New("sessiya raqami noto'g'ri"), "ProductionFinPressReprint", "")
		return
	}

	printerID, printerV2ID, err := s.productionFinPressPrinterIDs(jsonMap)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressReprint", "")
		return
	}

	_, err = s.productionFinPressResolveLineID(jsonMap)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressReprint", "")
		return
	}

	session, err := s.Store.Repo().FinPressPrintSessionGetByID(sessionID)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressReprint: FinPressPrintSessionGetByID", "")
		return
	}

	component, err := s.Store.Repo().FinPressComponentGetByComponentID(session.ComponentID)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressReprint: FinPressComponentGetByComponentID", "")
		return
	}

	if printerV2ID > 0 {
		err = s.finPressSendPrintV2(printerV2ID, session.ID, component, session.Count)
	} else {
		err = s.finPressSendPrint(printerID, session.ID)
	}
	if err != nil {
		s.Utils.SendError(c, err, "ProductionFinPressReprint: finPressSendPrint", "")
		return
	}

	s.Utils.DebugLogAny("ProductionFinPressReprint session_id: ", session.ID)
	s.Utils.DebugLogAny("ProductionFinPressReprint component_id: ", session.ComponentID)
	s.Utils.DebugLogAny("ProductionFinPressReprint count: ", session.Count)
	s.Utils.SendOK(c, map[string]any{
		"session_id":   session.ID,
		"component_id": session.ComponentID,
		"count":        session.Count,
	})
}
