package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func buildKlapanV2PrintData(component store.KlapanComponent, serial string, counter int) map[string]any {
	return map[string]any{
		"serial":         serial,
		"seriya_raqami":  component.SeriyaRaqami,
		"klapan_counter": counter,
		"index_1":        component.Index1,
		"index_2":        component.Index2,
		"klapan": map[string]any{
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

func (s *ServerModel) klapanSendPrintV2(
	printerV2ID int,
	component store.KlapanComponent,
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
	if printerV2.LineID != store.KlapanLineID {
		return errors.New("printer klapan yig'ish liniyasiga tegishli emas")
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

	printData := buildKlapanV2PrintData(component, serial, counter)
	return s.Utils.PrintLabelV2(template, printerV2, copyCount, printData)
}

func (s *ServerModel) LinesKlapanV2Print(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: ReadBody", "")
		return
	}

	klapanComponentID := int(jsonMap["klapan_component_id"].(float64))
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))
	copyCount := 1
	if rawCopy, ok := jsonMap["copy"].(float64); ok && int(rawCopy) > 0 {
		copyCount = int(rawCopy)
	}
	if copyCount > 3 {
		copyCount = 3
	}
	userID := c.GetInt("user_id")

	if klapanComponentID <= 0 {
		s.Utils.SendError(c, errors.New("komponent tanlanmagan"), "LinesKlapanV2Print", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesKlapanV2Print", "")
		return
	}

	component, err := s.Store.Repo().KlapanComponentGetByID(klapanComponentID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: KlapanComponentGetByID", "")
		return
	}

	klapanLineID, err := s.Store.Repo().LinesKlapanLineID()
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: LinesKlapanLineID", "")
		return
	}

	if err := s.enforceProductionPlan(klapanLineID, 0, component.ComponentID, 1); err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: plan", "")
		return
	}

	counter, err := s.Store.Repo().KlapanNextDailyCounter(klapanComponentID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: KlapanNextDailyCounter", "")
		return
	}

	serial, err := s.GenerateSerial(strings.TrimSpace(component.SeriyaRaqami), counter)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: GenerateSerial", "")
		return
	}

	sessionID, err := s.Store.Repo().KlapanPrintSessionCreate(component, serial, counter, userID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: KlapanPrintSessionCreate", "")
		return
	}

	auxiliaryProductID, err := s.Store.Repo().KlapanChiqishBalanceApply(
		klapanLineID,
		component.ComponentID,
		userID,
		sessionID,
		serial,
	)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: KlapanChiqishBalanceApply", "")
		return
	}

	if err := s.klapanSendPrintV2(printerV2ID, component, serial, counter, copyCount); err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Print: klapanSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id":           sessionID,
		"serial":               serial,
		"counter":              counter,
		"auxiliary_product_id": auxiliaryProductID,
	})
}

func (s *ServerModel) LinesKlapanV2Reprint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Reprint: ReadBody", "")
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
		s.Utils.SendError(c, errors.New("sessiya raqami noto'g'ri"), "LinesKlapanV2Reprint", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesKlapanV2Reprint", "")
		return
	}

	session, err := s.Store.Repo().KlapanPrintSessionGetByID(sessionID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Reprint: KlapanPrintSessionGetByID", "")
		return
	}

	component := session.ToKlapanComponent()
	if session.KlapanComponentID > 0 {
		if liveComponent, liveErr := s.Store.Repo().KlapanComponentGetByID(session.KlapanComponentID); liveErr == nil {
			component = liveComponent
		}
	}

	if err := s.klapanSendPrintV2(printerV2ID, component, session.Serial, session.Counter, copyCount); err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2Reprint: klapanSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id": session.ID,
		"serial":     session.Serial,
	})
}

func (s *ServerModel) LinesKlapanV2SerialRePrint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2SerialRePrint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	printerV2ID := int(jsonMap["printer_v2_id"].(float64))

	if serial == "" {
		s.Utils.SendError(c, errors.New("serial bo'sh"), "LinesKlapanV2SerialRePrint", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesKlapanV2SerialRePrint", "")
		return
	}

	session, err := s.Store.Repo().KlapanPrintSessionGetBySerial(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2SerialRePrint: KlapanPrintSessionGetBySerial", "")
		return
	}

	component := session.ToKlapanComponent()
	if session.KlapanComponentID > 0 {
		if liveComponent, liveErr := s.Store.Repo().KlapanComponentGetByID(session.KlapanComponentID); liveErr == nil {
			component = liveComponent
		}
	}

	if err := s.klapanSendPrintV2(printerV2ID, component, session.Serial, session.Counter, 1); err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2SerialRePrint: klapanSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id": session.ID,
		"serial":     session.Serial,
	})
}

func (s *ServerModel) LinesKlapanV2SessionsLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2SessionsLast: ReadBody", "")
		return
	}

	klapanComponentID := 0
	if rawID, ok := jsonMap["klapan_component_id"].(float64); ok && int(rawID) > 0 {
		klapanComponentID = int(rawID)
	}
	limit := store.LastRecordsLimit
	if rawLimit, ok := jsonMap["limit"].(float64); ok && int(rawLimit) > 0 {
		limit = int(rawLimit)
	}

	data, err := s.Store.Repo().KlapanPrintSessionsGetLast(klapanComponentID, limit)
	if err != nil {
		s.Utils.SendError(c, err, "LinesKlapanV2SessionsLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}
