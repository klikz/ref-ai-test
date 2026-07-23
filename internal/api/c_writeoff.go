package api

import (
	"errors"
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

func (s *ServerModel) WriteoffResponsiblesGetAll(c *gin.Context) {
	data, err := s.Store.Repo().WriteoffResponsiblesGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffResponsiblesGetAll", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WriteoffResponsiblesAdd(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffResponsiblesAdd: ReadBody", "")
		return
	}
	userID := int(jsonMap["user_id"].(float64))
	if userID <= 0 {
		s.Utils.SendError(c, errors.New("foydalanuvchi tanlanmagan"), "WriteoffResponsiblesAdd", "")
		return
	}
	if err := s.Store.Repo().WriteoffResponsibleAdd(userID, c.GetInt("user_id")); err != nil {
		s.Utils.SendError(c, err, "WriteoffResponsiblesAdd", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) WriteoffResponsiblesDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffResponsiblesDelete: ReadBody", "")
		return
	}
	id := int64(jsonMap["id"].(float64))
	if err := s.Store.Repo().WriteoffResponsibleDelete(id); err != nil {
		s.Utils.SendError(c, err, "WriteoffResponsiblesDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) WriteoffDocumentsList(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentsList: ReadBody", "")
		return
	}
	status, _ := jsonMap["status"].(string)
	limit := 200
	if raw, ok := jsonMap["limit"].(float64); ok && int(raw) > 0 {
		limit = int(raw)
	}
	data, err := s.Store.Repo().WriteoffDocumentsList(status, limit)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentsList", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WriteoffDocumentGet(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentGet: ReadBody", "")
		return
	}
	documentID := int64(jsonMap["document_id"].(float64))
	doc, items, approvals, err := s.Store.Repo().WriteoffDocumentGet(documentID)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentGet", "")
		return
	}
	s.Utils.SendOK(c, map[string]any{"document": doc, "items": items, "approvals": approvals})
}

func (s *ServerModel) WriteoffDocumentCreate(c *gin.Context) {
	id, err := s.Store.Repo().WriteoffDocumentCreate(c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentCreate", "")
		return
	}
	s.Utils.SendOK(c, map[string]int64{"document_id": id})
}

func (s *ServerModel) WriteoffDocumentDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentDelete: ReadBody", "")
		return
	}
	documentID := int64(jsonMap["document_id"].(float64))
	if err := s.Store.Repo().WriteoffDocumentDelete(documentID); err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentDelete", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func parseWriteoffSaveItems(rawItems []any) ([]store.WriteoffSaveItemInput, error) {
	items := make([]store.WriteoffSaveItemInput, 0, len(rawItems))
	for i, raw := range rawItems {
		row, ok := raw.(map[string]any)
		if !ok {
			return nil, errors.New("noto'g'ri qator formati")
		}
		item := store.WriteoffSaveItemInput{SortOrder: i}
		if v, ok := row["id"].(float64); ok {
			item.ID = int64(v)
		}
		if v, ok := row["line_id"].(float64); ok {
			item.LineID = int(v)
		}
		if v, ok := row["item_type"].(string); ok {
			item.ItemType = strings.TrimSpace(v)
		}
		if v, ok := row["model_id"].(float64); ok {
			item.ModelID = int(v)
		}
		if v, ok := row["component_id"].(float64); ok {
			item.ComponentID = int(v)
		}
		if v, ok := row["serial"].(string); ok {
			item.Serial = v
		}
		if v, ok := row["quantity"].(float64); ok {
			item.Quantity = v
		}
		if v, ok := row["comment"].(string); ok {
			item.Comment = v
		}
		items = append(items, item)
	}
	return items, nil
}

func (s *ServerModel) WriteoffDocumentItemsSave(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentItemsSave: ReadBody", "")
		return
	}
	documentID := int64(jsonMap["document_id"].(float64))
	rawItems, ok := jsonMap["items"].([]any)
	if !ok {
		s.Utils.SendError(c, errors.New("items talab qilinadi"), "WriteoffDocumentItemsSave", "")
		return
	}
	items, err := parseWriteoffSaveItems(rawItems)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentItemsSave", "")
		return
	}
	if err := s.Store.Repo().WriteoffDocumentItemsSave(documentID, items); err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentItemsSave", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"saved_rows": len(items)})
}

