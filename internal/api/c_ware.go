package api

import (
	"errors"
	"os"
	"strings"
	"time"

	"github.com/bingoohuang/xlsx"
	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/models"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
)

func (s *ServerModel) WareStockGetAll(c *gin.Context) {
	data, err := s.Store.Repo().WareStockGetAll()
	if err != nil {
		s.Utils.SendError(c, err, "WareStockGetAll", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareStockSnapshot(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareStockSnapshot: ReadBody", "")
		return
	}

	snapshotDate := ""
	if rawDate, ok := jsonMap["snapshot_date"].(string); ok {
		snapshotDate = strings.TrimSpace(rawDate)
	}
	if snapshotDate == "" {
		snapshotDate = time.Now().Format("2006-01-02")
	}

	data, err := s.Store.Repo().WareStockSnapshotGet(snapshotDate)
	if err != nil {
		s.Utils.SendError(c, err, "WareStockSnapshot", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareIncome(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareIncome: ReadBody", "")
		return
	}

	componentID := int(jsonMap["component_id"].(float64))
	quantity := getProductionFloat(jsonMap, "quantity")
	comment := ""
	if rawComment, ok := jsonMap["comment"].(string); ok {
		comment = rawComment
	}

	quantityAfter, err := s.Store.Repo().WareIncomeApply(componentID, c.GetInt("user_id"), quantity, comment)
	if err != nil {
		s.Utils.SendError(c, err, "WareIncome", "")
		return
	}
	s.Utils.SendOK(c, map[string]float64{"quantity_after": quantityAfter})
}

func (s *ServerModel) WareIncomeHistory(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareIncomeHistory: ReadBody", "")
		return
	}

	filter := store.WareIncomeFilter{}
	if rawComponentID, ok := jsonMap["component_id"].(float64); ok && int(rawComponentID) > 0 {
		filter.ComponentID = int(rawComponentID)
	}
	if rawDateFrom, ok := jsonMap["date_from"].(string); ok {
		filter.DateFrom = rawDateFrom
	}
	if rawDateTo, ok := jsonMap["date_to"].(string); ok {
		filter.DateTo = rawDateTo
	}

	data, err := s.Store.Repo().WareIncomeHistoryGet(filter)
	if err != nil {
		s.Utils.SendError(c, err, "WareIncomeHistory", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareGPProductBalanceLast(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareGPProductBalanceLast: ReadBody", "")
		return
	}

	limit := getProductionInt(jsonMap, "limit")
	if limit <= 0 {
		limit = 10
	}

	data, err := s.Store.Repo().WareGPProductBalanceLast(limit)
	if err != nil {
		s.Utils.SendError(c, err, "WareGPProductBalanceLast", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareGPProductBalanceHistory(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareGPProductBalanceHistory: ReadBody", "")
		return
	}

	limit := getProductionInt(jsonMap, "limit")
	if limit <= 0 {
		limit = 100
	}

	data, err := s.Store.Repo().WareGPProductBalanceHistory(limit)
	if err != nil {
		s.Utils.SendError(c, err, "WareGPProductBalanceHistory", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareGPProductReport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareGPProductReport: ReadBody", "")
		return
	}

	dateFrom, _ := jsonMap["date_from"].(string)
	dateTo, _ := jsonMap["date_to"].(string)
	limit := getProductionInt(jsonMap, "limit")
	if limit <= 0 {
		limit = 500
	}

	summary, err := s.Store.Repo().WareGPProductBalanceSummary()
	if err != nil {
		s.Utils.SendError(c, err, "WareGPProductReport: BalanceSummary", "")
		return
	}

	items, err := s.Store.Repo().WareGPProductTransactions(dateFrom, dateTo, limit)
	if err != nil {
		s.Utils.SendError(c, err, "WareGPProductReport: Transactions", "")
		return
	}

	s.Utils.SendOK(c, map[string]any{
		"summary": summary,
		"items":   items,
	})
}

func (s *ServerModel) WareDeliveryNotesList(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNotesList: ReadBody", "")
		return
	}

	limit := getProductionInt(jsonMap, "limit")
	data, err := s.Store.Repo().WareDeliveryNotesList(limit)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNotesList", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareDeliveryNoteItems(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItems: ReadBody", "")
		return
	}

	noteID := int64(getProductionInt(jsonMap, "delivery_note_id", "id"))
	data, err := s.Store.Repo().WareDeliveryNoteItemsGet(noteID)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItems", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareDeliveryNoteSnapshotsList(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteSnapshotsList: ReadBody", "")
		return
	}

	filter := store.WareDeliveryNoteSnapshotFilter{
		Limit: getProductionInt(jsonMap, "limit"),
	}
	if rawModelID, ok := jsonMap["model_id"].(float64); ok && int(rawModelID) > 0 {
		filter.ModelID = int(rawModelID)
	}
	if rawDateFrom, ok := jsonMap["date_from"].(string); ok {
		filter.DateFrom = strings.TrimSpace(rawDateFrom)
	}
	if rawDateTo, ok := jsonMap["date_to"].(string); ok {
		filter.DateTo = strings.TrimSpace(rawDateTo)
	}

	data, err := s.Store.Repo().WareDeliveryNoteSnapshotsList(filter)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteSnapshotsList", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareDeliveryNoteSnapshotItems(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteSnapshotItems: ReadBody", "")
		return
	}

	noteID := int64(getProductionInt(jsonMap, "delivery_note_id", "id"))
	data, err := s.Store.Repo().WareDeliveryNoteSnapshotItemsGet(noteID)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteSnapshotItems", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareDeliveryNoteItemUpdate(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemUpdate: ReadBody", "")
		return
	}

	itemID := int64(getProductionInt(jsonMap, "item_id", "id"))
	quantityActual := getProductionFloat(jsonMap, "quantity_actual", "quantity")
	if err := s.Store.Repo().WareDeliveryNoteItemUpdateQuantity(itemID, quantityActual); err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemUpdate", "")
		return
	}
	s.Utils.SendOK(c, map[string]bool{"updated": true})
}

