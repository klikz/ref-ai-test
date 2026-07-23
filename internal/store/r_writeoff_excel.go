package store

import (
	"bytes"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const writeoffTemplateDataRows = 80

type writeoffExcelStyles struct {
	title    int
	meta     int
	hdrInput int
	hdrRead  int
	input    int
	readOnly int
}

func newWriteoffExcelStyles(f *excelize.File) (*writeoffExcelStyles, error) {
	s := &writeoffExcelStyles{}
	var err error
	s.title, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0F172A"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Size: 14, Color: "#FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	s.meta, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#E2E8F0"}, Pattern: 1},
		Font:      &excelize.Font{Size: 10, Color: "#334155"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	s.hdrInput, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DCFCE7"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Color: "#166534"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#86EFAC"),
	})
	if err != nil {
		return nil, err
	}
	s.hdrRead, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F1F5F9"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Color: "#475569"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#CBD5E1"),
	})
	if err != nil {
		return nil, err
	}
	s.input, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ECFDF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    planThinBorder("#BBF7D0"),
	})
	if err != nil {
		return nil, err
	}
	s.readOnly, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F1F5F9"}, Pattern: 1},
		Font:      &excelize.Font{Color: "#64748B"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#E2E8F0"),
	})
	return s, err
}

func writeoffApplySheetStyles(f *excelize.File, sheet string, styles *writeoffExcelStyles, dataStart, dataEnd int) error {
	_ = f.SetCellStyle(sheet, "A1", "H1", styles.title)
	_ = f.SetCellStyle(sheet, "A2", "A2", styles.meta)
	_ = f.SetCellStyle(sheet, "B2", "B2", styles.readOnly)

	readHdrCols := []string{"A", "C", "D"}
	inputHdrCols := []string{"B", "E", "F", "G", "H"}
	for _, col := range readHdrCols {
		_ = f.SetCellStyle(sheet, col+"3", col+"3", styles.hdrRead)
	}
	for _, col := range inputHdrCols {
		_ = f.SetCellStyle(sheet, col+"3", col+"3", styles.hdrInput)
	}

	for row := dataStart; row <= dataEnd; row++ {
		for _, col := range readHdrCols {
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), styles.readOnly)
		}
		for _, col := range inputHdrCols {
			_ = f.SetCellStyle(sheet, fmt.Sprintf("%s%d", col, row), fmt.Sprintf("%s%d", col, row), styles.input)
		}
	}
	return nil
}

type WriteoffImportResult struct {
	ImportedRows int                   `json:"imported_rows"`
	Errors       []WriteoffImportError `json:"errors"`
}

func writeoffRefSheetName(lineID int) string {
	return fmt.Sprintf("Ref_WO_L%d", lineID)
}

func writeoffWriteRefLines(f *excelize.File, lines []WriteoffCatalogItem) error {
	const sheet = "Ref_Lines"
	if err := planEnsureSheet(f, sheet); err != nil {
		return err
	}
	_ = f.SetCellValue(sheet, "A1", "line_id")
	_ = f.SetCellValue(sheet, "B1", "line_name")
	for i, line := range lines {
		row := i + 2
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), line.LineID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), line.LineName)
	}
	_ = f.SetSheetVisible(sheet, false)
	return nil
}

func writeoffWriteRefItems(f *excelize.File, lineID int, items []WriteoffCatalogItem) error {
	refSheet := writeoffRefSheetName(lineID)
	if err := planEnsureSheet(f, refSheet); err != nil {
		return err
	}
	_ = f.SetCellValue(refSheet, "A1", "item_id")
	_ = f.SetCellValue(refSheet, "B1", "label")
	_ = f.SetCellValue(refSheet, "C1", "item_type")
	for i, item := range items {
		row := i + 2
		_ = f.SetCellValue(refSheet, fmt.Sprintf("A%d", row), item.ItemID)
		_ = f.SetCellValue(refSheet, fmt.Sprintf("B%d", row), item.Label)
		_ = f.SetCellValue(refSheet, fmt.Sprintf("C%d", row), item.ItemType)
	}
	_ = f.SetSheetVisible(refSheet, false)
	return nil
}

