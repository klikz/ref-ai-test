package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func buildRadiatorV2PrintData(component store.RadiatorComponent, serial string, counter int) map[string]any {
	return map[string]any{
		"serial":           serial,
		"seriya_raqami":    component.SeriyaRaqami,
		"radiator_counter": counter,
		"index_1":          component.Index1,
		"index_2":          component.Index2,
		"radiator": map[string]any{
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

func (s *ServerModel) radiatorSendPrintV2(
	printerV2ID int,
	component store.RadiatorComponent,
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
	if printerV2.LineID != store.RadiatorLineID {
		return errors.New("printer radiator liniyasiga tegishli emas")
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

	printData := buildRadiatorV2PrintData(component, serial, counter)
	return s.Utils.PrintLabelV2(template, printerV2, copyCount, printData)
}

func (s *ServerModel) LinesRadiatorV2Print(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: ReadBody", "")
		return
	}

	radiatorComponentID := int(jsonMap["radiator_component_id"].(float64))
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	copyCount := 1
	if rawCopy, ok := jsonMap["copy"].(float64); ok && int(rawCopy) > 0 {
		copyCount = int(rawCopy)
	}
	if copyCount > 3 {
		copyCount = 3
	}
	userID := c.GetInt("user_id")

	if radiatorComponentID <= 0 {
		s.Utils.SendError(c, errors.New("komponent tanlanmagan"), "LinesRadiatorV2Print", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesRadiatorV2Print", "")
		return
	}

	component, err := s.Store.Repo().RadiatorComponentGetByID(radiatorComponentID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: RadiatorComponentGetByID", "")
		return
	}

	radiatorLineID, err := s.Store.Repo().LinesRadiatorLineID()
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: LinesRadiatorLineID", "")
		return
	}

	if err := s.Store.Repo().RadiatorChiqishBalancePrecheck(radiatorLineID, component.FinPressComponentRefID); err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: RadiatorChiqishBalancePrecheck", "")
		return
	}

	if err := s.enforceProductionPlan(radiatorLineID, 0, component.ComponentID, 1); err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: plan", "")
		return
	}

	counter, err := s.Store.Repo().RadiatorNextDailyCounter(radiatorComponentID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: RadiatorNextDailyCounter", "")
		return
	}

	serial, err := s.GenerateSerial(strings.TrimSpace(component.SeriyaRaqami), counter)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: GenerateSerial", "")
		return
	}

	sessionID, err := s.Store.Repo().RadiatorPrintSessionCreate(component, serial, counter, userID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: RadiatorPrintSessionCreate", "")
		return
	}

	auxiliaryProductID, err := s.Store.Repo().RadiatorChiqishBalanceApply(
		radiatorLineID,
		component.ComponentID,
		component.FinPressComponentRefID,
		userID,
		sessionID,
		serial,
	)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: RadiatorChiqishBalanceApply", "")
		return
	}

	if err := s.radiatorSendPrintV2(printerV2ID, component, serial, counter, copyCount); err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Print: radiatorSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id":           sessionID,
		"serial":               serial,
		"counter":              counter,
		"auxiliary_product_id": auxiliaryProductID,
	})
}

func (s *ServerModel) LinesRadiatorV2Reprint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Reprint: ReadBody", "")
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
		s.Utils.SendError(c, errors.New("sessiya raqami noto'g'ri"), "LinesRadiatorV2Reprint", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesRadiatorV2Reprint", "")
		return
	}

	session, err := s.Store.Repo().RadiatorPrintSessionGetByID(sessionID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Reprint: RadiatorPrintSessionGetByID", "")
		return
	}

	component := session.ToRadiatorComponent()
	if session.RadiatorComponentID > 0 {
		if liveComponent, liveErr := s.Store.Repo().RadiatorComponentGetByID(session.RadiatorComponentID); liveErr == nil {
			component = liveComponent
		}
	}

	if err := s.radiatorSendPrintV2(printerV2ID, component, session.Serial, session.Counter, copyCount); err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2Reprint: radiatorSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id": session.ID,
		"serial":     session.Serial,
	})
}

func (s *ServerModel) LinesRadiatorV2SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2SerialRePrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))

	if serial == "" {
		s.Utils.SendError(c, errors.New("serial bo'sh"), "LinesRadiatorV2SerialRePrint", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesRadiatorV2SerialRePrint", "")
		return
	}

	session, err := s.Store.Repo().RadiatorPrintSessionGetBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2SerialRePrint: RadiatorPrintSessionGetBySerial", "")
		return
	}

	component := session.ToRadiatorComponent()
	if session.RadiatorComponentID > 0 {
		if liveComponent, liveErr := s.Store.Repo().RadiatorComponentGetByID(session.RadiatorComponentID); liveErr == nil {
			component = liveComponent
		}
	}

	if err := s.radiatorSendPrintV2(printerV2ID, component, session.Serial, session.Counter, 1); err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2SerialRePrint: radiatorSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id": session.ID,
		"serial":     session.Serial,
	})
}

func (s *ServerModel) LinesRadiatorV2SessionsLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2SessionsLast: ReadBody", "")
		return
	}

	radiatorComponentID := 0
	if rawID, ok := jsonMap["radiator_component_id"].(float64); ok && int(rawID) > 0 {
		radiatorComponentID = int(rawID)
	}
	limit := store.LastRecordsLimit
	if rawLimit, ok := jsonMap["limit"].(float64); ok && int(rawLimit) > 0 {
		limit = int(rawLimit)
	}

	data, err := s.Store.Repo().RadiatorPrintSessionsGetLast(radiatorComponentID, limit)
	if err != nil {
		s.Utils.SendError(c, err, "LinesRadiatorV2SessionsLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}
