package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func (s *ServerModel) ProductionEshikGetAll(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikGetAll: ReadBody", "")
		return
	}

	data, err := s.Store.Repo().EshikComponentsGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikGetAll", "")
		return
	}

	if rawID, ok := jsonMap["id"].(float64); ok {
		eshikID := int(rawID)
		if eshikID > 0 {
			for _, item := range data {
				if item.ID == eshikID {
					sessions, err := s.Store.Repo().EshikPrintSessionsGetLast(eshikID, store.LastRecordsLimit)
					if err != nil {
						s.Utils.SendError(c, err, "ProductionEshikGetAll: EshikPrintSessionsGetLast", "")
						return
					}
					s.Utils.SendOK(c, map[string]any{
						"item":          item,
						"last_sessions": sessions,
					})
					return
				}
			}
			s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionEshikGetAll", "")
			return
		}
	}

	s.Utils.SendOK(c, data)
}

func (s *ServerModel) ProductionEshikAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikAdd: ReadBody", "")
		return
	}

	componentID := int(jsonMap["component_id"].(float64))
	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	seriyaRaqami, _ := jsonMap["seriya_raqami"].(string)
	err = s.Store.Repo().EshikComponentAdd(
		componentID,
		c.GetInt("user_id"),
		strings.TrimSpace(seriyaRaqami),
		strings.TrimSpace(index1),
		strings.TrimSpace(index2),
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikAdd", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionEshikUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikUpdate: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	if id <= 0 {
		s.Utils.SendError(c, errors.New("komponent topilmadi"), "ProductionEshikUpdate", "")
		return
	}

	index1, _ := jsonMap["index1"].(string)
	index2, _ := jsonMap["index2"].(string)
	seriyaRaqami, _ := jsonMap["seriya_raqami"].(string)
	err = s.Store.Repo().EshikComponentUpdateFields(
		id,
		strings.TrimSpace(seriyaRaqami),
		strings.TrimSpace(index1),
		strings.TrimSpace(index2),
	)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikUpdate", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionEshikDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikDelete: ReadBody", "")
		return
	}

	id := int(jsonMap["id"].(float64))
	err = s.Store.Repo().EshikComponentDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}
