package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
)

func jsonMapEshikPartInput(jsonMap map[string]any, key string) store.EshikPartInput {
	raw, _ := jsonMap[key].(map[string]any)
	if raw == nil {
		return store.EshikPartInput{}
	}
	in := store.EshikPartInput{}
	if v, ok := raw["component_id"].(float64); ok {
		in.ComponentID = int(v)
	}
	if v, ok := raw["seriya_raqami"].(string); ok {
		in.SeriyaRaqami = v
	}
	if v, ok := raw["index1"].(string); ok {
		in.Index1 = v
	}
	if v, ok := raw["index2"].(string); ok {
		in.Index2 = v
	}
	return in
}

func (s *ServerModel) ProductionEshikGetAll(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikGetAll: ReadBody", "")
		return
	}

	data, err := s.Store.Repo().EshikModelsGetAll()
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
			s.Utils.SendError(c, errors.New("model topilmadi"), "ProductionEshikGetAll", "")
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

	modelName, _ := jsonMap["model_name"].(string)
	err = s.Store.Repo().EshikModelAdd(
		strings.TrimSpace(modelName),
		c.GetInt("user_id"),
		jsonMapEshikPartInput(jsonMap, "freeze"),
		jsonMapEshikPartInput(jsonMap, "ref"),
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

	id := int(jsonMapInt64(jsonMap, "id"))
	if id <= 0 {
		s.Utils.SendError(c, errors.New("model topilmadi"), "ProductionEshikUpdate", "")
		return
	}

	modelName, _ := jsonMap["model_name"].(string)
	err = s.Store.Repo().EshikModelUpdate(
		id,
		strings.TrimSpace(modelName),
		jsonMapEshikPartInput(jsonMap, "freeze"),
		jsonMapEshikPartInput(jsonMap, "ref"),
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

	id := int(jsonMapInt64(jsonMap, "id"))
	err = s.Store.Repo().EshikModelDelete(id)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) ProductionEshikExport(c *gin.Context) {
	data, err := s.Store.Repo().EshikModelsExportWorkbook()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikExport", "")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="eshik_models.xlsx"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (s *ServerModel) ProductionEshikTemplate(c *gin.Context) {
	data, err := s.Store.Repo().EshikModelsTemplateWorkbook()
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikTemplate", "")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="eshik_models_template.xlsx"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (s *ServerModel) ProductionEshikImport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikImport: ReadBody", "")
		return
	}

	file64, _ := jsonMap["file64"].(string)
	file64 = strings.TrimSpace(file64)
	if file64 == "" {
		s.Utils.SendError(c, errors.New("file64 is required"), "ProductionEshikImport", "")
		return
	}

	decoded, err := utils.Base64Decode(file64)
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikImport: Base64Decode", "")
		return
	}

	result, err := s.Store.Repo().EshikModelsImportWorkbook([]byte(decoded), c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "ProductionEshikImport", "")
		return
	}
	if len(result.Errors) > 0 && result.ImportedRows == 0 {
		s.Utils.SendError(c, errors.New("import xatoliklari"), "ProductionEshikImport", result)
		return
	}
	s.Utils.SendOK(c, result)
}