func writeoffLineIDFormula(row int) string {
	return fmt.Sprintf(
		`IF(B%d="","",IFERROR(INDEX(Ref_Lines!$A$2:$A$20,MATCH(B%d,Ref_Lines!$B$2:$B$20,0)),""))`,
		row, row,
	)
}

func writeoffItemTypeFormula(row int) string {
	return fmt.Sprintf(
		`IF(A%d="","",IF(OR(A%d=4,A%d=5,A%d=6,A%d=7),"product","component"))`,
		row, row, row, row, row,
	)
}

func writeoffItemIDFormula(row int) string {
	return fmt.Sprintf(
		`IF(OR(A%d=4,A%d=5,A%d=6,A%d=7),"",IF(E%d="","",IFERROR(INDEX(INDIRECT("Ref_WO_L"&A%d&"!$A$2:$A$500"),MATCH(E%d,INDIRECT("Ref_WO_L"&A%d&"!$B$2:$B$500"),0)),"")))`,
		row, row, row, row, row, row, row, row,
	)
}

func writeoffApplyLineValidation(f *excelize.File, fromRow, toRow, listSize int) error {
	if toRow < fromRow || listSize < 1 {
		return nil
	}
	lastRow := listSize + 1
	dv := excelize.NewDataValidation(true)
	dv.Sqref = fmt.Sprintf("B%d:B%d", fromRow, toRow)
	dv.SetSqrefDropList(fmt.Sprintf("Ref_Lines!$B$2:$B$%d", lastRow))
	dv.SetError(excelize.DataValidationErrorStyleStop, "Noto'g'ri qiymat", "Ro'yxatdan liniya tanlang")
	return f.AddDataValidation("Writeoff", dv)
}

func writeoffApplyItemValidation(f *excelize.File, fromRow, toRow int) error {
	if toRow < fromRow {
		return nil
	}
	dv := excelize.NewDataValidation(true)
	dv.Sqref = fmt.Sprintf("E%d:E%d", fromRow, toRow)
	formula := `=IF(OR($A` + strconv.Itoa(fromRow) + `=4,$A` + strconv.Itoa(fromRow) + `=5,$A` + strconv.Itoa(fromRow) + `=6,$A` + strconv.Itoa(fromRow) + `=7),"",INDIRECT("Ref_WO_L"&$A` + strconv.Itoa(fromRow) + `&"!$B$2:$B$500"))`
	if err := dv.SetDropList([]string{formula}); err != nil {
		return err
	}
	dv.SetError(excelize.DataValidationErrorStyleStop, "Noto'g'ri qiymat", "Model yoki komponent tanlang")
	return f.AddDataValidation("Writeoff", dv)
}

func (r *Repo) WriteoffBuildWorkbook(documentID int64, items []WriteoffDocumentItem) ([]byte, error) {
	lines, err := r.WriteoffCatalogLines()
	if err != nil {
		return nil, err
	}
	catalogs := make(map[int][]WriteoffCatalogItem, len(lines))
	for _, line := range lines {
		if !IsWriteoffAuxLine(line.LineID) {
			continue
		}
		catalog, err := r.WriteoffCatalogItems(line.LineID)
		if err != nil {
			return nil, err
		}
		catalogs[line.LineID] = catalog
	}
	return buildWriteoffWorkbook(documentID, items, lines, catalogs)
}

