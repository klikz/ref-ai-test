package store

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/xuri/excelize/v2"
)

const planTemplateDataRows = 80

type planExcelStyles struct {
	title     int
	meta      int
	dateHdr   int
	rejaHdr   int
	faktHdr   int
	labelCol  int
	idCol     int
	rejaCell  int
	faktCell  int
	sozHdr    int
}

func newPlanExcelStyles(f *excelize.File) (*planExcelStyles, error) {
	s := &planExcelStyles{}
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
	s.dateHdr, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#0891B2"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	if err != nil {
		return nil, err
	}
	s.rejaHdr, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#DCFCE7"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Color: "#166534"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#86EFAC"),
	})
	if err != nil {
		return nil, err
	}
	s.faktHdr, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F1F5F9"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Color: "#475569"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#CBD5E1"),
	})
	if err != nil {
		return nil, err
	}
	s.labelCol, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#FEF9C3"}, Pattern: 1},
		Font:      &excelize.Font{Color: "#713F12"},
		Alignment: &excelize.Alignment{Horizontal: "left", Vertical: "center"},
		Border:    planThinBorder("#FDE047"),
	})
	if err != nil {
		return nil, err
	}
	s.idCol, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F8FAFC"}, Pattern: 1},
		Font:      &excelize.Font{Color: "#64748B"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#E2E8F0"),
	})
	if err != nil {
		return nil, err
	}
	s.rejaCell, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#ECFDF5"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#BBF7D0"),
	})
	if err != nil {
		return nil, err
	}
	s.faktCell, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#F8FAFC"}, Pattern: 1},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
		Border:    planThinBorder("#E2E8F0"),
	})
	if err != nil {
		return nil, err
	}
	s.sozHdr, err = f.NewStyle(&excelize.Style{
		Fill:      excelize.Fill{Type: "pattern", Color: []string{"#7C3AED"}, Pattern: 1},
		Font:      &excelize.Font{Bold: true, Color: "#FFFFFF"},
		Alignment: &excelize.Alignment{Horizontal: "center", Vertical: "center"},
	})
	return s, err
}

func planThinBorder(color string) []excelize.Border {
	return []excelize.Border{
		{Type: "left", Color: color, Style: 1},
		{Type: "right", Color: color, Style: 1},
		{Type: "top", Color: color, Style: 1},
		{Type: "bottom", Color: color, Style: 1},
	}
}

func planRefSheetName(lineID int) string {
	return fmt.Sprintf("Ref_L%d", lineID)
}

func planEnsureSheet(f *excelize.File, sheet string) error {
	if idx, _ := f.GetSheetIndex(sheet); idx >= 0 {
		return nil
	}
	_, err := f.NewSheet(sheet)
	return err
}

func (r *Repo) writePlanRefSheet(f *excelize.File, lineID int, items []planTemplateItem) error {
	refSheet := planRefSheetName(lineID)
	if err := planEnsureSheet(f, refSheet); err != nil {
		return err
	}
	_ = f.SetCellValue(refSheet, "A1", "item_id")
	_ = f.SetCellValue(refSheet, "B1", "label")
	for i, item := range items {
		row := i + 2
		_ = f.SetCellValue(refSheet, fmt.Sprintf("A%d", row), item.ItemKey)
		_ = f.SetCellValue(refSheet, fmt.Sprintf("B%d", row), item.Label)
	}
	_ = f.SetSheetVisible(refSheet, false)
	return nil
}

func (r *Repo) applyPlanLabelValidation(f *excelize.File, sheet, refSheet string, fromRow, toRow, listSize int) error {
	if toRow < fromRow {
		return nil
	}
	lastRow := listSize + 1
	if lastRow < 2 {
		lastRow = 2
	}
	dv := excelize.NewDataValidation(true)
	dv.Sqref = fmt.Sprintf("B%d:B%d", fromRow, toRow)
	dv.SetSqrefDropList(fmt.Sprintf("%s!$B$2:$B$%d", refSheet, lastRow))
	dv.SetError(excelize.DataValidationErrorStyleStop, "Noto'g'ri qiymat", "Ro'yxatdan model yoki komponent tanlang")
	return f.AddDataValidation(sheet, dv)
}

