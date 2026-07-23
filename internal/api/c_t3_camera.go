package api

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/utils"
)

const t3CameraLineID = 6

type t3CameraTimingMs struct {
	OnvifInit     int64 `json:"onvif_init_ms"`
	OnvifProfiles int64 `json:"onvif_profiles_ms"`
	OnvifSnapshot int64 `json:"onvif_snapshot_uri_ms"`
	FetchImage    int64 `json:"fetch_image_ms"`
	SaveFile      int64 `json:"save_file_ms"`
	DBInsert      int64 `json:"db_insert_ms"`
	Total         int64 `json:"total_ms"`
}

func (s *ServerModel) logT3CameraCapture(
	label, manufacturer, model string,
	ok bool,
	timing t3CameraTimingMs,
	errMsg string,
) {
	event := s.Logger.Info().
		Str("route", "T3CameraCapture").
		Str("label", label).
		Bool("ok", ok).
		Int64("onvif_init_ms", timing.OnvifInit).
		Int64("onvif_profiles_ms", timing.OnvifProfiles).
		Int64("onvif_snapshot_uri_ms", timing.OnvifSnapshot).
		Int64("fetch_image_ms", timing.FetchImage).
		Int64("save_file_ms", timing.SaveFile).
		Int64("db_insert_ms", timing.DBInsert).
		Int64("total_ms", timing.Total)
	if manufacturer != "" {
		event = event.Str("camera_manufacturer", manufacturer)
	}
	if model != "" {
		event = event.Str("camera_model", model)
	}
	if errMsg != "" {
		event = event.Str("error", errMsg)
	}
	event.Msg("t3_camera_capture")
}

type t3ParallelPhotoResult struct {
	photo        map[string]any
	timing       t3CameraTimingMs
	manufacturer string
	model        string
	err          error
}

func (s *ServerModel) startParallelT3ScanPhoto(
	parentCtx context.Context,
	productID, userID int,
	serial string,
	cfg utils.CameraConfig,
) <-chan t3ParallelPhotoResult {
	done := make(chan t3ParallelPhotoResult, 1)
	cameraCtx, cancel := context.WithTimeout(parentCtx, cfg.Timeout+5*time.Second)
	go func() {
		defer cancel()
		captureStart := time.Now()
		photo, timing, manufacturer, model, err := s.captureT3ScanPhoto(cameraCtx, productID, t3CameraLineID, userID, serial)
		timing.Total = time.Since(captureStart).Milliseconds()
		done <- t3ParallelPhotoResult{
			photo:        photo,
			timing:       timing,
			manufacturer: manufacturer,
			model:        model,
			err:          err,
		}
	}()
	return done
}

func (s *ServerModel) discardParallelT3Photo(done <-chan t3ParallelPhotoResult) {
	res := <-done
	if res.err == nil && res.photo != nil {
		s.removeT3ScanPhoto(res.photo)
	}
}

func (s *ServerModel) removeT3ScanPhoto(photo map[string]any) {
	if photo == nil {
		return
	}

	var id int64
	switch v := photo["id"].(type) {
	case int64:
		id = v
	case int:
		id = int64(v)
	case float64:
		id = int64(v)
	}

	filePath, err := s.Store.Repo().T3ScanPhotoDelete(id)
	if err != nil {
		s.Logger.Warn().Err(err).Int64("id", id).Msg("t3_scan_photo_delete_db")
	}
	if filePath == "" {
		if fp, ok := photo["file_path"].(string); ok {
			filePath = fp
		}
	}
	if err := deleteT3ScanPhotoFile(filePath); err != nil && !os.IsNotExist(err) {
		s.Logger.Warn().Err(err).Str("path", filePath).Msg("t3_scan_photo_delete_file")
	}
}

func deleteT3ScanPhotoFile(filePath string) error {
	filePath = strings.TrimSpace(filePath)
	if filePath == "" {
		return nil
	}
	rel := strings.TrimPrefix(filePath, "/")
	return os.Remove(filepath.FromSlash(rel))
}

func (s *ServerModel) attachParallelT3PhotoResponse(
	resp map[string]any,
	serial string,
	res t3ParallelPhotoResult,
) {
	if res.err != nil {
		s.logT3CameraCapture(serial, res.manufacturer, res.model, false, res.timing, res.err.Error())
		s.Logger.Warn().
			Str("route", "LinesT3V2SerialPrint").
			Str("serial", serial).
			Str("error", res.err.Error()).
			Msg("t3_scan_photo_capture")
		resp["scan_photo_error"] = utils.T3CameraUserMessage(res.err)
		return
	}
	s.logT3CameraCapture(serial, res.manufacturer, res.model, true, res.timing, "")
	resp["scan_photo"] = res.photo
}

