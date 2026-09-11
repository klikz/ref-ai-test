package store

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const eshikExcelSheet = "EshikModels"
const eshikExcelRefSheet = "Ref_Components"

var eshikExcelHeaders = []string{
	"id",
	"model_name",
	"freeze_component_id",
	"freeze_factory_code",
	"freeze_seriya_raqami",
	"freeze_index1",
	"freeze_index2",
	"ref_component_id",
	"ref_factory_code",
	"ref_seriya_raqami",
	"ref_index1",
	"ref_index2",
}

type EshikImportError struct {
	Row     int    `json:"row"`
	Column  string `json:"column"`
	Message string `json:"message"`
}

type EshikImportResult struct {
	ImportedRows int                `json:"imported_rows"`
	Inserted     int                `json:"inserted"`
	Updated      int                `json:"updated"`
	Errors       []EshikImportError `json:"errors"`
}

func (r *Repo) EshikModelsExportWorkbook() ([]byte, error) {
	models, err := r.EshikModelsGetAll()
	if err != nil {
		return nil, err
	}
	return r.eshikModelsBuildWorkbook(models)
}

func (r *Repo) EshikModelsTemplateWorkbook() ([]byte, error) {
	return r.eshikModelsBuildWorkbook(nil)
}

func (r *Repo) eshikModelsBuildWorkbook(models []EshikModel) ([]byte, error) {
	f := excelize.NewFile()
	defer f.Close()

	if err := f.SetSheetName("Sheet1", eshikExcelSheet); err != nil {
		return nil, err
	}
	for i, header := range eshikExcelHeaders {
		cell, _ := excelize.CoordinatesToCellName(i+1, 1)
		_ = f.SetCellValue(eshikExcelSheet, cell, header)
	}

	for i, model := range models {
		row := i + 2
		values := []any{
			model.ID,
			model.ModelName,
			model.Freeze.ComponentID,
			model.Freeze.FactoryCode,
			model.Freeze.SeriyaRaqami,
			model.Freeze.Index1,
			model.Freeze.Index2,
			model.Ref.ComponentID,
			model.Ref.FactoryCode,
			model.Ref.SeriyaRaqami,
			model.Ref.Index1,
			model.Ref.Index2,
		}
		for col, value := range values {
			cell, _ := excelize.CoordinatesToCellName(col+1, row)
			_ = f.SetCellValue(eshikExcelSheet, cell, value)
		}
	}

	if err := r.eshikWriteRefComponentsSheet(f); err != nil {
		return nil, err
	}

	_ = f.SetColWidth(eshikExcelSheet, "A", "A", 8)
	_ = f.SetColWidth(eshikExcelSheet, "B", "B", 22)
	_ = f.SetColWidth(eshikExcelSheet, "C", "L", 18)
	f.SetActiveSheet(0)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *Repo) eshikWriteRefComponentsSheet(f *excelize.File) error {
	if err := planEnsureSheet(f, eshikExcelRefSheet); err != nil {
		return err
	}
	_ = f.SetCellValue(eshikExcelRefSheet, "A1", "component_id")
	_ = f.SetCellValue(eshikExcelRefSheet, "B1", "factory_code")
	_ = f.SetCellValue(eshikExcelRefSheet, "C1", "comment")

	rows, err := r.store.db.Query(`
		SELECT c.id,
			COALESCE(NULLIF(TRIM(c.factory_code), ''), c.manufacturer_code, '') AS factory_code,
			COALESCE(c.comment, '')
		FROM production.components c
		WHERE c.status = true
		ORDER BY factory_code, c.id`)
	if err != nil {
		return err
	}
	defer rows.Close()

	rowNum := 2
	for rows.Next() {
		var id int
		var factoryCode, comment string
		if err := rows.Scan(&id, &factoryCode, &comment); err != nil {
			return err
		}
		_ = f.SetCellValue(eshikExcelRefSheet, fmt.Sprintf("A%d", rowNum), id)
		_ = f.SetCellValue(eshikExcelRefSheet, fmt.Sprintf("B%d", rowNum), factoryCode)
		_ = f.SetCellValue(eshikExcelRefSheet, fmt.Sprintf("C%d", rowNum), comment)
		rowNum++
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_ = f.SetColWidth(eshikExcelRefSheet, "A", "A", 14)
	_ = f.SetColWidth(eshikExcelRefSheet, "B", "B", 22)
	_ = f.SetColWidth(eshikExcelRefSheet, "C", "C", 40)
	return nil
}

func (r *Repo) EshikModelsImportWorkbook(data []byte, userID int) (*EshikImportResult, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheet := f.GetSheetName(0)
	if sheet == "" {
		return nil, errors.New("excel sheet topilmadi")
	}
	for _, name := range f.GetSheetList() {
		if strings.EqualFold(name, eshikExcelSheet) {
			sheet = name
			break
		}
	}

	rows, err := f.GetRows(sheet)
	if err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return nil, errors.New("fayl bo'sh")
	}

	colIndex := eshikExcelHeaderIndex(rows[0])
	if _, ok := colIndex["model_name"]; !ok {
		return nil, errors.New("model_name ustuni topilmadi")
	}

	result := &EshikImportResult{Errors: []EshikImportError{}}
	existingByName, err := r.eshikModelIDsByName()
	if err != nil {
		return nil, err
	}

	for i := 1; i < len(rows); i++ {
		row := rows[i]
		rowNum := i + 1
		if eshikExcelRowEmpty(row) {
			continue
		}

		modelName := strings.TrimSpace(eshikExcelCell(row, colIndex, "model_name"))
		if modelName == "" {
			result.Errors = append(result.Errors, EshikImportError{Row: rowNum, Column: "model_name", Message: "model nomi bo'sh"})
			continue
		}

		freeze, freezeErr := r.eshikResolvePartInput(row, colIndex, "freeze")
		if freezeErr != nil {
			result.Errors = append(result.Errors, EshikImportError{Row: rowNum, Column: freezeErr.column, Message: freezeErr.message})
			continue
		}
		ref, refErr := r.eshikResolvePartInput(row, colIndex, "ref")
		if refErr != nil {
			result.Errors = append(result.Errors, EshikImportError{Row: rowNum, Column: refErr.column, Message: refErr.message})
			continue
		}
		if freeze.ComponentID == ref.ComponentID {
			result.Errors = append(result.Errors, EshikImportError{
				Row:     rowNum,
				Column:  "ref_component_id",
				Message: "freeze va ref uchun bir xil komponent tanlab bo'lmaydi",
			})
			continue
		}

		id := eshikExcelInt(row, colIndex, "id")
		if id <= 0 {
			if existingID, ok := existingByName[strings.ToLower(modelName)]; ok {
				id = existingID
			}
		}

		if id > 0 {
			if err := r.EshikModelUpdate(id, modelName, freeze, ref); err != nil {
				result.Errors = append(result.Errors, EshikImportError{Row: rowNum, Column: "", Message: err.Error()})
				continue
			}
			result.Updated++
			existingByName[strings.ToLower(modelName)] = id
		} else {
			if err := r.EshikModelAdd(modelName, userID, freeze, ref); err != nil {
				result.Errors = append(result.Errors, EshikImportError{Row: rowNum, Column: "", Message: err.Error()})
				continue
			}
			result.Inserted++
			// Refresh name map for subsequent rows that might update by name.
			if refreshed, err := r.eshikModelIDsByName(); err == nil {
				existingByName = refreshed
			}
		}
		result.ImportedRows++
	}

	if result.ImportedRows == 0 && len(result.Errors) == 0 {
		return nil, errors.New("faylda import qilinadigan qatorlar topilmadi")
	}
	return result, nil
}

