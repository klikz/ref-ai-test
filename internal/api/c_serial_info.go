package api

import (
	"strings"

	"github.com/gin-gonic/gin"
)

func (s *ServerModel) SerialInfo(c *gin.Context) {
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "SerialInfo: ReadBody", "")
		return
	}

	serial, _ := jsonMap["serial"].(string)
	serial = strings.TrimSpace(serial)
	if serial == "" {
		s.Utils.SendError(c, nil, "SerialInfo", "serial bo'sh")
		return
	}

	info, err := s.Store.Repo().SerialInfoLookup(serial)
	if err != nil {
		s.Utils.SendError(c, err, "SerialInfo", "")
		return
	}

	s.Utils.SendOK(c, info)
}