func (s *ServerModel) WareDeliveryNoteItemConfirm(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemConfirm: ReadBody", "")
		return
	}

	itemID := int64(getProductionInt(jsonMap, "item_id", "id"))
	quantityActual := getProductionFloat(jsonMap, "quantity_actual", "quantity")
	if err := s.Store.Repo().WareDeliveryNoteItemConfirm(itemID, quantityActual); err != nil {
		var insufficient store.WareInsufficientStockError
		if errors.As(err, &insufficient) {
			s.Utils.SendError(c, err, "WareDeliveryNoteItemConfirm", insufficient.Shortages)
			return
		}
		s.Utils.SendError(c, err, "WareDeliveryNoteItemConfirm", "")
		return
	}
	s.Utils.SendOK(c, map[string]bool{"is_ready": true})
}

func parseWareDeliveryNoteBulkConfirmItems(rawItems any) ([]store.WareDeliveryNoteBulkConfirmItemInput, error) {
	itemsList, ok := rawItems.([]any)
	if !ok {
		return nil, nil
	}

	items := make([]store.WareDeliveryNoteBulkConfirmItemInput, 0, len(itemsList))
	for _, rawItem := range itemsList {
		itemMap, ok := rawItem.(map[string]any)
		if !ok {
			return nil, errors.New("noto'g'ri komponent formati")
		}

		itemID := int64(getProductionInt(itemMap, "item_id", "id"))
		if itemID <= 0 {
			continue
		}

		items = append(items, store.WareDeliveryNoteBulkConfirmItemInput{
			ItemID:         itemID,
			QuantityActual: getProductionFloat(itemMap, "quantity_actual", "quantity"),
		})
	}
	return items, nil
}