type eshikPartResolveError struct {
	column  string
	message string
}

func (r *Repo) eshikResolvePartInput(row []string, colIndex map[string]int, prefix string) (EshikPartInput, *eshikPartResolveError) {
	componentID := eshikExcelInt(row, colIndex, prefix+"_component_id")
	factoryCode := strings.TrimSpace(eshikExcelCell(row, colIndex, prefix+"_factory_code"))
	seriya := strings.TrimSpace(eshikExcelCell(row, colIndex, prefix+"_seriya_raqami"))
	index1 := strings.TrimSpace(eshikExcelCell(row, colIndex, prefix+"_index1"))
	index2 := strings.TrimSpace(eshikExcelCell(row, colIndex, prefix+"_index2"))

	if componentID <= 0 && factoryCode != "" {
		resolved, err := r.eshikComponentIDByFactoryCode(factoryCode)
		if err != nil {
			return EshikPartInput{}, &eshikPartResolveError{column: prefix + "_factory_code", message: err.Error()}
		}
		componentID = resolved
	}
	if componentID <= 0 {
		return EshikPartInput{}, &eshikPartResolveError{
			column:  prefix + "_component_id",
			message: "komponent id yoki factory_code kerak",
		}
	}
	if seriya == "" {
		return EshikPartInput{}, &eshikPartResolveError{
			column:  prefix + "_seriya_raqami",
			message: "serial prefix bo'sh",
		}
	}
	return EshikPartInput{
		ComponentID:  componentID,
		SeriyaRaqami: seriya,
		Index1:       index1,
		Index2:       index2,
	}, nil
}

func (r *Repo) eshikComponentIDByFactoryCode(factoryCode string) (int, error) {
	factoryCode = strings.TrimSpace(factoryCode)
	if factoryCode == "" {
		return 0, errors.New("factory_code bo'sh")
	}
	var id int
	err := r.store.db.QueryRow(`
		SELECT id
		FROM production.components
		WHERE status = true
		  AND (
			LOWER(TRIM(COALESCE(factory_code, ''))) = LOWER($1)
			OR LOWER(TRIM(COALESCE(manufacturer_code, ''))) = LOWER($1)
		  )
		ORDER BY id
		LIMIT 1`, factoryCode).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("komponent topilmadi: %s", factoryCode)
	}
	return id, nil
}

func (r *Repo) eshikModelIDsByName() (map[string]int, error) {
	rows, err := r.store.db.Query(`SELECT id, model_name FROM production.eshik_models`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := map[string]int{}
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		out[strings.ToLower(strings.TrimSpace(name))] = id
	}
	return out, rows.Err()
}

func eshikExcelHeaderIndex(header []string) map[string]int {
	out := map[string]int{}
	for i, cell := range header {
		key := strings.ToLower(strings.TrimSpace(cell))
		if key == "" {
			continue
		}
		out[key] = i
	}
	return out
}

func eshikExcelCell(row []string, colIndex map[string]int, key string) string {
	idx, ok := colIndex[key]
	if !ok || idx < 0 || idx >= len(row) {
		return ""
	}
	return row[idx]
}

func eshikExcelInt(row []string, colIndex map[string]int, key string) int {
	raw := strings.TrimSpace(eshikExcelCell(row, colIndex, key))
	if raw == "" {
		return 0
	}
	raw = strings.TrimSuffix(raw, ".0")
	n, err := strconv.Atoi(raw)
	if err != nil {
		f, ferr := strconv.ParseFloat(raw, 64)
		if ferr != nil {
			return 0
		}
		return int(f)
	}
	return n
}

func eshikExcelRowEmpty(row []string) bool {
	for _, cell := range row {
		if strings.TrimSpace(cell) != "" {
			return false
		}
	}
	return true
}
