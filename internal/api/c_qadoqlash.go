package api

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
)

const qadoqlashSerialPrefixLen = 6

func qadoqlashSerialMismatchError(label, scanned, stored string) error {
	expected := strings.Join(store.ParseModelAccSerialPrefixes(stored), ", ")
	if expected == "" {
		expected = "(bo'sh)"
	}
	return fmt.Errorf("%s mos emas — skan: %s ≠ model: %s", label, strings.TrimSpace(scanned), expected)
}

func (s *ServerModel) LinesQadoqlashComplete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashComplete: ReadBody", "")
		return
	}

	accSerial, _ := jsonMap["acc_serial"].(string)
	accSerial = strings.TrimSpace(accSerial)
	doorSerial, _ := jsonMap["door_serial"].(string)
	doorSerial = strings.TrimSpace(doorSerial)
	if doorSerial == "" {
		if v, ok := jsonMap["eshik_serial"].(string); ok {
			doorSerial = strings.TrimSpace(v)
		}
	}
	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	reprint := false
	switch v := jsonMap["reprint"].(type) {
	case bool:
		reprint = v
	case string:
		reprint = strings.EqualFold(strings.TrimSpace(v), "true") || strings.TrimSpace(v) == "1"
	case float64:
		reprint = v != 0
	}
	userID := c.GetInt("user_id")

	if serial == "" {
		s.Utils.SendError(c, errors.New("Product serial bo'sh"), "LinesQadoqlashComplete: serial", "")
		return
	}
	if userID <= 0 {
		s.Utils.SendError(c, errors.New("foydalanuvchi aniqlanmadi"), "LinesQadoqlashComplete: user", "")
		return
	}

	// Qayta chop: faqat product serial — qolganini DB dan olib barcha printerlarga chop.
	if reprint {
		ctx, err := s.qadoqlashLoadPrintContext(serial)
		if err != nil {
			s.Utils.SendError(c, err, "LinesQadoqlashComplete: reprint load", "")
			return
		}
		gsCode, _ := s.Store.Repo().GsCodeBySerial(serial)
		if err := s.qadoqlashPrintAll(serial, ctx.AccSerial, ctx.DoorSerial, gsCode, ctx.Model, ctx.Lab); err != nil {
			s.Utils.SendError(c, err, "LinesQadoqlashComplete: reprint print", "")
			return
		}
		s.Utils.SendOK(c, map[string]any{
			"serial":      serial,
			"acc_serial":  ctx.AccSerial,
			"door_serial": ctx.DoorSerial,
			"bx_result":   ctx.Lab.BxResult,
			"compressor":  strings.TrimSpace(ctx.Lab.Compressor),
			"bx_model":    strings.TrimSpace(ctx.Lab.BxModel),
			"model_id":    ctx.Model.ID,
			"modeli":      ctx.Model.Modeli,
			"product_id":  ctx.ProductID,
			"reprint":     true,
		})
		return
	}

	if accSerial == "" {
		s.Utils.SendError(c, errors.New("Acc serial bo'sh"), "LinesQadoqlashComplete: acc_serial", "")
		return
	}
	if doorSerial == "" {
		s.Utils.SendError(c, errors.New("Eshik serial bo'sh"), "LinesQadoqlashComplete: door_serial", "")
		return
	}

	if err := s.qadoqlashCheckAccDoorUnique(serial, accSerial, doorSerial); err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashComplete: unique", "")
		return
	}

	labDebugOK := gin.IsDebugging()
	labRow := store.LabBxRow{Serial: serial, BxResult: "01"}
	labResp, err := s.Store.Repo().LabInfoLookup(serial)
	if err != nil {
		if !labDebugOK {
			s.Utils.SendError(c, err, "LinesQadoqlashComplete: LabInfoLookup", "")
			return
		}
		s.Logger.Warn().Err(err).Str("serial", serial).Msg("qadoqlash lab debug bypass")
	} else if len(labResp.Rows) == 0 {
		if !labDebugOK {
			s.Utils.SendError(c, errors.New("Laboratoriya ma'lumoti topilmadi"), "LinesQadoqlashComplete: lab empty", "")
			return
		}
		s.Logger.Warn().Str("serial", serial).Msg("qadoqlash lab empty debug bypass")
	} else {
		labRow = labResp.Rows[len(labResp.Rows)-1]
		if strings.TrimSpace(labRow.BxResult) != "01" {
			if !labDebugOK {
				s.Utils.SendError(c, errors.New("Laboratoriyada muammo"), "LinesQadoqlashComplete: bx_result", "")
				return
			}
			s.Logger.Warn().
				Str("serial", serial).
				Str("bx_result", labRow.BxResult).
				Msg("qadoqlash lab result debug bypass")
			labRow.BxResult = "01"
		}
	}

	modelShort, err := s.Store.Repo().ModelsShortInfoBySerialPrefix(serial, qadoqlashSerialPrefixLen)
	if err != nil {
		s.Utils.SendError(c, errors.New("Model topilmadi"), "LinesQadoqlashComplete: model", "")
		return
	}
	if !store.ModelAccSerialMatches(doorSerial, modelShort.DoorCode) {
		s.Utils.SendError(c, qadoqlashSerialMismatchError("Eshik serial", doorSerial, modelShort.DoorCode), "LinesQadoqlashComplete: door", "")
		return
	}
	if !store.ModelAccSerialMatches(accSerial, modelShort.AccSerial) {
		s.Utils.SendError(c, qadoqlashSerialMismatchError("Acc serial", accSerial, modelShort.AccSerial), "LinesQadoqlashComplete: acc", "")
		return
	}

	selectedModel, err := s.Store.Repo().ModelsGetByID(modelShort.ModelId)
	if err != nil || selectedModel.ID <= 0 {
		s.Utils.SendError(c, errors.New("Model topilmadi"), "LinesQadoqlashComplete: ModelsGetByID", "")
		return
	}

	if _, err := s.Store.Repo().LabBxDataInsert(labRow, serial, userID); err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashComplete: LabBxDataInsert", "")
		return
	}

	productID, transfer, err := s.qadoqlashEnsureProduct(serial, accSerial, selectedModel.ID, userID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashComplete: product", "")
		return
	}

	cfg := utils.LoadCameraConfigFromEnv()
	var scanPhoto map[string]any
	var scanPhotoError string
	if cfg.Enabled {
		cameraCtx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout+5*time.Second)
		photo, timing, manufacturer, modelName, camErr := s.captureT3ScanPhoto(
			cameraCtx, productID, store.QadoqlashLineID, userID, serial,
		)
		cancel()
		timing.Total = timing.OnvifInit + timing.OnvifProfiles + timing.OnvifSnapshot + timing.FetchImage + timing.SaveFile + timing.DBInsert
		if camErr != nil {
			s.logT3CameraCapture(serial, manufacturer, modelName, false, timing, camErr.Error())
			scanPhotoError = utils.T3CameraUserMessage(camErr)
			s.Logger.Warn().
				Str("route", "LinesQadoqlashComplete").
				Str("serial", serial).
				Str("error", camErr.Error()).
				Msg("qadoqlash_scan_photo_warning")
		} else {
			s.logT3CameraCapture(serial, manufacturer, modelName, true, timing, "")
			scanPhoto = photo
		}
	}

	if err := s.Store.Repo().ProductParamsUpsertQadoqlash(
		serial, labRow.Compressor, accSerial, doorSerial, selectedModel.ID, userID,
	); err != nil {
		if transfer.ToProductID > 0 {
			_ = s.Store.Repo().ProductLineTransferRollback(transfer)
		}
		s.Utils.SendError(c, err, "LinesQadoqlashComplete: ProductParamsUpsert", "")
		return
	}

	gsCode, _ := s.Store.Repo().GsCodeBySerial(serial)
	if err := s.qadoqlashPrintAll(serial, accSerial, doorSerial, gsCode, selectedModel, labRow); err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashComplete: print", "")
		return
	}

	resp := map[string]any{
		"serial":      serial,
		"acc_serial":  accSerial,
		"door_serial": doorSerial,
		"bx_result":   labRow.BxResult,
		"compressor":  strings.TrimSpace(labRow.Compressor),
		"bx_model":    strings.TrimSpace(labRow.BxModel),
		"model_id":    selectedModel.ID,
		"modeli":      selectedModel.Modeli,
		"product_id":  productID,
	}
	if scanPhoto != nil {
		resp["scan_photo"] = scanPhoto
	}
	if scanPhotoError != "" {
		resp["scan_photo_error"] = scanPhotoError
	}
	s.Utils.SendOK(c, resp)
}

