package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func buildEshikV2PrintData(component store.EshikComponent, serial string, counter int) map[string]any {
	return map[string]any{
		"serial":        serial,
		"seriya_raqami": component.SeriyaRaqami,
		"eshik_counter": counter,
		"index_1":       component.Index1,
		"index_2":       component.Index2,
		"eshik": map[string]any{
			"index1": component.Index1,
			"index2": component.Index2,
		},
		"component": map[string]any{
			"factory_code":      component.FactoryCode,
			"full_name_uz":      component.FullNameUz,
			"manufacturer_code": component.ManufacturerCode,
			"standard_name_uz":  component.StandardNameUz,
			"odoo_code":         component.OdooCode,
		},
		"gscode": map[string]any{
			"data": serial,
		},
	}
}

func (s *ServerModel) eshikSendPrintV2(
	printerV2ID int,
	component store.EshikComponent,
	serial string,
	counter, copyCount int,
) error {
	if printerV2ID <= 0 {
		return errors.New("printer not found")
	}
	if copyCount <= 0 {
		copyCount = 1
	}
	if strings.TrimSpace(serial) == "" {
		return errors.New("serial bo'sh")
	}

	printerV2, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		return err
	}
	if printerV2.LineID != store.EshikLineID {
		return errors.New("printer eshik liniyasiga tegishli emas")
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

	printData := buildEshikV2PrintData(component, serial, counter)
	return s.Utils.PrintLabelV2(template, printerV2, copyCount, printData)
}

type eshikPrintAttempt struct {
	sessionID          int64
	auxiliaryProductID int64
	eshikComponentID   int
	componentID        int
	counter            int
}

func (s *ServerModel) rollbackEshikPrintAttempt(lineID, userID int, attempt eshikPrintAttempt) {
	if attempt.sessionID <= 0 && attempt.auxiliaryProductID <= 0 {
		return
	}
	_ = s.Store.Repo().EshikPrintRollback(
		lineID,
		attempt.sessionID,
		attempt.eshikComponentID,
		attempt.componentID,
		attempt.auxiliaryProductID,
		userID,
	)
}

func (s *ServerModel) LinesEshikV2Print(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: ReadBody", "")
		return
	}

	eshikComponentID := int(jsonMap["eshik_component_id"].(float64))
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	copyCount := 1
	if rawCopy, ok := jsonMap["copy"].(float64); ok && int(rawCopy) > 0 {
		copyCount = int(rawCopy)
	}
	if copyCount > 3 {
		copyCount = 3
	}
	userID := c.GetInt("user_id")

	if eshikComponentID <= 0 {
		s.Utils.SendError(c, errors.New("komponent tanlanmagan"), "LinesEshikV2Print", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesEshikV2Print", "")
		return
	}

	component, err := s.Store.Repo().EshikComponentGetByID(eshikComponentID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: EshikComponentGetByID", "")
		return
	}

	eshikLineID, err := s.Store.Repo().LinesEshikLineID()
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: LinesEshikLineID", "")
		return
	}

	if err := s.enforceProductionPlan(eshikLineID, 0, component.ComponentID, 1); err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: plan", "")
		return
	}

	counter, err := s.Store.Repo().EshikNextDailyCounter(eshikComponentID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: EshikNextDailyCounter", "")
		return
	}

	serial, err := s.GenerateSerial(strings.TrimSpace(component.SeriyaRaqami), counter)
	if err != nil {
		_ = s.Store.Repo().EshikDailyCounterUndo(eshikComponentID)
		s.Utils.SendError(c, err, "LinesEshikV2Print: GenerateSerial", "")
		return
	}

	attempt := eshikPrintAttempt{
		eshikComponentID: eshikComponentID,
		componentID:    component.ComponentID,
		counter:        counter,
	}

	sessionID, err := s.Store.Repo().EshikPrintSessionCreate(component, serial, counter, userID)
	if err != nil {
		_ = s.Store.Repo().EshikDailyCounterUndo(eshikComponentID)
		s.Utils.SendError(c, err, "LinesEshikV2Print: EshikPrintSessionCreate", "")
		return
	}
	attempt.sessionID = sessionID

	auxiliaryProductID, err := s.Store.Repo().EshikChiqishBalanceApply(
		eshikLineID,
		component.ComponentID,
		userID,
		sessionID,
		serial,
	)
	if err != nil {
		s.rollbackEshikPrintAttempt(eshikLineID, userID, attempt)
		s.Utils.SendError(c, err, "LinesEshikV2Print: EshikChiqishBalanceApply", "")
		return
	}
	attempt.auxiliaryProductID = auxiliaryProductID

	if err := s.eshikSendPrintV2(printerV2ID, component, serial, counter, copyCount); err != nil {
		s.rollbackEshikPrintAttempt(eshikLineID, userID, attempt)
		s.Utils.SendError(c, err, "LinesEshikV2Print: eshikSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id":           sessionID,
		"serial":               serial,
		"counter":              counter,
		"auxiliary_product_id": auxiliaryProductID,
	})
}

func (s *ServerModel) LinesEshikV2Reprint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Reprint: ReadBody", "")
		return
	}

	sessionID := int64(jsonMap["session_id"].(float64))
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	copyCount := 1
	if rawCopy, ok := jsonMap["copy"].(float64); ok && int(rawCopy) > 0 {
		copyCount = int(rawCopy)
	}
	if copyCount > 3 {
		copyCount = 3
	}

	if sessionID <= 0 {
		s.Utils.SendError(c, errors.New("sessiya raqami noto'g'ri"), "LinesEshikV2Reprint", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesEshikV2Reprint", "")
		return
	}

	session, err := s.Store.Repo().EshikPrintSessionGetByID(sessionID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Reprint: EshikPrintSessionGetByID", "")
		return
	}

	component := session.ToEshikComponent()
	if session.EshikComponentID > 0 {
		if liveComponent, liveErr := s.Store.Repo().EshikComponentGetByID(session.EshikComponentID); liveErr == nil {
			component = liveComponent
		}
	}

	if err := s.eshikSendPrintV2(printerV2ID, component, session.Serial, session.Counter, copyCount); err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Reprint: eshikSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id": session.ID,
		"serial":     session.Serial,
	})
}

func (s *ServerModel) LinesEshikV2SessionsLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2SessionsLast: ReadBody", "")
		return
	}

	eshikComponentID := 0
	if rawID, ok := jsonMap["eshik_component_id"].(float64); ok && int(rawID) > 0 {
		eshikComponentID = int(rawID)
	}
	limit := store.LastRecordsLimit
	if rawLimit, ok := jsonMap["limit"].(float64); ok && int(rawLimit) > 0 {
		limit = int(rawLimit)
	}

	data, err := s.Store.Repo().EshikPrintSessionsGetLast(eshikComponentID, limit)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2SessionsLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}
