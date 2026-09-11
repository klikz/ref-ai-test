package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func buildEshikV2PrintData(model store.EshikModel, part store.EshikModelPart, serial string, counter int) map[string]any {
	return map[string]any{
		"serial":        serial,
		"seriya_raqami": part.SeriyaRaqami,
		"eshik_counter": counter,
		"index_1":       part.Index1,
		"index_2":       part.Index2,
		"door_code":     part.DoorCode,
		"model_name":    model.ModelName,
		"eshik": map[string]any{
			"index1":     part.Index1,
			"index2":     part.Index2,
			"door_code":  part.DoorCode,
			"model_name": model.ModelName,
		},
		"component": map[string]any{
			"factory_code":      part.FactoryCode,
			"full_name_uz":      part.FullNameUz,
			"manufacturer_code": part.ManufacturerCode,
			"standard_name_uz":  part.StandardNameUz,
			"odoo_code":         part.OdooCode,
		},
		"gscode": map[string]any{
			"data": serial,
		},
	}
}

func (s *ServerModel) eshikSendPrintV2(
	printerV2ID int,
	model store.EshikModel,
	part store.EshikModelPart,
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

	printData := buildEshikV2PrintData(model, part, serial, counter)
	return s.Utils.PrintLabelV2(template, printerV2, copyCount, printData)
}

type eshikPrintAttempt struct {
	sessionID          int64
	auxiliaryProductID int64
	eshikModelPartID   int
	componentID        int
	counter            int
	doorCode           string
	serial             string
	part               store.EshikModelPart
}

func (s *ServerModel) rollbackEshikPrintAttempt(lineID, userID int, attempt eshikPrintAttempt) {
	if attempt.sessionID <= 0 && attempt.auxiliaryProductID <= 0 {
		return
	}
	_ = s.Store.Repo().EshikPrintRollback(
		lineID,
		attempt.sessionID,
		attempt.eshikModelPartID,
		attempt.componentID,
		attempt.auxiliaryProductID,
		userID,
	)
}

// eshikCommitOnePart writes session + balance + auxiliary. Does not print.
func (s *ServerModel) eshikCommitOnePart(
	model store.EshikModel,
	part store.EshikModelPart,
	eshikLineID, userID int,
) (eshikPrintAttempt, error) {
	attempt := eshikPrintAttempt{
		eshikModelPartID: part.ID,
		componentID:      part.ComponentID,
		doorCode:         part.DoorCode,
		part:             part,
	}

	if !part.IsComplete() {
		return attempt, errors.New("eshik qismi to'liq emas")
	}

	counter, err := s.Store.Repo().EshikNextDailyCounter(part.ID)
	if err != nil {
		return attempt, err
	}
	attempt.counter = counter

	serial, err := s.GenerateSerial(strings.TrimSpace(part.SeriyaRaqami), counter)
	if err != nil {
		_ = s.Store.Repo().EshikDailyCounterUndo(part.ID)
		return attempt, err
	}
	attempt.serial = serial

	sessionID, err := s.Store.Repo().EshikPrintSessionCreate(model, part, serial, counter, userID)
	if err != nil {
		_ = s.Store.Repo().EshikDailyCounterUndo(part.ID)
		return attempt, err
	}
	attempt.sessionID = sessionID

	auxiliaryProductID, err := s.Store.Repo().EshikChiqishBalanceApply(
		eshikLineID,
		part.ComponentID,
		userID,
		sessionID,
		serial,
	)
	if err != nil {
		s.rollbackEshikPrintAttempt(eshikLineID, userID, attempt)
		attempt.sessionID = 0
		return attempt, err
	}
	attempt.auxiliaryProductID = auxiliaryProductID
	return attempt, nil
}

func eshikAttemptResultParts(attempts []eshikPrintAttempt) []map[string]any {
	resultParts := make([]map[string]any, 0, len(attempts))
	for _, attempt := range attempts {
		resultParts = append(resultParts, map[string]any{
			"door_code":            attempt.doorCode,
			"session_id":           attempt.sessionID,
			"serial":               attempt.serial,
			"counter":              attempt.counter,
			"auxiliary_product_id": attempt.auxiliaryProductID,
		})
	}
	return resultParts
}