func (s *ServerModel) LinesQadoqlashReprint(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashReprint: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	printerV2ID := 0
	if raw, ok := jsonMap["printer_v2_id"].(float64); ok {
		printerV2ID = int(raw)
	}
	if serial == "" {
		s.Utils.SendError(c, errors.New("Product serial bo'sh"), "LinesQadoqlashReprint: serial", "")
		return
	}
	if printerV2ID <= 0 {
		s.Utils.SendError(c, errors.New("Printer tanlang"), "LinesQadoqlashReprint: printer", "")
		return
	}

	ctx, err := s.qadoqlashLoadPrintContext(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashReprint: load", "")
		return
	}

	printer, err := s.Store.Repo().PrinterV2GetByID(printerV2ID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashReprint: PrinterV2GetByID", "")
		return
	}

	gsCode, _ := s.Store.Repo().GsCodeBySerial(serial)
	printData, err := s.qadoqlashBuildPrintData(serial, ctx.AccSerial, ctx.DoorSerial, gsCode, ctx.Model, ctx.Lab)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashReprint: printData", "")
		return
	}
	if err := s.qadoqlashExecutePrint(printer, 1, printData); err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashReprint: print", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"serial":      serial,
		"acc_serial":  ctx.AccSerial,
		"door_serial": ctx.DoorSerial,
		"printer_id":  printer.ID,
		"printer":     printer.PrinterName,
	})
}