func (s *ServerModel) captureT3ScanPhoto(ctx context.Context, productID, lineID, userID int, serial string) (map[string]any, t3CameraTimingMs, string, string, error) {
	timing := t3CameraTimingMs{}
	label := utils.SanitizeT3ScanLabel(serial)

	cfg := utils.LoadCameraConfigFromEnv()
	if !cfg.Enabled {
		return nil, timing, "", "", fmt.Errorf("kamera o'chirilgan (T3_CAMERA_ENABLED=false)")
	}
	if lineID <= 0 {
		lineID = t3CameraLineID
	}

	capture, err := utils.FetchONVIFSnapshot(ctx, cfg)
	timing.OnvifInit = capture.Timing.InitMs
	timing.OnvifProfiles = capture.Timing.ProfilesMs
	timing.OnvifSnapshot = capture.Timing.SnapshotURIMs
	timing.FetchImage = capture.Timing.FetchImageMs
	if err != nil {
		return nil, timing, capture.Manufacturer, capture.Model, err
	}

	capturedAt := time.Now()
	saveStart := time.Now()
	saved, err := utils.SaveT3ScanPhoto(label, capture.Data, capturedAt)
	timing.SaveFile = time.Since(saveStart).Milliseconds()
	if err != nil {
		return nil, timing, capture.Manufacturer, capture.Model, err
	}

	dbStart := time.Now()
	id, err := s.Store.Repo().T3ScanPhotoInsert(productID, lineID, userID, label, saved.FilePath)
	timing.DBInsert = time.Since(dbStart).Milliseconds()
	if err != nil {
		return nil, timing, capture.Manufacturer, capture.Model, err
	}

	return map[string]any{
		"id":          id,
		"file_path":   saved.FilePath,
		"captured_at": saved.CapturedAt.Format(time.RFC3339),
		"serial":      label,
	}, timing, capture.Manufacturer, capture.Model, nil
}

func (s *ServerModel) T3CameraCapture(c *gin.Context) {
	totalStart := time.Now()

	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "T3CameraCapture: ReadBody", "")
		return
	}

	label := "test"
	if raw, ok := jsonMap["label"].(string); ok {
		label = utils.SanitizeT3ScanLabel(raw)
	}

	cfg := utils.LoadCameraConfigFromEnv()
	ctx, cancel := context.WithTimeout(c.Request.Context(), cfg.Timeout+5*time.Second)
	defer cancel()

	photo, timing, manufacturer, model, err := s.captureT3ScanPhoto(ctx, 0, t3CameraLineID, c.GetInt("user_id"), label)
	if err != nil {
		timing.Total = time.Since(totalStart).Milliseconds()
		s.logT3CameraCapture(label, manufacturer, model, false, timing, err.Error())
		s.Utils.SendError(c, err, "T3CameraCapture: captureT3ScanPhoto", "")
		return
	}

	timing.Total = time.Since(totalStart).Milliseconds()
	s.logT3CameraCapture(label, manufacturer, model, true, timing, "")
	s.Utils.SendOK(c, map[string]any{
		"id":           photo["id"],
		"file_path":    photo["file_path"],
		"captured_at":  photo["captured_at"],
		"serial":       photo["serial"],
		"manufacturer": manufacturer,
		"model":        model,
		"timing_ms":    timing,
	})
}

func (s *ServerModel) T3CameraPhotosList(c *gin.Context) {
	start := time.Now()

	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "T3CameraPhotosList: ReadBody", "")
		return
	}

	dateFrom, _ := jsonMap["date_from"].(string)
	dateTo, _ := jsonMap["date_to"].(string)
	limit := 100
	offset := 0
	if raw, ok := jsonMap["limit"].(float64); ok && int(raw) > 0 {
		limit = int(raw)
	}
	if raw, ok := jsonMap["offset"].(float64); ok && int(raw) >= 0 {
		offset = int(raw)
	}

	items, err := s.Store.Repo().T3ScanPhotosList(dateFrom, dateTo, limit, offset)
	durationMs := time.Since(start).Milliseconds()
	if err != nil {
		s.Logger.Info().
			Str("route", "T3CameraPhotosList").
			Bool("ok", false).
			Int64("duration_ms", durationMs).
			Str("error", err.Error()).
			Msg("t3_camera_photos_list")
		s.Utils.SendError(c, err, "T3CameraPhotosList", "")
		return
	}

	s.Logger.Info().
		Str("route", "T3CameraPhotosList").
		Bool("ok", true).
		Int("count", len(items)).
		Int64("duration_ms", durationMs).
		Msg("t3_camera_photos_list")

	s.Utils.SendOK(c, items)
}
