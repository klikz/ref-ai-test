package api

import (
	"errors"
	"io"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func (s *ServerModel) enforceProductionPlan(lineID, modelID, componentID, qty int) error {
	return s.Store.Repo().ProductionPlanAssertCanProduce(lineID, modelID, componentID, qty)
}

func (s *ServerModel) checkProductionPlanAllowMissing(lineID, modelID, componentID, qty int) (string, error) {
	return s.Store.Repo().ProductionPlanCheckCanProduceAllowMissing(lineID, modelID, componentID, qty)
}

func (s *ServerModel) ProductionPlanDay(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanDay: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	planDate, _ := jsonMap["plan_date"].(string)
	shiftNo := 0
	if v, ok := jsonMap["shift_no"].(float64); ok {
		shiftNo = int(v)
	}
	currentShift := false
	if v, ok := jsonMap["current_shift"].(bool); ok {
		currentShift = v
	}
	var current store.ProductionCurrentShiftInfo
	if currentShift {
		current, err = s.Store.Repo().ProductionCurrentShift(time.Time{})
		if err != nil {
			s.Utils.SendError(c, err, "ProductionPlanDay: current shift", "")
			return
		}
		if current.ShiftNo > 0 {
			shiftNo = current.ShiftNo
		}
		if strings.TrimSpace(planDate) == "" && current.PlanDate != "" {
			planDate = current.PlanDate
		}
	}
	items, status, err := s.Store.Repo().ProductionPlanGetDay(planDate, lineID, shiftNo)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanDay", "")
		return
	}
	if !currentShift {
		current, _ = s.Store.Repo().ProductionCurrentShift(time.Time{})
	}
	s.Utils.SendOK(c, map[string]any{
		"plan_date":         strings.TrimSpace(planDate),
		"line_id":           lineID,
		"status":            status,
		"shift_no":          shiftNo,
		"current_shift":     current,
		"items":             items,
	})
}

func (s *ServerModel) ProductionPlanReport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanReport: ReadBody", "")
		return
	}
	dateFrom, _ := jsonMap["date_from"].(string)
	dateTo, _ := jsonMap["date_to"].(string)
	lineIDs := parseRequestLineIDs(jsonMap)
	items, err := s.Store.Repo().ProductionPlanReport(dateFrom, dateTo, lineIDs)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanReport", "")
		return
	}
	s.Utils.SendOK(c, items)
}