func (s *ServerModel) LinesQadoqlashSessionsLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashSessionsLast: ReadBody", "")
		return
	}
	limit := store.LastRecordsLimit
	if raw, ok := jsonMap["limit"].(float64); ok && int(raw) > 0 {
		limit = int(raw)
	}
	data, err := s.Store.Repo().QadoqlashSessionsGetLast(limit)
	if err != nil {
		s.Utils.SendError(c, err, "LinesQadoqlashSessionsLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}

type qadoqlashPrintContext struct {
	ProductID int
	AccSerial string
	DoorSerial string
	Model     models.ModelInfo
	Lab       store.LabBxRow
}

func (s *ServerModel) qadoqlashLoadPrintContext(serial string) (qadoqlashPrintContext, error) {
	out := qadoqlashPrintContext{}
	serial = strings.TrimSpace(serial)
	if serial == "" {
		return out, errors.New("Product serial bo'sh")
	}

	active, err := s.Store.Repo().ProductFindActiveBySerial(serial)
	if err != nil {
		return out, err
	}
	if active.LineID != store.QadoqlashLineID {
		return out, errors.New("Serial qadoqlash liniyasida emas")
	}

	params, err := s.Store.Repo().ProductParamsGetBySerial(serial)
	if err != nil {
		return out, err
	}
	accSerial := strings.TrimSpace(params.AccSerial)
	if accSerial == "" {
		accSerial = strings.TrimSpace(active.AccSerial)
	}
	doorSerial := strings.TrimSpace(params.DoorSerial)
	if accSerial == "" {
		return out, errors.New("DB da acc_serial topilmadi")
	}
	if doorSerial == "" {
		return out, errors.New("DB da eshik serial topilmadi")
	}

	modelID := params.ModelID
	if modelID <= 0 {
		modelID = active.ModelID
	}
	selectedModel, err := s.Store.Repo().ModelsGetByID(modelID)
	if err != nil || selectedModel.ID <= 0 {
		return out, errors.New("Model topilmadi")
	}

	labRow, err := s.Store.Repo().LabBxDataLatestBySerial(serial)
	if err != nil {
		return out, err
	}

	out.ProductID = active.ID
	out.AccSerial = accSerial
	out.DoorSerial = doorSerial
	out.Model = selectedModel
	out.Lab = labRow
	return out, nil
}

func (s *ServerModel) qadoqlashCheckAccDoorUnique(serial, accSerial, doorSerial string) error {
	if byAcc, err := s.Store.Repo().ProductParamsGetByAccSerial(accSerial); err != nil {
		return err
	} else if byAcc.ID > 0 && !strings.EqualFold(byAcc.SerialNumber, serial) {
		return fmt.Errorf("bu aksessuar nomer allaqachon kiritilgan (%s)", byAcc.SerialNumber)
	}

	if byDoor, err := s.Store.Repo().ProductParamsGetByDoorSerial(doorSerial); err != nil {
		return err
	} else if byDoor.ID > 0 && !strings.EqualFold(byDoor.SerialNumber, serial) {
		return fmt.Errorf("bu eshik nomer allaqachon kiritilgan (%s)", byDoor.SerialNumber)
	}
	return nil
}

func (s *ServerModel) qadoqlashEnsureProduct(
	serial, accSerial string,
	modelID, userID int,
) (productID int, transfer store.ProductLineTransferResult, err error) {
	active, err := s.Store.Repo().ProductFindActiveBySerial(serial)
	if err != nil {
		return 0, transfer, err
	}

	if active.LineID == store.QadoqlashLineID {
		return 0, transfer, errors.New("Bu serial allaqachon qadoqlash liniyasida. Qayta chop uchun belgilang")
	}

	transfer, err = s.Store.Repo().ProductLineTransfer(
		active.LineID, store.QadoqlashLineID, 0, userID, modelID, serial, accSerial,
	)
	if err != nil {
		return 0, transfer, err
	}
	return transfer.ToProductID, transfer, nil
}

func (s *ServerModel) qadoqlashBuildPrintData(
	serial, accSerial, doorSerial, gsCode string,
	selectedModel models.ModelInfo,
	labRow store.LabBxRow,
) (map[string]any, error) {
	printData := utils.BuildLabelPrintData(serial, accSerial, selectedModel, gsCode)
	printData["door_serial"] = doorSerial
	printData["eshik"] = map[string]any{"serial": doorSerial}
	printData["compressor"] = strings.TrimSpace(labRow.Compressor)
	printData["bx_model"] = strings.TrimSpace(labRow.BxModel)
	printData["bx_result"] = strings.TrimSpace(labRow.BxResult)
	printData["lab"] = map[string]any{
		"compressor": strings.TrimSpace(labRow.Compressor),
		"bx_model":   strings.TrimSpace(labRow.BxModel),
		"bx_result":  strings.TrimSpace(labRow.BxResult),
		"line_num":   strings.TrimSpace(labRow.LineNum),
		"point_num":  strings.TrimSpace(labRow.PointNum),
		"start_time": strings.TrimSpace(labRow.StartTime),
		"stop_time":  strings.TrimSpace(labRow.StopTime),
		"test_time":  strings.TrimSpace(labRow.TestTime),
	}
	if err := s.applyBrandLogoData(selectedModel.Brend, printData); err != nil {
		return nil, err
	}
	return printData, nil
}

func (s *ServerModel) qadoqlashPrintAll(
	serial, accSerial, doorSerial, gsCode string,
	selectedModel models.ModelInfo,
	labRow store.LabBxRow,
) error {
	printers, err := s.Store.Repo().PrintersV2GetByLine(store.QadoqlashLineID)
	if err != nil {
		return err
	}
	if len(printers) == 0 {
		return errors.New("Qadoqlash liniyasida printer topilmadi")
	}

	printData, err := s.qadoqlashBuildPrintData(serial, accSerial, doorSerial, gsCode, selectedModel, labRow)
	if err != nil {
		return err
	}

	var (
		wg       sync.WaitGroup
		mu       sync.Mutex
		firstErr error
	)
	for _, printer := range printers {
		printer := printer
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := s.qadoqlashExecutePrint(printer, 1, printData); err != nil {
				mu.Lock()
				if firstErr == nil {
					firstErr = fmt.Errorf("%s: %w", printer.PrinterName, err)
				}
				mu.Unlock()
			}
		}()
	}
	wg.Wait()
	return firstErr
}

func (s *ServerModel) qadoqlashExecutePrint(
	printer models.PrinterV2,
	copyCount int,
	printData map[string]any,
) error {
	if printer.LineID != store.QadoqlashLineID {
		return errors.New("printer Qadoqlash liniyasiga tegishli emas")
	}
	if printer.LabelTemplateID <= 0 {
		return errors.New("etiketka shablon tanlanmagan")
	}
	if strings.TrimSpace(printer.PrinterName) == "" {
		return errors.New("printer nomi bo'sh")
	}
	template, err := s.Store.Repo().LabelTemplateGetByID(printer.LabelTemplateID)
	if err != nil {
		return err
	}
	return s.Utils.PrintLabelV2(template, printer, copyCount, printData)
}
