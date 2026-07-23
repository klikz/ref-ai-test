package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func (s *ServerModel) ProductionKlapanGetAll(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanGetAll: ReadBody", "")
		return
	}

	data, err := s.Store.Repo().KlapanComponentsGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanGetAll", "")
		return
	}

	if rawID, ok := jsonMap["id"].(float64); ok {
		klapanID := int(rawID)
		if klapanID > 0 {
			for _, item := range data {
				if item.ID == klapanID {
					sessions, err := s.Store.Repo().KlapanPrintSessionsGetLast(klapanID, store.LastRecordsLimit)
					if err != nil {
						s.Utils.SendError(c, err, "ProductionKlapanGetAll: KlapanPrintSessionsGetLast", "")
						return
					}
					s.Utils.SendOK(c, map[string]any{
						"item":          item,
						"last_sessions": sessions,
					})
					return
				}
			}
			s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionKlapanGetAll", "")
			return
		}
	}

	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionKlapanAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanAdd: ReadBody", "")
		return
	}

	componentID := int(jsonMap["component_id"].(float64))
	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	seriyaRaqami, _ := jsonMap["seriya_raqami"].(string)
	err = s.Store.Repo().KlapanComponentAdd(
		componentID,
		c.GetInt("user_id"),
		strings.TrimSpace(seriyaRaqami),
		strings.TrimSpace(index1),
		strings.TrimSpace(index2),
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanAdd", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionKlapanUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanUpdate: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	if id <= 0 {
		s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionKlapanUpdate", "")
		return
	}

	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	seriyaRaqami, _ := jsonMap["seriya_raqami"].(string)
	err = s.Store.Repo().KlapanComponentUpdateFields(
		id,
		strings.TrimSpace(seriyaRaqami),
		strings.TrimSpace(index1),
		strings.TrimSpace(index2),
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanUpdate", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionKlapanDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanDelete: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	err = s.Store.Repo().KlapanComponentDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionKlapanDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}
