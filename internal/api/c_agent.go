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

func agentBinaryName() string {
	if v := strings.TrimSpace(os.Getenv("APP_BINARY")); v != "" {
		return v
	}
	return "api_v3.exe"
}

// agentRootFromAppDir: layout is <root>/bin + <root>/app/...
func agentRootFromAppDir(appDir string) string {
	base := filepath.Base(appDir)
	if strings.EqualFold(base, "app") {
		return filepath.Dir(appDir)
	}
	return appDir
}

// agentSwapNewBinary renames bin/api_v3.exe.new -> api_v3.exe if present.
func agentSwapNewBinary(rootDir string) error {
	bin := agentBinaryName()
	newer := filepath.Join(rootDir, "bin", bin+".new")
	dest := filepath.Join(rootDir, "bin", bin)
	if _, err := os.Stat(newer); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	_ = os.Remove(dest)
	return os.Rename(newer, dest)
}

// agentPm2RestartClean restarts without --update-env so caller env
// (e.g. test PORT=3081) cannot override the target app's .env.
func agentPm2RestartClean(name string) (string, error) {
	out, err := exec.Command("pm2", "restart", name).CombinedOutput()
	return string(out), err
}

func agentPm2StartEco(prodDir, eco string) (string, error) {
	cmd := exec.Command("pm2", "start", eco)
	cmd.Dir = prodDir
	out, err := cmd.CombinedOutput()
	return string(out), err
}

func (s *ServerModel) agentRunDeployTest(taskID int64) {
	appDir := agentAppDir()
	script := filepath.Join(appDir, "scripts", "agent-deploy-test.ps1")
	_, _ = s.Store.Repo().AgentTaskUpdateStatus(
		taskID,
		"testing",
		"Test deploy boshlandi: build/copy (git skip)...\nappDir="+appDir,
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
		// PM2/SYSTEM ostida git osilib qoladi — deployda o'chirilgan.
		"GIT_TERMINAL_PROMPT=0",
		"GCM_INTERACTIVE=never",
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
	rootDir := agentRootFromAppDir(appDir)

	// Avval DB status — keyin stop/swap/restart (restart processni o'ldiradi).
	_, _ = s.Store.Repo().AgentTaskUpdateStatus(
		taskID,
		"ready_for_prod",
		"Test deploy OK (pm2 swap+restart...).\n"+out+"\nEndi Prodga tugmasini bosishingiz mumkin.",
		"",
		"",
	)

	_, _ = exec.Command("pm2", "stop", pm2Name).CombinedOutput()
	if swapErr := agentSwapNewBinary(rootDir); swapErr != nil {
		_, _ = s.Store.Repo().AgentTaskUpdateStatus(
			taskID,
			"failed",
			out,
			"Build OK, lekin binary swap xato: "+swapErr.Error(),
			"",
		)
		return
	}
	// No --update-env: keep PORT/CONN_STRING from the test app .env.
	pm2Out, pm2Err := agentPm2RestartClean(pm2Name)
	if pm2Err != nil {
		msg := out + "\npm2 restart:\n" + pm2Out
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
		"Test deploy navbatga qo'yildi (build/copy, git skip)...",
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

func (s *ServerModel) agentRunDeployProd(taskID int64) {
	appDir := agentAppDir()
	script := filepath.Join(appDir, "scripts", "agent-deploy-prod.ps1")
	testDir := strings.TrimSpace(os.Getenv("REF_AI_TEST_DIR"))
	if testDir == "" {
		testDir = "D:\\ref-main\\ref-ai\\ref-ai-test"
	}
	prodDir := strings.TrimSpace(os.Getenv("REF_AI_PROD_DIR"))
	if prodDir == "" {
		prodDir = "D:\\ref-main\\ref-ai\\ref-ai-prod"
	}

	_, _ = s.Store.Repo().AgentTaskUpdateStatus(
		taskID,
		"promoting",
		"Prod deploy: test artifaktlari -> "+prodDir,
		"",
		"",
	)

	if _, err := os.Stat(script); err != nil {
		_, _ = s.Store.Repo().AgentTaskUpdateStatus(
			taskID,
			"ready_for_prod",
			"",
			"prod deploy script topilmadi: "+script,
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
		"REF_AI_TEST_DIR="+testDir,
		"REF_AI_PROD_DIR="+prodDir,
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
			"ready_for_prod",
			out,
			"Prod deploy xato: "+err.Error(),
			"",
		)
		return
	}

	pm2Name := strings.TrimSpace(os.Getenv("PM2_PROD_NAME"))
	if pm2Name == "" {
		pm2Name = "ref-ai-prod"
	}

	_, _ = s.Store.Repo().AgentTaskUpdateStatus(
		taskID,
		"promoted",
		"Prod deploy OK (pm2...).\n"+out,
		"",
		"",
	)

	// Script already stopped prod to unlock the exe. Restart WITHOUT --update-env
	// so test API env (PORT=3081, dbname=ref-test) cannot leak into prod.
	pm2Out, pm2Err := agentPm2RestartClean(pm2Name)
	if pm2Err != nil {
		eco := filepath.Join(prodDir, "ecosystem.config.cjs")
		startOut, startErr := agentPm2StartEco(prodDir, eco)
		msg := out + "\npm2 restart:\n" + pm2Out + "\npm2 start:\n" + startOut
		if len(msg) > 6000 {
			msg = msg[len(msg)-6000:]
		}
		if startErr != nil {
			_, _ = s.Store.Repo().AgentTaskUpdateStatus(
				taskID,
				"ready_for_prod",
				msg,
				"Copy OK, lekin pm2 start/restart xato: "+startErr.Error(),
				"",
			)
			return
		}
		_, _ = s.Store.Repo().AgentTaskUpdateStatus(
			taskID,
			"promoted",
			"Prod deploy OK (pm2 start).\n"+msg,
			"",
			"",
		)
	}
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
		"promoting",
		"Prod deploy navbatga qo'yildi (copy test → ref-ai-prod)...",
		"",
		"",
	)
	if err != nil {
		s.Utils.SendError(c, err, "AgentTasksApproveProd: promoting", "")
		return
	}

	go s.agentRunDeployProd(id)

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