func (s *ServerModel) LinesEshikV2Print(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: ReadBody", "")
		return
	}

	eshikModelID := int(jsonMapInt64(jsonMap, "eshik_model_id"))
	if eshikModelID <= 0 {
		eshikModelID = int(jsonMapInt64(jsonMap, "id"))
	}
	printerV2ID := int(jsonMapInt64(jsonMap, "printer_v2_id"))
	copyCount := 1
	if rawCopy := jsonMapInt64(jsonMap, "copy"); rawCopy > 0 {
		copyCount = int(rawCopy)
	}
	if copyCount > 3 {
		copyCount = 3
	}
	userID := c.GetInt("user_id")

	if eshikModelID <= 0 {
		s.Utils.SendError(c, errors.New("model tanlanmagan"), "LinesEshikV2Print", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesEshikV2Print", "")
		return
	}

	model, err := s.Store.Repo().EshikModelGetByID(eshikModelID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: EshikModelGetByID", "")
		return
	}
	if !model.Freeze.IsComplete() || !model.Ref.IsComplete() {
		s.Utils.SendError(c, errors.New("modelda freeze va ref to'liq emas"), "LinesEshikV2Print", "")
		return
	}

	eshikLineID, err := s.Store.Repo().LinesEshikLineID()
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: LinesEshikLineID", "")
		return
	}

	// One pair = one plan unit; gate once on freeze component (same planned qty as ref).
	if err := s.enforceProductionPlan(eshikLineID, 0, model.Freeze.ComponentID, 1); err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Print: plan", "")
		return
	}

	parts := []store.EshikModelPart{model.Freeze, model.Ref}
	attempts := make([]eshikPrintAttempt, 0, 2)
	for _, part := range parts {
		attempt, commitErr := s.eshikCommitOnePart(model, part, eshikLineID, userID)
		if commitErr != nil {
			for i := len(attempts) - 1; i >= 0; i-- {
				s.rollbackEshikPrintAttempt(eshikLineID, userID, attempts[i])
			}
			s.Utils.SendError(c, commitErr, "LinesEshikV2Print: "+part.DoorCode, "")
			return
		}
		attempts = append(attempts, attempt)
	}

	resultParts := eshikAttemptResultParts(attempts)
	resp := map[string]any{
		"eshik_model_id": model.ID,
		"model_name":     model.ModelName,
		"parts":          resultParts,
		"serial":         attempts[0].serial,
		"session_id":     attempts[0].sessionID,
	}

	// Print after DB commit. Printer failure must not roll back balance/sessions.
	var printErrs []string
	for _, attempt := range attempts {
		if err := s.eshikSendPrintV2(printerV2ID, model, attempt.part, attempt.serial, attempt.counter, copyCount); err != nil {
			printErrs = append(printErrs, attempt.doorCode+": "+err.Error())
		}
	}
	if len(printErrs) > 0 {
		resp["print_error"] = strings.Join(printErrs, "; ")
		s.Utils.SendError(c, errors.New("Saqlandi, lekin print xato: "+strings.Join(printErrs, "; ")+" — qayta chop eting"), "LinesEshikV2Print: print", resp)
		return
	}

	s.Utils.SendOK(c, resp)
}

func jsonMapInt64(jsonMap map[string]any, key string) int64 {
	switch v := jsonMap[key].(type) {
	case float64:
		return int64(v)
	case int:
		return int64(v)
	case int64:
		return v
	default:
		return 0
	}
}

func (s *ServerModel) LinesEshikV2Reprint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Reprint: ReadBody", "")
		return
	}

	sessionID := jsonMapInt64(jsonMap, "session_id")
	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	printerV2ID := int(jsonMapInt64(jsonMap, "printer_v2_id"))
	copyCount := 1
	if rawCopy := jsonMapInt64(jsonMap, "copy"); rawCopy > 0 {
		copyCount = int(rawCopy)
	}
	if copyCount > 3 {
		copyCount = 3
	}

	if sessionID <= 0 && serial == "" {
		s.Utils.SendError(c, errors.New("serial yoki sessiya raqami kerak"), "LinesEshikV2Reprint", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("printer not found"), "LinesEshikV2Reprint", "")
		return
	}

	var session store.EshikPrintSession
	if sessionID > 0 {
		session, err = s.Store.Repo().EshikPrintSessionGetByID(sessionID)
	} else {
		session, err = s.Store.Repo().EshikPrintSessionGetBySerial(serial)
	}
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Reprint: session", "")
		return
	}

	model := store.EshikModel{
		ID:        session.EshikModelID,
		ModelName: session.ModelName,
	}
	part := session.ToEshikModelPart()
	if session.EshikModelPartID > 0 {
		if livePart, liveErr := s.Store.Repo().EshikModelPartGetByID(session.EshikModelPartID); liveErr == nil {
			part = livePart
		}
	}
	if session.EshikModelID > 0 {
		if liveModel, liveErr := s.Store.Repo().EshikModelGetByID(session.EshikModelID); liveErr == nil {
			model = liveModel
		}
	}

	if err := s.eshikSendPrintV2(printerV2ID, model, part, session.Serial, session.Counter, copyCount); err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2Reprint: eshikSendPrintV2", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"session_id": session.ID,
		"serial":     session.Serial,
		"door_code":  part.DoorCode,
	})
}

func (s *ServerModel) LinesEshikV2SessionsLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2SessionsLast: ReadBody", "")
		return
	}

	eshikModelID := int(jsonMapInt64(jsonMap, "eshik_model_id"))
	limit := store.LastRecordsLimit
	if rawLimit := jsonMapInt64(jsonMap, "limit"); rawLimit > 0 {
		limit = int(rawLimit)
	}

	data, err := s.Store.Repo().EshikPrintSessionsGetLast(eshikModelID, limit)
	if err != nil {
		s.Utils.SendError(c, err, "LinesEshikV2SessionsLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}
