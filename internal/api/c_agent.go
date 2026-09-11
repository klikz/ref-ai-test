package api

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
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

func agentAppDir() string {
	if v := strings.TrimSpace(os.Getenv("AGENT_APP_DIR")); v != "" {
		return v
	}
	// Common layout: <root>/bin/api_v3.exe and <root>/app/...
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}
	candidate := filepath.Join(cwd, "app")
	if st, err := os.Stat(filepath.Join(candidate, "scripts", "agent-deploy-test.ps1")); err == nil && !st.IsDir() {
		return candidate
	}
	if st, err := os.Stat(filepath.Join(cwd, "scripts", "agent-deploy-test.ps1")); err == nil && !st.IsDir() {
		return cwd
	}
	return cwd
}

func (s *ServerModel) agentRunDeployTest(taskID int64) {
	appDir := agentAppDir()
	script := filepath.Join(appDir, "scripts", "agent-deploy-test.ps1")
	_, _ = s.Store.Repo().AgentTaskUpdateStatus(
		taskID,
		"testing",
		"Test deploy boshlandi: git/build/pm2...\nappDir="+appDir,
		"",
		"",
	)

	if _, err := os.Stat(script); err != nil {
		_, _ = s.Store.Repo().AgentTaskUpdateStatus(
			taskID,
			"failed",
			"",
			"deploy script topilmadi: "+script,
			"",
		)
		return
	}

	cmd := exec.Command(
		"powershell.exe",
		"-NoProfile",
		"-ExecutionPolicy", "Bypass",
		"-File", script,
	)
	cmd.Dir = appDir
	cmd.Env = append(os.Environ(),
		"AGENT_APP_DIR="+appDir,
		"AGENT_TASK_ID="+strconv.FormatInt(taskID, 10),
		// Status DB ga yozilguncha process o'lmasin — pm2 ni Go keyin ishga tushiradi.
		"AGENT_SKIP_PM2=1",
	)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf

	err := cmd.Run()
	out := buf.String()
	if len(out) > 6000 {
		out = out[len(out)-6000:]
	}
	if err != nil {
		_, _ = s.Store.Repo().AgentTaskUpdateStatus(
			taskID,
			"failed",
			out,
			"Test deploy xato: "+err.Error(),
			"",
		)
		return
	}

	pm2Name := strings.TrimSpace(os.Getenv("PM2_TEST_NAME"))
	if pm2Name == "" {
		pm2Name = "ref-ai-test"
	}

	// Avval DB status — keyin pm2 (restart processni o'ldiradi).
	_, _ = s.Store.Repo().AgentTaskUpdateStatus(
		taskID,
		"ready_for_prod",
		"Test deploy OK (pm2 restart...).\n"+out+"\nEndi Prodga tugmasini bosishingiz mumkin.",
		"",
		"",
	)

	pm2Out, pm2Err := exec.Command("pm2", "restart", pm2Name, "--update-env").CombinedOutput()
	if pm2Err != nil {
		msg := out + "\npm2 restart:\n" + string(pm2Out)
		if len(msg) > 6000 {
			msg = msg[len(msg)-6000:]
		}
		_, _ = s.Store.Repo().AgentTaskUpdateStatus(
			taskID,
			"failed",
			msg,
			"Build OK, lekin pm2 restart xato: "+pm2Err.Error(),
			"",
		)
		return
	}
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
		"testing",
		"Test deploy navbatga qo'yildi (git/build/pm2)...",
		"",
		"",
	)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveTest: testing", "")
		return
	}

	go s.agentRunDeployTest(id)

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