func (s *ServerModel) WareDeliveryNoteItemsConfirmAll(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemsConfirmAll: ReadBody", "")
		return
	}

	noteID := int64(getProductionInt(jsonMap, "delivery_note_id", "id"))
	items, err := parseWareDeliveryNoteBulkConfirmItems(jsonMap["items"])
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemsConfirmAll", "")
		return
	}

	count, err := s.Store.Repo().WareDeliveryNoteItemsConfirmAll(noteID, items)
	if err != nil {
		var insufficient store.WareInsufficientStockError
		if errors.As(err, &insufficient) {
			s.Utils.SendError(c, err, "WareDeliveryNoteItemsConfirmAll", insufficient.Shortages)
			return
		}
		s.Utils.SendError(c, err, "WareDeliveryNoteItemsConfirmAll", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"confirmed_count": count})
}

func (s *ServerModel) WareDeliveryNoteItemsUnconfirmAll(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemsUnconfirmAll: ReadBody", "")
		return
	}

	noteID := int64(getProductionInt(jsonMap, "delivery_note_id", "id"))
	count, err := s.Store.Repo().WareDeliveryNoteItemsUnconfirmAll(noteID)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemsUnconfirmAll", "")
		return
	}
	s.Utils.SendOK(c, map[string]int{"unconfirmed_count": count})
}

func (s *ServerModel) WareDeliveryNoteItemUnconfirm(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemUnconfirm: ReadBody", "")
		return
	}

	itemID := int64(getProductionInt(jsonMap, "item_id", "id"))
	if err := s.Store.Repo().WareDeliveryNoteItemUnconfirm(itemID); err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemUnconfirm", "")
		return
	}
	s.Utils.SendOK(c, map[string]bool{"is_ready": false})
}

func (s *ServerModel) WareDeliveryNoteItemDelete(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemDelete: ReadBody", "")
		return
	}

	itemID := int64(getProductionInt(jsonMap, "item_id", "id"))
	if err := s.Store.Repo().WareDeliveryNoteItemDelete(itemID); err != nil {
		s.Utils.SendError(c, err, "WareDeliveryNoteItemDelete", "")
		return
	}
	s.Utils.SendOK(c, map[string]bool{"deleted": true})
}

func (s *ServerModel) WareUpload(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareUpload: ReadBody", "")
		return
	}

	userID := c.GetInt("user_id")
	file64 := jsonMap["file64"].(string)

	data, err := utils.Base64Decode(file64)
	if err != nil {
		s.Utils.SendError(c, err, "WareUpload: Base64Decode", "")
		return
	}

	err = os.WriteFile("ware_income.xlsx", []byte(data), 0644)
	if err != nil {
		s.Utils.SendError(c, err, "WareUpload: WriteFile", "")
		return
	}

	var file []models.WareIncome
	x, _ := xlsx.New(xlsx.WithInputFile("ware_income.xlsx"))
	defer x.Close()

	if err := x.Read(&file); err != nil {
		s.Utils.SendError(c, err, "WareUpload: Read", "")
		return
	}

	errComponents := []string{}
	for i, row := range file {
		component, err := s.Store.Repo().ComponentsGetByFactoryCode(row.FactoryCode)
		if err != nil || component.ID == 0 {
			errComponents = append(errComponents, row.FactoryCode)
			continue
		}
		file[i].ComponentId = component.ID
	}
	if len(errComponents) > 0 {
		s.Utils.SendError(c, errors.New(strings.Join(errComponents, ", ")), "WareUpload", errComponents)
		return
	}

	added := 0
	for _, row := range file {
		if row.Quantity <= 0 {
			continue
		}
		_, err := s.Store.Repo().WareIncomeApply(row.ComponentId, userID, row.Quantity, "Excel import")
		if err != nil {
			s.Utils.SendError(c, err, "WareUpload: WareIncomeApply", "")
			return
		}
		added++
	}

	s.Utils.SendOK(c, map[string]int{"added": added})
}