func (s *ServerModel) ProductionPlanDashboard(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanDashboard: ReadBody", "")
		return
	}
	planDate, _ := jsonMap["plan_date"].(string)
	data, err := s.Store.Repo().ProductionPlanDashboard(planDate)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanDashboard", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionPlanLockDay(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanLockDay: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	planDate, _ := jsonMap["plan_date"].(string)
	if err := s.Store.Repo().ProductionPlanLockDay(planDate, lineID, c.GetInt("user_id")); err != nil {
		s.Utils.SendError(c, err, "ProductionPlanLockDay", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionPlanLockMonth(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanLockMonth: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	yearMonth, _ := jsonMap["year_month"].(string)
	locked, err := s.Store.Repo().ProductionPlanLockMonth(yearMonth, lineID, c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanLockMonth", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"locked_days": locked})
}

func (s *ServerModel) ProductionPlanUnlockDay(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanUnlockDay: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	planDate, _ := jsonMap["plan_date"].(string)
	if err := s.Store.Repo().ProductionPlanUnlockDay(planDate, lineID); err != nil {
		s.Utils.SendError(c, err, "ProductionPlanUnlockDay", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionPlanUnlockMonth(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanUnlockMonth: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	yearMonth, _ := jsonMap["year_month"].(string)
	unlocked, err := s.Store.Repo().ProductionPlanUnlockMonth(yearMonth, lineID)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanUnlockMonth", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"unlocked_days": unlocked})
}

func (s *ServerModel) ProductionPlanAllowedModels(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanAllowedModels: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	planDate, _ := jsonMap["plan_date"].(string)
	ids, err := s.Store.Repo().ProductionPlanAllowedModelIDs(planDate, lineID)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanAllowedModels", "")
		return
	}
	status, err := s.Store.Repo().ProductionPlanDayStatus(planDate, lineID)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanAllowedModels", "")
		return
	}
	s.Utils.SendOK(c, map[string]any{
		"status":    status,
		"model_ids": ids,
	})
}

func (s *ServerModel) ProductionPlanMonth(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanMonth: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	yearMonth, _ := jsonMap["year_month"].(string)
	includeActual := true
	if val, ok := jsonMap["include_actual"].(bool); ok {
		includeActual = val
	}
	data, err := s.Store.Repo().ProductionPlanGetMonth(yearMonth, lineID, includeActual)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanMonth", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionPlanSave(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanSave: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	rawCells, ok := jsonMap["cells"].([]any)
	if !ok || len(rawCells) == 0 {
		s.Utils.SendError(c, errors.New("cells talab qilinadi"), "ProductionPlanSave", "")
		return
	}
	cells := make([]store.PlanUpsertCell, 0, len(rawCells))
	for _, raw := range rawCells {
		m, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		cell := store.PlanUpsertCell{LineID: lineID, AllowOverplan: true, ShiftNo: 1}
		if v, ok := m["plan_date"].(string); ok {
			cell.PlanDate = v
		}
		if v, ok := m["model_id"].(float64); ok {
			cell.ModelID = int(v)
		}
		if v, ok := m["component_id"].(float64); ok {
			cell.ComponentID = int(v)
		}
		if v, ok := m["shift_no"].(float64); ok {
			cell.ShiftNo = int(v)
		}
		if v, ok := m["planned_qty"].(float64); ok {
			cell.PlannedQty = int(v)
		}
		if v, ok := m["allow_overplan"].(bool); ok {
			cell.AllowOverplan = v
		}
		if cell.LineID <= 0 {
			if v, ok := m["line_id"].(float64); ok {
				cell.LineID = int(v)
			}
		}
		if cell.PlanDate == "" {
			continue
		}
		if cell.ModelID <= 0 && cell.ComponentID <= 0 {
			s.Utils.SendError(c, errors.New("har bir katak uchun model yoki komponent kerak"), "ProductionPlanSave", "")
			return
		}
		cells = append(cells, cell)
	}
	if err := s.Store.Repo().ProductionPlanSaveCells(cells); err != nil {
		s.Utils.SendError(c, err, "ProductionPlanSave", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"saved_cells": len(cells)})
}

func (s *ServerModel) ProductionPlanExport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanExport: ReadBody", "")
		return
	}
	yearMonth, _ := jsonMap["year_month"].(string)
	includeActual := false
	if val, ok := jsonMap["include_actual"].(bool); ok {
		includeActual = val
	}
	lineIDs := parseRequestLineIDs(jsonMap)
	data, err := s.Store.Repo().ProductionPlanExportWorkbook(yearMonth, lineIDs, includeActual)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanExport", "")
		return
	}
	filename := "ishlab_chiqarish_rejasi_" + strings.TrimSpace(yearMonth) + ".xlsx"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (s *ServerModel) ProductionPlanImport(c *gin.Context) {
	yearMonth := strings.TrimSpace(c.PostForm("year_month"))
	if yearMonth == "" {
		s.Utils.SendError(c, errors.New("year_month talab qilinadi"), "ProductionPlanImport", "")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanImport: FormFile", "")
		return
	}
	opened, err := file.Open()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanImport: Open", "")
		return
	}
	defer opened.Close()
	data, err := io.ReadAll(opened)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanImport: ReadAll", "")
		return
	}
	result, err := s.Store.Repo().ProductionPlanImportWorkbook(data, yearMonth)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionPlanImport", "")
		return
	}
	if len(result.Errors) > 0 {
		s.Utils.SendError(c, errors.New("import xatoliklari"), "ProductionPlanImport", result)
		return
	}
	s.Utils.SendOK(c, result)
}
