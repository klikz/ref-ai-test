package api

import (
	"errors"
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *ServerModel) LabInfo(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "LabInfo: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	if serial == "" {
		s.Utils.SendError(c, errors.New("serial bo'sh"), "LabInfo", "")
		return
	}

	info, err := s.Store.Repo().LabInfoLookup(serial)
	if err != nil {
		s.Utils.SendError(c, err, "LabInfo", "")
		return
	}

	s.Utils.SendOK(c, info)
}