func planItemIDFormula(refSheet string, row int) string {
	return fmt.Sprintf(
		`IF(B%d="","",IFERROR(INDEX(%s!$A$2:$A$500,MATCH(B%d,%s!$B$2:$B$500,0)),""))`,
		row, refSheet, row, refSheet,
	)
}

func (r *Repo) writePlanLineSheet(
	f *excelize.File,
	styles *planExcelStyles,
	lineID int,
	lineName, yearMonth string,
	daysInMonth int,
	templateItems []planTemplateItem,
	planMap map[int]map[string]planMonthCell,
	includeActual bool,
) error {
	sheet := sanitizeSheetName(lineName)
	if err := planEnsureSheet(f, sheet); err != nil {
		return err
	}
	refSheet := planRefSheetName(lineID)

	if err := r.writePlanRefSheet(f, lineID, templateItems); err != nil {
		return err
	}

	lastCol, _ := excelize.ColumnNumberToName(2 + daysInMonth*4)
	_ = f.MergeCell(sheet, "A1", lastCol+"1")
	_ = f.SetCellValue(sheet, "A1", fmt.Sprintf("%s — %s", lineName, yearMonth))
	_ = f.SetCellStyle(sheet, "A1", lastCol+"1", styles.title)

	_ = f.SetCellValue(sheet, "A2", "line_id")
	_ = f.SetCellValue(sheet, "B2", lineID)
	_ = f.SetCellStyle(sheet, "A2", "B2", styles.meta)
	_ = f.SetCellValue(sheet, "A3", "item_id")
	_ = f.SetCellValue(sheet, "B3", "label")
	_ = f.SetCellStyle(sheet, "A3", "B3", styles.meta)

	col := 3
	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%s-%02d", yearMonth, day)
		s1PlanCol, _ := excelize.ColumnNumberToName(col)
		s1FaktCol, _ := excelize.ColumnNumberToName(col + 1)
		s2PlanCol, _ := excelize.ColumnNumberToName(col + 2)
		s2FaktCol, _ := excelize.ColumnNumberToName(col + 3)
		_ = f.SetCellValue(sheet, s1PlanCol+"2", dateStr)
		_ = f.MergeCell(sheet, s1PlanCol+"2", s2FaktCol+"2")
		_ = f.SetCellStyle(sheet, s1PlanCol+"2", s2FaktCol+"2", styles.dateHdr)
		_ = f.SetCellValue(sheet, s1PlanCol+"3", "S1 Reja")
		_ = f.SetCellStyle(sheet, s1PlanCol+"3", s1PlanCol+"3", styles.rejaHdr)
		_ = f.SetCellValue(sheet, s1FaktCol+"3", "S1 Fakt")
		_ = f.SetCellStyle(sheet, s1FaktCol+"3", s1FaktCol+"3", styles.faktHdr)
		_ = f.SetCellValue(sheet, s2PlanCol+"3", "S2 Reja")
		_ = f.SetCellStyle(sheet, s2PlanCol+"3", s2PlanCol+"3", styles.rejaHdr)
		_ = f.SetCellValue(sheet, s2FaktCol+"3", "S2 Fakt")
		_ = f.SetCellStyle(sheet, s2FaktCol+"3", s2FaktCol+"3", styles.faktHdr)
		col += 4
	}

	dataRowKeys := []int{}
	for key := range planMap {
		dataRowKeys = append(dataRowKeys, key)
	}
	sort.Ints(dataRowKeys)

	row := 4
	for _, key := range dataRowKeys {
		label, err := r.productionPlanItemLabel(lineID, key)
		if err != nil {
			continue
		}
		if err := r.fillPlanDataRow(f, styles, sheet, refSheet, lineID, row, key, label, yearMonth, daysInMonth, planMap[key], includeActual); err != nil {
			return err
		}
		row++
	}

	emptyStart := row
	emptyEnd := row + planTemplateDataRows - 1
	if emptyEnd < 4 {
		emptyEnd = 4 + planTemplateDataRows - 1
	}
	for rnum := emptyStart; rnum <= emptyEnd; rnum++ {
		_ = f.SetCellFormula(sheet, fmt.Sprintf("A%d", rnum), planItemIDFormula(refSheet, rnum))
		_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", rnum), fmt.Sprintf("A%d", rnum), styles.idCol)
		_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", rnum), fmt.Sprintf("B%d", rnum), styles.labelCol)
		col = 3
		for day := 1; day <= daysInMonth; day++ {
			s1PlanCol, _ := excelize.ColumnNumberToName(col)
			s1FaktCol, _ := excelize.ColumnNumberToName(col + 1)
			s2PlanCol, _ := excelize.ColumnNumberToName(col + 2)
			s2FaktCol, _ := excelize.ColumnNumberToName(col + 3)
			_ = f.SetCellStyle(sheet, s1PlanCol+strconv.Itoa(rnum), s1PlanCol+strconv.Itoa(rnum), styles.rejaCell)
			_ = f.SetCellStyle(sheet, s1FaktCol+strconv.Itoa(rnum), s1FaktCol+strconv.Itoa(rnum), styles.faktCell)
			_ = f.SetCellStyle(sheet, s2PlanCol+strconv.Itoa(rnum), s2PlanCol+strconv.Itoa(rnum), styles.rejaCell)
			_ = f.SetCellStyle(sheet, s2FaktCol+strconv.Itoa(rnum), s2FaktCol+strconv.Itoa(rnum), styles.faktCell)
			col += 4
		}
	}

	if err := r.applyPlanLabelValidation(f, sheet, refSheet, emptyStart, emptyEnd, len(templateItems)); err != nil {
		return err
	}
	if emptyStart > 4 {
		_ = r.applyPlanLabelValidation(f, sheet, refSheet, 4, emptyStart-1, len(templateItems))
	}

	_ = f.SetColWidth(sheet, "A", "A", 10)
	_ = f.SetColWidth(sheet, "B", "B", 24)
	_ = f.SetPanes(sheet, &excelize.Panes{
		Freeze:      true,
		XSplit:      2,
		YSplit:      3,
		TopLeftCell: "C4",
		ActivePane:  "bottomRight",
	})
	return nil
}

