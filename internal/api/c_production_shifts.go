package api

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func (s *ServerModel) ProductionShiftsGet(c *gin.Context) {
	settings, err := s.Store.Repo().ProductionShiftSettingsGet()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionShiftsGet", "")
		return
	}
	current, err := s.Store.Repo().ProductionCurrentShift(time.Time{})
	if err != nil {
		s.Utils.SendError(c, err, "ProductionShiftsGet: current", "")
		return
	}
	s.Utils.SendOK(c, map[string]any{
		"settings": settings,
		"current":  current,
	})
}

func (s *ServerModel) ProductionShiftsSave(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionShiftsSave: ReadBody", "")
		return
	}
	settings := store.ProductionShiftSettings{}
	if v, ok := jsonMap["shift1_start"].(string); ok {
		settings.Shift1Start = v
	}
	if v, ok := jsonMap["shift1_end"].(string); ok {
		settings.Shift1End = v
	}
	if v, ok := jsonMap["shift2_start"].(string); ok {
		settings.Shift2Start = v
	}
	if v, ok := jsonMap["shift2_end"].(string); ok {
		settings.Shift2End = v
	}
	saved, err := s.Store.Repo().ProductionShiftSettingsSave(settings, c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "ProductionShiftsSave", "")
		return
	}
	current, err := s.Store.Repo().ProductionCurrentShift(time.Time{})
	if err != nil {
		s.Utils.SendError(c, err, "ProductionShiftsSave: current", "")
		return
	}
	s.Utils.SendOK(c, map[string]any{
		"settings": saved,
		"current":  current,
	})
}
