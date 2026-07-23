package api

import (
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

const t3SerialWSRoute = "/api/lines/t3/v2/serialcurrent/ws"

var t3SerialUpgrader = websocket.Upgrader{
	CheckOrigin: func(_ *http.Request) bool {
		return true
	},
}

func (s *ServerModel) publishT3V2Serial(serial string) {
	s.CurrentT3V2Serial = serial
	s.T3SerialHub.Broadcast(serial)
}

func (s *ServerModel) wsCheckPermission(c *gin.Context, routePath string) (int, error) {
	token := strings.TrimSpace(c.Query("token"))
	if token == "" {
		token = c.GetHeader("Authorization")
	}

	parsedToken, err := s.Utils.ParseToken(token)
	if err != nil {
		return 0, err
	}

	userID := parsedToken.UserID
	if userID == 0 {
		userID, err = s.Store.Repo().GetUserID(parsedToken.Login)
		if err != nil {
			return 0, err
		}
	}

	ok, err := s.Store.Repo().EnsureRouteAndCheckPermission(userID, routePath)
	if err != nil {
		return 0, err
	}
	if !ok {
		return 0, errors.New("no permission")
	}

	return userID, nil
}

func (s *ServerModel) LinesT3V2SerialCurrentWS(c *gin.Context) {
	if _, err := s.wsCheckPermission(c, t3SerialWSRoute); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "unauthorized"})
		return
	}

	conn, err := t3SerialUpgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		s.Logger.Warn().Err(err).Msg("t3_serial_ws_upgrade")
		return
	}
	defer conn.Close()

	if s.T3SerialHub == nil {
		return
	}

	s.T3SerialHub.Subscribe(conn)
	defer s.T3SerialHub.Unsubscribe(conn)

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			break
		}
	}
}
