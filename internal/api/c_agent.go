package api

import (
	"errors"
	"os"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
)

func agentOwnerIDs() map[int]struct{} {
	raw := strings.TrimSpace(os.Getenv("AGENT_OWNER_USER_IDS"))
	result := make(map[int]struct{})
	if raw == "" {
		return result
	}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		id, err := strconv.Atoi(part)
		if err != nil || id <= 0 {
			continue
		}
		result[id] = struct{}{}
	}
	return result
}

func (s *ServerModel) agentIsOwner(userID int) bool {
	owners := agentOwnerIDs()
	if len(owners) == 0 {
		return false
	}
	_, ok := owners[userID]
	return ok
}

func (s *ServerModel) agentRequireOwner(c *gin.Context) bool {
	userID := c.GetInt("user_id")
	if !s.agentIsOwner(userID) {
		s.Utils.SendError(c, errors.New("faqat agent egasi uchun"), "agentRequireOwner", nil)
		return false
	}
	return true
}

func (s *ServerModel) AgentAccess(c *gin.Context) {
	userID := c.GetInt("user_id")
	allowed := s.agentIsOwner(userID)
	s.Utils.SendOK(c, map[string]any{
		"allowed": allowed,
		"user_id": userID,
	})
}

func (s *ServerModel) AgentTasksList(c *gin.Context) {
	if !s.agentRequireOwner(c) {
		return
	}
	items, err := s.Store.Repo().AgentTaskList(50)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksList", "")
		return
	}
	s.Utils.SendOK(c, items)
}

func (s *ServerModel) AgentTasksCreate(c *gin.Context) {
	if !s.agentRequireOwner(c) {
		return
	}
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksCreate: ReadBody", "")
		return
	}
	prompt := getProductionString(jsonMap, "prompt")
	origin := getProductionString(jsonMap, "origin")
	if origin == "" {
		origin = "server"
	}
	item, err := s.Store.Repo().AgentTaskCreate(prompt, origin, c.GetInt("user_id"))
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksCreate", "")
		return
	}
	if origin == "pc" {
		item, err = s.Store.Repo().AgentTaskUpdateStatus(
			item.ID,
			"ready_for_test",
			"PC vazifasi qabul qilindi. Cursor da bajaring, commit/push qiling, keyin testga chiqaring.",
			"",
			"",
		)
		if err != nil {
			s.Utils.SendError(c, err, "AgentTasksCreate: status", "")
			return
		}
	} else {
		item, err = s.Store.Repo().AgentTaskUpdateStatus(
			item.ID,
			"queued",
			"Navbatda. Cursor SDK worker (ref-ai-agent-worker) olib bajaradi.",
			"",
			"",
		)
		if err != nil {
			s.Utils.SendError(c, err, "AgentTasksCreate: queued", "")
			return
		}
	}
	s.Utils.SendOK(c, item)
}

func (s *ServerModel) AgentTasksGet(c *gin.Context) {
	if !s.agentRequireOwner(c) {
		return
	}
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksGet: ReadBody", "")
		return
	}
	id := int64(getProductionInt(jsonMap, "id"))
	if id == 0 {
		s.Utils.SendError(c, errors.New("id is required"), "AgentTasksGet", "")
		return
	}
	item, err := s.Store.Repo().AgentTaskByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksGet", "")
		return
	}
	s.Utils.SendOK(c, item)
}

func (s *ServerModel) AgentTasksApproveTest(c *gin.Context) {
	if !s.agentRequireOwner(c) {
		return
	}
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveTest: ReadBody", "")
		return
	}
	id := int64(getProductionInt(jsonMap, "id"))
	if id == 0 {
		s.Utils.SendError(c, errors.New("id is required"), "AgentTasksApproveTest", "")
		return
	}
	item, err := s.Store.Repo().AgentTaskByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveTest: get", "")
		return
	}
	if item.Status != "ready_for_test" && item.Status != "failed" {
		s.Utils.SendError(c, errors.New("vazifa testga chiqarish holatida emas"), "AgentTasksApproveTest", item)
		return
	}
	item, err = s.Store.Repo().AgentTaskUpdateStatus(
		id,
		"ready_for_prod",
		"Test tasdiqlandi. Serverda: git pull → build → pm2 restart ref-ai-test, keyin prodga chiqaring.",
		"",
		"",
	)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveTest", "")
		return
	}
	s.Utils.SendOK(c, item)
}

func (s *ServerModel) AgentTasksApproveProd(c *gin.Context) {
	if !s.agentRequireOwner(c) {
		return
	}
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveProd: ReadBody", "")
		return
	}
	id := int64(getProductionInt(jsonMap, "id"))
	if id == 0 {
		s.Utils.SendError(c, errors.New("id is required"), "AgentTasksApproveProd", "")
		return
	}
	item, err := s.Store.Repo().AgentTaskByID(id)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveProd: get", "")
		return
	}
	if item.Status != "ready_for_prod" {
		s.Utils.SendError(c, errors.New("vazifa prodga chiqarish holatida emas"), "AgentTasksApproveProd", item)
		return
	}
	item, err = s.Store.Repo().AgentTaskUpdateStatus(
		id,
		"promoted",
		"Prod tasdiqlandi. Serverda promote-to-prod.ps1 / ref-ai-prod update qiling.",
		"",
		"",
	)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveProd", "")
		return
	}
	s.Utils.SendOK(c, item)
}

func (s *ServerModel) AgentTasksReject(c *gin.Context) {
	if !s.agentRequireOwner(c) {
		return
	}
	jsonMap, err := s.Utils.ReadBody(c)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksReject: ReadBody", "")
		return
	}
	id := int64(getProductionInt(jsonMap, "id"))
	reason := getProductionString(jsonMap, "reason", "comment")
	if id == 0 {
		s.Utils.SendError(c, errors.New("id is required"), "AgentTasksReject", "")
		return
	}
	item, err := s.Store.Repo().AgentTaskUpdateStatus(id, "rejected", "", reason, "")
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksReject", "")
		return
	}
	s.Utils.SendOK(c, item)
}