func (r *Repo) fillPlanDataRow(
	f *excelize.File,
	styles *planExcelStyles,
	sheet, refSheet string,
	lineID, row, itemKey int,
	label, yearMonth string,
	daysInMonth int,
	dayCells map[string]planMonthCell,
	includeActual bool,
) error {
	_ = f.SetCellValue(sheet, fmt.Sprintf("A%d", row), itemKey)
	_ = f.SetCellValue(sheet, fmt.Sprintf("B%d", row), label)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("A%d", row), fmt.Sprintf("A%d", row), styles.idCol)
	_ = f.SetCellStyle(sheet, fmt.Sprintf("B%d", row), fmt.Sprintf("B%d", row), styles.labelCol)

	col := 3
	for day := 1; day <= daysInMonth; day++ {
		dateStr := fmt.Sprintf("%s-%02d", yearMonth, day)
		s1PlanCol, _ := excelize.ColumnNumberToName(col)
		s1FaktCol, _ := excelize.ColumnNumberToName(col + 1)
		s2PlanCol, _ := excelize.ColumnNumberToName(col + 2)
		s2FaktCol, _ := excelize.ColumnNumberToName(col + 3)
		if cell, ok := dayCells[dateStr]; ok {
			if cell.Shift1Planned > 0 {
				_ = f.SetCellValue(sheet, s1PlanCol+strconv.Itoa(row), cell.Shift1Planned)
			}
			if cell.Shift2Planned > 0 {
				_ = f.SetCellValue(sheet, s2PlanCol+strconv.Itoa(row), cell.Shift2Planned)
			}
		}
		_ = f.SetCellStyle(sheet, s1PlanCol+strconv.Itoa(row), s1PlanCol+strconv.Itoa(row), styles.rejaCell)
		_ = f.SetCellStyle(sheet, s2PlanCol+strconv.Itoa(row), s2PlanCol+strconv.Itoa(row), styles.rejaCell)
		if includeActual {
			for _, shiftNo := range []int{1, 2} {
				actual := 0
				var err error
				if lineID == EshikLineID {
					actual, err = r.ProductionPlanEshikPairActualQty(dateStr, shiftNo, itemKey)
				} else {
					modelID, componentID := 0, 0
					if IsProductionPlanProductLine(lineID) {
						modelID = itemKey
					} else {
						componentID = itemKey
					}
					actual, err = r.ProductionPlanActualQty(dateStr, lineID, shiftNo, modelID, componentID)
				}
				if err != nil {
					return err
				}
				if actual > 0 {
					faktCol := s1FaktCol
					if shiftNo == 2 {
						faktCol = s2FaktCol
					}
					_ = f.SetCellValue(sheet, faktCol+strconv.Itoa(row), actual)
				}
			}
		}
		_ = f.SetCellStyle(sheet, s1FaktCol+strconv.Itoa(row), s1FaktCol+strconv.Itoa(row), styles.faktCell)
		_ = f.SetCellStyle(sheet, s2FaktCol+strconv.Itoa(row), s2FaktCol+strconv.Itoa(row), styles.faktCell)
		col += 4
	}
	_ = refSheet
	return nil
}

