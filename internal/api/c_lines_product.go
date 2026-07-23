package api

import (
	"github.com/gin-gonic/gin"
	"github.com/klikz/api_v3/internal/store"
)

type productBalanceResponse struct {
	Summary []store.ProductBalanceSummaryRow `json:"summary"`
	Items   []store.ProductBalanceItem       `json:"items"`
}

func (s *ServerModel) LinesProductBalance(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesProductBalance: ReadBody", "")
		return
	}

	lineID := int(jsonMap["line_id"].(float64))

	summary, err := s.Store.Repo().ProductBalanceSummary(lineID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesProductBalance: ProductBalanceSummary", "")
		return
	}

	items, err := s.Store.Repo().ProductBalanceByLine(lineID)
	if err != nil {
		s.Utils.SendError(c, err, "LinesProductBalance: ProductBalanceByLine", "")
		return
	}

	s.Utils.SendOK(c, productBalanceResponse{
		Summary: summary,
		Items:   items,
	})
}

func (s *ServerModel) LinesProductReport(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LinesProductReport: ReadBody", "")
		return
	}

	lineID := 0
	if raw, ok := jsonMap["line_id"].(float64); ok {
		lineID = int(raw)
	}

	dateFrom, _ := jsonMap["date_from"].(string)
	dateTo, _ := jsonMap["date_to"].(string)

	data, err := s.Store.Repo().ProductTransferReport(lineID, dateFrom, dateTo)
	if err != nil {
		s.Utils.SendError(c, err, "LinesProductReport", "")
		return
	}

	s.Utils.SendOK(c, data)
}