func buildWriteoffWorkbook(
	documentID int64,
	items []WriteoffDocumentItem,
	lines []WriteoffCatalogItem,
	catalogs map[int][]WriteoffCatalogItem,
) ([]byte, error) {
	f := excelize.NewFile()
	const sheet = "Writeoff"
	_ = f.SetSheetName("Sheet1", sheet)

	if err := writeoffWriteRefLines(f, lines); err != nil {
		return nil, err
	}
	for _, line := range lines {
		if !IsWriteoffAuxLine(line.LineID) {
			continue
		}
		if err := writeoffWriteRefItems(f, line.LineID, catalogs[line.LineID]); err != nil {
			return nil, err
		}
	}

	_ = f.MergeCell(sheet, "A1", "H1")
	_ = f.SetCellValue(sheet, "A1", "Hisobdan chiqarish hujjati")
	_ = f.SetCellValue(sheet, "A2", "document_id")
	if documentID > 0 {
		_ = f.SetCellValue(sheet, "B2", documentID)
	}

	headers := []string{"line_id", "line_name", "item_type", "item_id", "item_label", "serial", "quantity", "comment"}
	for i, h := range headers {
		col, _ := excelize.ColumnNumberToName(i + 1)
		_ = f.SetCellValue(sheet, col+"3", h)
	}

	dataStart := 4
	row := dataStart
	for _, item := range items {
		_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), item.LineID)
		_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), item.LineName)
		_ = f.SetCellValue(sheet, fmt.Sprintf("C%d", row), item.ItemType)
		if item.ItemType == WriteoffItemProduct {
			_ = f.SetCellValue(sheet, fmt.Sprintf("F%d", row), item.Serial)
		} else {
			_ = f.SetCellValue(sheet, fmt.Sprintf("D%d", row), item.ComponentID)
			_ = f.SetCellValue(sheet, fmt.Sprintf("E%d", row), item.ItemLabel)
		}
		_ = f.SetCellValue(sheet, fmt.Sprintf("G%d", row), item.Quantity)
		_ = f.SetCellValue(sheet, fmt.Sprintf("H%d", row), item.Comment)
		row++
	}

	emptyEnd := dataStart + writeoffTemplateDataRows - 1
	if row > emptyEnd {
		emptyEnd = row + writeoffTemplateDataRows - 1
	}
	for rnum := dataStart; rnum <= emptyEnd; rnum++ {
		if rnum < row {
			continue
		}
		_ = f.SetCellFormula(sheet, fmt.Sprintf("A%d", rnum), writeoffLineIDFormula(rnum))
		_ = f.SetCellFormula(sheet, fmt.Sprintf("C%d", rnum), writeoffItemTypeFormula(rnum))
		_ = f.SetCellFormula(sheet, fmt.Sprintf("D%d", rnum), writeoffItemIDFormula(rnum))
	}

	if err := writeoffApplyLineValidation(f, dataStart, emptyEnd, len(lines)); err != nil {
		return nil, err
	}
	if err := writeoffApplyItemValidation(f, dataStart, emptyEnd); err != nil {
		return nil, err
	}

	styles, err := newWriteoffExcelStyles(f)
	if err != nil {
		return nil, err
	}
	if err := writeoffApplySheetStyles(f, sheet, styles, dataStart, emptyEnd); err != nil {
		return nil, err
	}

	if idx, err := f.GetSheetIndex(sheet); err == nil && idx >= 0 {
		f.SetActiveSheet(idx)
	}

	_ = f.SetColWidth(sheet, "B", "B", 18)
	_ = f.SetColWidth(sheet, "E", "E", 28)
	_ = f.SetColWidth(sheet, "F", "F", 16)
	_ = f.SetColWidth(sheet, "H", "H", 24)

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (r *Repo) WriteoffTemplateWorkbook() ([]byte, error) {
	return r.WriteoffBuildWorkbook(0, nil)
}

func (r *Repo) WriteoffExportWorkbook(documentID int64) ([]byte, error) {
	_, items, _, err := r.WriteoffDocumentGet(documentID)
	if err != nil {
		return nil, err
	}
	return r.WriteoffBuildWorkbook(documentID, items)
}