func parseWareDeliveryNoteItems(rawItems any) ([]store.WareDeliveryNoteItemInput, error) {
	itemsList, ok := rawItems.([]any)
	if !ok || len(itemsList) == 0 {
		return nil, errors.New("komponentlar ro'yxati bo'sh")
	}

	items := make([]store.WareDeliveryNoteItemInput, 0, len(itemsList))
	for _, rawItem := range itemsList {
		itemMap, ok := rawItem.(map[string]any)
		if !ok {
			return nil, errors.New("noto'g'ri komponent formati")
		}

		quantity := getProductionFloat(itemMap, "needed_quantity", "quantity")

		lineName := ""
		if rawLineName, ok := itemMap["line_name"].(string); ok {
			lineName = rawLineName
		}
		componentName := ""
		if rawComponentName, ok := itemMap["component_name"].(string); ok {
			componentName = rawComponentName
		}
		manufacturerCode := ""
		if rawManufacturerCode, ok := itemMap["manufacturer_code"].(string); ok {
			manufacturerCode = rawManufacturerCode
		}
		factoryCode := ""
		if rawFactoryCode, ok := itemMap["factory_code"].(string); ok {
			factoryCode = rawFactoryCode
		}

		items = append(items, store.WareDeliveryNoteItemInput{
			ComponentID:      getProductionInt(itemMap, "component_id"),
			LineID:           getProductionInt(itemMap, "line_id"),
			LineName:         lineName,
			ComponentName:    componentName,
			ManufacturerCode: manufacturerCode,
			FactoryCode:      factoryCode,
			Quantity:         quantity,
		})
	}

	return store.ValidateWareDeliveryNoteItems(items)
}

func (s *ServerModel) WareRequestConfirm(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareRequestConfirm: ReadBody", "")
		return
	}

	modelID := getProductionInt(jsonMap, "model_id")
	modelName := ""
	if rawModelName, ok := jsonMap["model_name"].(string); ok {
		modelName = strings.TrimSpace(rawModelName)
	}
	modelQuantity := getProductionFloat(jsonMap, "model_quantity", "quantity")

	items, err := parseWareDeliveryNoteItems(jsonMap["items"])
	if err != nil {
		s.Utils.SendError(c, err, "WareRequestConfirm", "")
		return
	}

	data, err := s.Store.Repo().WareDeliveryNoteCreate(store.WareDeliveryNoteCreateInput{
		ModelID:       modelID,
		ModelName:     modelName,
		ModelQuantity: modelQuantity,
		UserID:        c.GetInt("user_id"),
		Items:         items,
	})
	if err != nil {
		var insufficient store.WareInsufficientStockError
		if errors.As(err, &insufficient) {
			s.Utils.SendError(c, err, "WareRequestConfirm", insufficient.Shortages)
			return
		}
		var invalidQty store.WareInvalidQuantityError
		if errors.As(err, &invalidQty) {
			s.Utils.SendError(c, err, "WareRequestConfirm", invalidQty.Items)
			return
		}
		s.Utils.SendError(c, err, "WareRequestConfirm", "")
		return
	}
	s.Utils.SendOK(c, data)
}

func (s *ServerModel) WareRequestBuild(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "WareRequestBuild: ReadBody", "")
		return
	}

	modelID := getProductionInt(jsonMap, "model_id")
	quantity := getProductionFloat(jsonMap, "quantity")
	if modelID == 0 {
		s.Utils.SendError(c, errors.New("model_id is required"), "WareRequestBuild", "")
		return
	}
	if quantity <= 0 {
		s.Utils.SendError(c, errors.New("miqdor 0 dan katta bo'lishi kerak"), "WareRequestBuild", "")
		return
	}

	data, err := s.Store.Repo().WareRequestBuild(modelID, quantity)
	if err != nil {
		s.Utils.SendError(c, err, "WareRequestBuild", "")
		return
	}
	s.Utils.SendOK(c, data)
}