func (s *ServerModel) WriteoffDocumentSubmit(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentSubmit: ReadBody", "")
		return
	}
	documentID := int64(jsonMap["document_id"].(float64))
	if err := s.Store.Repo().WriteoffDocumentSubmit(documentID, c.GetInt("user_id")); err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentSubmit", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) WriteoffDocumentApprove(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentApprove: ReadBody", "")
		return
	}
	documentID := int64(jsonMap["document_id"].(float64))
	result, err := s.Store.Repo().WriteoffDocumentApprove(documentID, c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentApprove", "")
		return
	}
	s.Utils.SendOK(c, result)
}

func (s *ServerModel) WriteoffDocumentReject(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentReject: ReadBody", "")
		return
	}
	documentID := int64(jsonMap["document_id"].(float64))
	comment, _ := jsonMap["comment"].(string)
	if err := s.Store.Repo().WriteoffDocumentReject(documentID, c.GetInt("user_id"), comment); err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentReject", "")
		return
	}
	s.Utils.SendOK(c, "ok")
}

func (s *ServerModel) WriteoffRecordsList(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffRecordsList: ReadBody", "")
		return
	}
	documentID := int64(0)
	if raw, ok := jsonMap["document_id"].(float64); ok {
		documentID = int64(raw)
	}
	limit := 500
	if raw, ok := jsonMap["limit"].(float64); ok && int(raw) > 0 {
		limit = int(raw)
	}
	data, err := s.Store.Repo().WriteoffRecordsList(documentID, limit)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffRecordsList", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WriteoffCatalogLines(c *gin.Context) {
	data, err := s.Store.Repo().WriteoffCatalogLines()
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffCatalogLines", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WriteoffCatalogItems(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffCatalogItems: ReadBody", "")
		return
	}
	lineID := int(jsonMap["line_id"].(float64))
	data, err := s.Store.Repo().WriteoffCatalogItems(lineID)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffCatalogItems", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WriteoffDocumentTemplate(c *gin.Context) {
	data, err := s.Store.Repo().WriteoffTemplateWorkbook()
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentTemplate", "")
		return
	}
	c.Header("Content-Disposition", `attachment; filename="writeoff_template.xlsx"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (s *ServerModel) WriteoffDocumentExport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentExport: ReadBody", "")
		return
	}
	documentID := int64(jsonMap["document_id"].(float64))
	data, err := s.Store.Repo().WriteoffExportWorkbook(documentID)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentExport", "")
		return
	}
	filename := "writeoff_" + strconv.FormatInt(documentID, 10) + ".xlsx"
	c.Header("Content-Disposition", `attachment; filename="`+filename+`"`)
	c.Data(200, "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet", data)
}

func (s *ServerModel) WriteoffDocumentImport(c *gin.Context) {
	documentIDRaw := strings.TrimSpace(c.PostForm("document_id"))
	if documentIDRaw == "" {
		s.Utils.SendError(c, errors.New("document_id talab qilinadi"), "WriteoffDocumentImport", "")
		return
	}
	documentID, err := strconv.ParseInt(documentIDRaw, 10, 64)
	if err != nil || documentID <= 0 {
		s.Utils.SendError(c, errors.New("document_id noto'g'ri"), "WriteoffDocumentImport", "")
		return
	}
	file, err := c.FormFile("file")
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentImport: FormFile", "")
		return
	}
	opened, err := file.Open()
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentImport: Open", "")
		return
	}
	defer opened.Close()
	data, err := io.ReadAll(opened)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentImport: ReadAll", "")
		return
	}
	result, err := s.Store.Repo().WriteoffImportWorkbook(documentID, data)
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffDocumentImport", "")
		return
	}
	if len(result.Errors) > 0 {
		s.Utils.SendError(c, errors.New("import xatoliklari"), "WriteoffDocumentImport", result)
		return
	}
	s.Utils.SendOK(c, result)
}

func (s *ServerModel) WriteoffIsResponsible(c *gin.Context) {
	ok, err := s.Store.Repo().WriteoffIsResponsible(c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "WriteoffIsResponsible", "")
		return
	}
	s.Utils.SendOK(c, map[string]bool{"is_responsible": ok})
}