func (r *Repo) WriteoffImportWorkbook(documentID int64, data []byte) (*WriteoffImportResult, error) {
	if documentID <= 0 {
		return nil, errors.New("document_id talab qilinadi")
	}
	status, err := r.writeoffDocumentStatus(documentID)
	if err != nil {
		return nil, err
	}
	if status != WriteoffStatusDraft && status != WriteoffStatusRejected {
		return nil, errors.New("faqat qoralama yoki rad etilgan hujjat import qilinadi")
	}

	f, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sheet := "Writeoff"
	if idx, _ := f.GetSheetIndex(sheet); idx < 0 {
		sheets := f.GetSheetList()
		if len(sheets) == 0 {
			return nil, errors.New("Excel varaq topilmadi")
		}
		sheet = sheets[0]
	}

	result := &WriteoffImportResult{Errors: []WriteoffImportError{}}
	items := []WriteoffSaveItemInput{}
	serialsInDoc := map[string]struct{}{}

	for row := 4; row <= 4+writeoffTemplateDataRows+len(items); row++ {
		lineName, _ := f.GetCellValue(sheet, fmt.Sprintf("B%d", row))
		itemLabel, _ := f.GetCellValue(sheet, fmt.Sprintf("E%d", row))
		serial, _ := f.GetCellValue(sheet, fmt.Sprintf("F%d", row))
		qtyRaw, _ := f.GetCellValue(sheet, fmt.Sprintf("G%d", row), excelize.Options{RawCellValue: true})
		comment, _ := f.GetCellValue(sheet, fmt.Sprintf("H%d", row))
		itemType, _ := f.GetCellValue(sheet, fmt.Sprintf("C%d", row))

		lineName = strings.TrimSpace(lineName)
		itemLabel = strings.TrimSpace(itemLabel)
		serial = strings.TrimSpace(serial)
		comment = strings.TrimSpace(comment)
		itemType = strings.TrimSpace(strings.ToLower(itemType))

		if lineName == "" && itemLabel == "" && serial == "" && strings.TrimSpace(qtyRaw) == "" {
			continue
		}

		lineID, err := r.WriteoffResolveLineID(lineName)
		if err != nil {
			result.Errors = append(result.Errors, WriteoffImportError{Row: row, Column: "line_name", Message: err.Error()})
			continue
		}

		var modelID, componentID int
		if IsWriteoffProductLine(lineID) {
			if strings.TrimSpace(serial) == "" {
				result.Errors = append(result.Errors, WriteoffImportError{Row: row, Column: "serial", Message: "serial kiritilishi shart"})
				continue
			}
			itemType = WriteoffItemProduct
		} else {
			if itemType != "" && itemType != WriteoffItemComponent {
				result.Errors = append(result.Errors, WriteoffImportError{Row: row, Column: "item_type", Message: "komponent liniyasi uchun component kerak"})
				continue
			}
			componentID, err = r.WriteoffResolveComponentID(lineID, itemLabel)
			if err != nil {
				result.Errors = append(result.Errors, WriteoffImportError{Row: row, Column: "item_label", Message: err.Error()})
				continue
			}
			itemType = WriteoffItemComponent
		}

		qty, qtyErr := strconv.ParseFloat(strings.TrimSpace(qtyRaw), 64)
		if IsWriteoffProductLine(lineID) {
			if qtyErr != nil || qty <= 0 {
				qty = 1
			}
		} else if qtyErr != nil || qty <= 0 {
			result.Errors = append(result.Errors, WriteoffImportError{Row: row, Column: "quantity", Message: "miqdor noto'g'ri"})
			continue
		}

		item := WriteoffSaveItemInput{
			LineID:      lineID,
			ItemType:    itemType,
			ModelID:     modelID,
			ComponentID: componentID,
			Serial:      serial,
			Quantity:    qty,
			Comment:     comment,
			SortOrder:   len(items),
		}
		if err := r.writeoffValidateItemInput(&item, serialsInDoc, nil, nil); err != nil {
			result.Errors = append(result.Errors, WriteoffImportError{Row: row, Column: "", Message: err.Error()})
			continue
		}
		items = append(items, item)
	}

	if len(result.Errors) > 0 {
		return result, nil
	}
	if len(items) == 0 {
		return nil, errors.New("import qilish uchun qatorlar topilmadi")
	}
	if err := r.WriteoffDocumentItemsSave(documentID, items); err != nil {
		return nil, err
	}
	result.ImportedRows = len(items)
	return result, nil
}