func (r *Repo) productionPlanResolveLabel(lineID int, label string) (int, error) {
	label = strings.TrimSpace(label)
	if label == "" {
		return 0, errors.New("label bo'sh")
	}
	if IsProductionPlanProductLine(lineID) {
		char, hasChar := productionPlanModelSeriyaSuffix(lineID)
		query := `
			SELECT id FROM production.models
			WHERE status = true AND deleted = false
			  AND LOWER(TRIM(modeli)) = LOWER($1)`
		args := []any{label}
		if hasChar {
			query += productionPlanSeriyaEndsWithClause(2)
			args = append(args, char)
		}
		return r.productionPlanResolveUniqueID(query, args, "model", label)
	}

	labelExpr := `LOWER(TRIM(COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, '')))`
	switch lineID {
	case FinPressLineID:
		return r.productionPlanResolveUniqueID(`
			SELECT c.id
			FROM production.fin_press_components fpc
			INNER JOIN production.components c ON c.id = fpc.component_id
			WHERE c.status = true AND `+labelExpr+` = LOWER($1)`,
			[]any{label}, "komponent", label)
	case RadiatorLineID:
		return r.productionPlanResolveUniqueID(`
			SELECT cr.id
			FROM production.radiator_components rc
			INNER JOIN production.components cr ON cr.id = rc.component_id
			WHERE cr.status = true AND LOWER(TRIM(COALESCE(NULLIF(cr.factory_code, ''), cr.manufacturer_code, ''))) = LOWER($1)`,
			[]any{label}, "komponent", label)
	case KlapanLineID:
		return r.productionPlanResolveUniqueID(`
			SELECT c.id
			FROM production.klapan_components kc
			INNER JOIN production.components c ON c.id = kc.component_id
			WHERE c.status = true AND LOWER(TRIM(COALESCE(NULLIF(c.factory_code, ''), c.manufacturer_code, ''))) = LOWER($1)`,
			[]any{label}, "komponent", label)
	case EshikLineID:
		return r.productionPlanResolveUniqueID(`
			SELECT m.id
			FROM production.eshik_models m
			WHERE LOWER(TRIM(m.model_name)) = LOWER($1)`,
			[]any{label}, "model", label)
	default:
		return 0, fmt.Errorf("noto'g'ri line_id: %d", lineID)
	}
}

func (r *Repo) productionPlanResolveUniqueID(query string, args []any, kind, label string) (int, error) {
	rows, err := r.store.db.Query(query, args...)
	if err != nil {
		return 0, err
	}
	defer rows.Close()
	ids := []int{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return 0, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, err
	}
	if len(ids) == 0 {
		return 0, fmt.Errorf("%s topilmadi: %s", kind, label)
	}
	if len(ids) > 1 {
		return 0, fmt.Errorf("%s noaniq (bir nechta): %s", kind, label)
	}
	return ids[0], nil
}
