package main

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"github.com/klikz/api_v3/internal/api"
	"github.com/klikz/api_v3/internal/store"
	"github.com/klikz/api_v3/utils"
	_ "github.com/lib/pq"
	"github.com/rs/zerolog"
)

const (
	testPort          = "18081"
	testModelID       = 107
	testFinComponent  = 1050
	testRadComponent  = 2051
	testRadConfigID   = 2
	i1PrinterID       = 13
	t1PrinterID       = 16
	t2PrinterID       = 21
	t3PrinterID       = 14
	finPrinterID      = 18
	radiatorPrinterID = 11
)

type checkResult struct {
	Area    string `json:"area"`
	Name    string `json:"name"`
	Status  string `json:"status"`
	Details string `json:"details"`
}

type runner struct {
	db        *sql.DB
	repo      *store.Repo
	baseURL   string
	token     string
	userID    int
	runID     string
	startedAt time.Time
	outputDir string
	checks    []checkResult
	client    *http.Client
}

func main() {
	if err := godotenv.Load(); err != nil {
		panic(err)
	}
	connString, err := utils.DBConnString()
	if err != nil {
		panic(err)
	}
	db, err := sql.Open("postgres", connString)
	if err != nil {
		panic(err)
	}
	defer db.Close()
	if err := db.Ping(); err != nil {
		panic(err)
	}

	now := time.Now()
	outputDir, err := filepath.Abs(filepath.Join(
		"test-output",
		"production-flow-"+now.Format("20060102-150405"),
	))
	if err != nil {
		panic(err)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		panic(err)
	}
	os.Setenv("LABEL_PDF_OUTPUT_DIR", outputDir)
	os.Setenv("T3_CAMERA_ENABLED", "false")
	os.Setenv("SECRET_KEY", strings.TrimSpace(os.Getenv("SECRET_KEY")))

	appStore := store.New(db)
	r := &runner{
		db:        db,
		repo:      appStore.Repo(),
		baseURL:   "http://127.0.0.1:" + testPort,
		runID:     "ZZTEST_" + now.Format("20060102_150405"),
		startedAt: now,
		outputDir: outputDir,
		client:    &http.Client{Timeout: 60 * time.Second},
	}

	if err := r.prepareAuth(); err != nil {
		panic(err)
	}

	logger := zerolog.Nop()
	go func() {
		if err := api.StartSrv(testPort, "debug", &logger, *appStore); err != nil {
			fmt.Fprintln(os.Stderr, "test API:", err)
		}
	}()
	if err := waitForPort("127.0.0.1:"+testPort, 20*time.Second); err != nil {
		panic(err)
	}

	r.run()
	if err := r.writeResults(); err != nil {
		panic(err)
	}

	pass, fail := 0, 0
	for _, check := range r.checks {
		if check.Status == "PASS" {
			pass++
		} else {
			fail++
		}
		fmt.Printf("%s | %s | %s | %s\n", check.Status, check.Area, check.Name, check.Details)
	}
	fmt.Printf("RESULTS_FILE=%s\n", filepath.Join(outputDir, "results.json"))
	fmt.Printf("PDF_DIR=%s\n", outputDir)
	fmt.Printf("SUMMARY=pass:%d fail:%d\n", pass, fail)
	if fail > 0 {
		os.Exit(2)
	}
}

func (r *runner) run() {
	r.testPlan()
	r.testProductLines()
	r.testAuxiliaryChain()
	r.testGSCodeConcurrency()
}

func (r *runner) prepareAuth() error {
	if err := r.db.QueryRow(`SELECT id FROM auth.users WHERE status = true ORDER BY id LIMIT 1`).Scan(&r.userID); err != nil {
		return err
	}
	routes := []string{
		"/api/production/plan/save",
		"/api/production/plan/day",
		"/api/production/plan/month",
		"/api/production/plan/dashboard",
		"/api/production/plan/report",
		"/api/production/plan/lock-day",
		"/api/production/plan/unlock-day",
		"/api/lines/ichki/v2/print",
		"/api/lines/ichki/v2/reprint",
		"/api/lines/t1/v2/serialprint",
		"/api/lines/t1/v2/serialreprint",
		"/api/lines/t2/v2/serialprint",
		"/api/lines/t2/v2/serialreprint",
		"/api/lines/t3/v2/serialprint",
		"/api/lines/t3/v2/serialreprint",
		"/api/production/fin_press/print",
		"/api/production/fin_press/reprint",
		"/api/lines/radiator/receive",
		"/api/lines/radiator/v2/print",
		"/api/lines/radiator/v2/reprint",
	}
	for _, route := range routes {
		if _, err := r.db.Exec(`
			INSERT INTO auth.routes (route)
			SELECT $1::text
			WHERE NOT EXISTS (SELECT 1 FROM auth.routes WHERE route = $1::text)`, route); err != nil {
			return fmt.Errorf("route %s: %w", route, err)
		}
		if _, err := r.db.Exec(`
			INSERT INTO auth.permissions (user_id, route_id)
			SELECT $1, id
			FROM auth.routes ar
			WHERE ar.route = $2
			  AND NOT EXISTS (
				SELECT 1 FROM auth.permissions p
				WHERE p.user_id = $1 AND p.route_id = ar.id
			  )`, r.userID, route); err != nil {
			return fmt.Errorf("permission %s: %w", route, err)
		}
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login":   r.runID,
		"user_id": r.userID,
		"nbf":     time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})
	signed, err := token.SignedString([]byte(os.Getenv("SECRET_KEY")))
	if err != nil {
		return err
	}
	r.token = signed
	return nil
}

func (r *runner) testPlan() {
	today := time.Now().Format("2006-01-02")
	targets := []struct {
		lineID      int
		modelID     int
		componentID int
	}{
		{4, testModelID, 0},
		{5, testModelID, 0},
		{6, testModelID, 0},
		{7, testModelID, 0},
		{8, 0, testFinComponent},
		{9, 0, testRadComponent},
	}
	for _, target := range targets {
		_, _ = r.post("/api/production/plan/unlock-day", map[string]any{
			"line_id": target.lineID, "plan_date": today,
		})
		resp, err := r.post("/api/production/plan/save", map[string]any{
			"line_id": target.lineID,
			"cells": []map[string]any{{
				"plan_date": today, "model_id": target.modelID,
				"component_id": target.componentID, "planned_qty": 10000,
				"allow_overplan": false,
			}},
		})
		r.recordAPI("Reja", fmt.Sprintf("line %d draft save", target.lineID), resp, err, true)
		resp, err = r.post("/api/production/plan/lock-day", map[string]any{
			"line_id": target.lineID, "plan_date": today,
		})
		r.recordAPI("Reja", fmt.Sprintf("line %d lock", target.lineID), resp, err, true)
	}

	resp, err := r.post("/api/production/plan/day", map[string]any{
		"line_id": 7, "plan_date": today,
	})
	locked := err == nil && apiOK(resp) && nestedString(resp, "data", "status") == "locked"
	r.record("Reja", "locked day read", locked, responseDetail(resp, err))

	resp, err = r.post("/api/production/plan/save", map[string]any{
		"line_id": 7,
		"cells": []map[string]any{{
			"plan_date": today, "model_id": testModelID, "planned_qty": 9999,
		}},
	})
	r.record("Reja", "locked day edit rejected", err == nil && !apiOK(resp), responseDetail(resp, err))

	for path, payload := range map[string]map[string]any{
		"/api/production/plan/month": {
			"line_id": 7, "year_month": today[:7], "include_actual": true,
		},
		"/api/production/plan/dashboard": {"plan_date": today},
		"/api/production/plan/report": {
			"date_from": today, "date_to": today, "line_ids": []int{4, 5, 6, 7, 8, 9},
		},
	} {
		resp, err := r.post(path, payload)
		r.recordAPI("Reja", path, resp, err, true)
	}
}

func (r *runner) testProductLines() {
	pdfBefore := r.pdfCount()
	resp, err := r.post("/api/lines/ichki/v2/print", map[string]any{
		"model_id": testModelID, "printer_v2_id": i1PrinterID, "quantity": 2,
	})
	r.recordAPI("I1", "quantity=2 print", resp, err, true)
	i1Serials, i1Codes, queryErr := r.latestI1Products()
	r.record(
		"I1",
		"two serials have distinct GSCODE",
		queryErr == nil && len(i1Serials) == 2 && len(i1Codes) == 2 && i1Codes[0] != i1Codes[1],
		fmt.Sprintf("serials=%v gscode_ids=%v err=%v", i1Serials, i1Codes, queryErr),
	)
	r.record("I1", "two PDFs created", r.pdfCount()-pdfBefore == 2, fmt.Sprintf("delta=%d", r.pdfCount()-pdfBefore))
	if len(i1Serials) > 0 {
		before := r.pdfCount()
		resp, err = r.post("/api/lines/ichki/v2/reprint", map[string]any{
			"serial": i1Serials[0], "printer_v2_id": i1PrinterID,
		})
		r.recordAPI("I1", "reprint", resp, err, true)
		r.record("I1", "reprint PDF created", r.pdfCount() == before+1, fmt.Sprintf("delta=%d", r.pdfCount()-before))
	}

	before := r.pdfCount()
	resp, err = r.post("/api/lines/t1/v2/serialprint", map[string]any{
		"model_id": testModelID, "printer_v2_id": t1PrinterID, "copy": 1,
	})
	r.recordAPI("T1", "serial print", resp, err, true)
	serial := nestedString(resp, "data", "serial")
	r.record("T1", "serial returned", serial != "", "serial="+serial)
	r.record("T1", "PDF created", r.pdfCount() == before+1, fmt.Sprintf("delta=%d", r.pdfCount()-before))

	if serial == "" {
		return
	}
	for _, test := range []struct {
		area    string
		path    string
		printer int
		payload map[string]any
	}{
		{"T1", "/api/lines/t1/v2/serialreprint", t1PrinterID, map[string]any{"serial": serial}},
		{"T2", "/api/lines/t2/v2/serialprint", t2PrinterID, map[string]any{"serial": serial, "copy": 1}},
		{"T2", "/api/lines/t2/v2/serialreprint", t2PrinterID, map[string]any{"serial": serial}},
		{"T3", "/api/lines/t3/v2/serialprint", t3PrinterID, map[string]any{
			"serial": serial, "acc_serial": r.uniqueAccSerial(), "copy": 1,
		}},
		{"T3", "/api/lines/t3/v2/serialreprint", t3PrinterID, map[string]any{"serial": serial}},
	} {
		test.payload["printer_v2_id"] = test.printer
		before = r.pdfCount()
		resp, err = r.post(test.path, test.payload)
		r.recordAPI(test.area, test.path, resp, err, true)
		r.record(test.area, test.path+" PDF", r.pdfCount() == before+1, fmt.Sprintf("delta=%d", r.pdfCount()-before))
	}

	statuses, statusErr := r.productStatuses(serial)
	r.record(
		"T1-T3",
		"transfer statuses",
		statusErr == nil && statuses[4] == "transferred" && statuses[5] == "transferred" && statuses[6] == "active",
		fmt.Sprintf("statuses=%v err=%v", statuses, statusErr),
	)

	resp, err = r.post("/api/lines/t2/v2/serialprint", map[string]any{
		"serial": r.runID + "_MISSING", "printer_v2_id": t2PrinterID,
	})
	r.record("T2", "missing serial rejected", err == nil && !apiOK(resp), responseDetail(resp, err))

	resp, err = r.post("/api/lines/t3/v2/serialprint", map[string]any{
		"serial": serial, "acc_serial": "WRONG", "printer_v2_id": t3PrinterID,
	})
	r.record("T3", "wrong accessory rejected", err == nil && !apiOK(resp), responseDetail(resp, err))
}

func (r *runner) testAuxiliaryChain() {
	before := r.pdfCount()
	resp, err := r.post("/api/production/fin_press/print", map[string]any{
		"component_id": testFinComponent, "count": 3,
		"printer_v2_id": finPrinterID, "line_id": 8,
	})
	r.recordAPI("Fin Press", "print count=3", resp, err, true)
	sessionID := nestedInt64(resp, "data", "session_id")
	r.record("Fin Press", "PDF created", r.pdfCount() == before+1, fmt.Sprintf("session=%d delta=%d", sessionID, r.pdfCount()-before))
	if sessionID <= 0 {
		return
	}

	balanceBefore := r.balance(8, testFinComponent)
	before = r.pdfCount()
	reprintResp, reprintErr := r.post("/api/production/fin_press/reprint", map[string]any{
		"session_id": sessionID, "printer_v2_id": finPrinterID, "line_id": 8,
	})
	r.recordAPI("Fin Press", "reprint", reprintResp, reprintErr, true)
	r.record("Fin Press", "reprint PDF created", r.pdfCount() == before+1, fmt.Sprintf("delta=%d", r.pdfCount()-before))
	r.record(
		"Fin Press",
		"reprint keeps balance",
		r.balance(8, testFinComponent) == balanceBefore,
		fmt.Sprintf("before=%.2f after=%.2f", balanceBefore, r.balance(8, testFinComponent)),
	)

	resp, err = r.post("/api/lines/radiator/receive", map[string]any{"session_id": sessionID})
	r.recordAPI("Radiator kirish", "receive FP session", resp, err, true)
	resp, err = r.post("/api/lines/radiator/receive", map[string]any{"session_id": sessionID})
	r.record("Radiator kirish", "duplicate receive rejected", err == nil && !apiOK(resp), responseDetail(resp, err))

	before = r.pdfCount()
	resp, err = r.post("/api/lines/radiator/v2/print", map[string]any{
		"radiator_component_id": testRadConfigID,
		"printer_v2_id":         radiatorPrinterID,
		"copy":                  1,
	})
	r.recordAPI("Radiator chiqish", "print", resp, err, true)
	radSessionID := nestedInt64(resp, "data", "session_id")
	r.record("Radiator chiqish", "PDF created", r.pdfCount() == before+1, fmt.Sprintf("session=%d delta=%d", radSessionID, r.pdfCount()-before))
	if radSessionID > 0 {
		balanceBefore = r.balance(9, testRadComponent)
		before = r.pdfCount()
		resp, err = r.post("/api/lines/radiator/v2/reprint", map[string]any{
			"session_id": radSessionID, "printer_v2_id": radiatorPrinterID, "copy": 1,
		})
		r.recordAPI("Radiator chiqish", "reprint", resp, err, true)
		r.record("Radiator chiqish", "reprint PDF created", r.pdfCount() == before+1, fmt.Sprintf("delta=%d", r.pdfCount()-before))
		r.record(
			"Radiator chiqish",
			"reprint keeps balance",
			r.balance(9, testRadComponent) == balanceBefore,
			fmt.Sprintf("before=%.2f after=%.2f", balanceBefore, r.balance(9, testRadComponent)),
		)
	}
}

func (r *runner) testGSCodeConcurrency() {
	serials := []string{r.runID + "_RACE_A", r.runID + "_RACE_B"}
	productIDs := make([]int, 2)
	for i, serial := range serials {
		id, err := r.repo.LinesAddProduct(7, 1, r.userID, testModelID, serial, "")
		if err != nil {
			r.record("GSCODE", "parallel fixture", false, err.Error())
			return
		}
		productIDs[i] = id
	}

	gsIDs := make([]int, 2)
	errs := make([]error, 2)
	var wg sync.WaitGroup
	for i := range productIDs {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			gsIDs[index], errs[index] = r.repo.GsCodeUpdate(testModelID, productIDs[index], r.userID)
		}(i)
	}
	wg.Wait()
	ok := errs[0] == nil && errs[1] == nil && gsIDs[0] > 0 && gsIDs[1] > 0 && gsIDs[0] != gsIDs[1]
	r.record("GSCODE", "parallel claims are distinct", ok, fmt.Sprintf("product_ids=%v gscode_ids=%v errors=%v", productIDs, gsIDs, errs))
}

func (r *runner) latestI1Products() ([]string, []int, error) {
	rows, err := r.db.Query(`
		SELECT p.serial, g.id
		FROM lines.products p
		JOIN production.gscodes g ON g.product_id = p.id
		WHERE p.line_id = 7 AND p.model_id = $1 AND p.time >= $2
		  AND p.serial NOT LIKE $3
		ORDER BY p.id DESC
		LIMIT 2`, testModelID, r.startedAt, r.runID+"%")
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	var serials []string
	var codes []int
	for rows.Next() {
		var serial string
		var code int
		if err := rows.Scan(&serial, &code); err != nil {
			return nil, nil, err
		}
		serials = append(serials, serial)
		codes = append(codes, code)
	}
	return serials, codes, rows.Err()
}

func (r *runner) modelAccSerial() string {
	var value string
	_ = r.db.QueryRow(`SELECT COALESCE(door_code, '') FROM production.models WHERE id = $1`, testModelID).Scan(&value)
	return value
}

func (r *runner) uniqueAccSerial() string {
	prefix := r.modelAccSerial()
	if prefix == "" {
		return r.runID + "_ACC"
	}
	return prefix + "-" + r.runID
}

func (r *runner) productStatuses(serial string) (map[int]string, error) {
	rows, err := r.db.Query(`SELECT line_id, status FROM lines.products WHERE serial = $1 ORDER BY line_id`, serial)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := map[int]string{}
	for rows.Next() {
		var lineID int
		var status string
		if err := rows.Scan(&lineID, &status); err != nil {
			return nil, err
		}
		result[lineID] = status
	}
	return result, rows.Err()
}

func (r *runner) balance(lineID, componentID int) float64 {
	var value float64
	_ = r.db.QueryRow(`
		SELECT COALESCE(quantity, 0)
		FROM lines.balance
		WHERE line_id = $1 AND component_id = $2`, lineID, componentID).Scan(&value)
	return value
}

func (r *runner) pdfCount() int {
	files, _ := filepath.Glob(filepath.Join(r.outputDir, "*.pdf"))
	return len(files)
}

func (r *runner) post(path string, payload any) (map[string]any, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequest(http.MethodPost, r.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", r.token)
	resp, err := r.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var decoded map[string]any
	if err := json.Unmarshal(data, &decoded); err != nil {
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(data)))
	}
	return decoded, nil
}

func (r *runner) recordAPI(area, name string, resp map[string]any, err error, expectOK bool) {
	ok := err == nil && apiOK(resp)
	if !expectOK {
		ok = err == nil && !apiOK(resp)
	}
	r.record(area, name, ok, responseDetail(resp, err))
}

func (r *runner) record(area, name string, ok bool, details string) {
	status := "FAIL"
	if ok {
		status = "PASS"
	}
	r.checks = append(r.checks, checkResult{
		Area: area, Name: name, Status: status, Details: details,
	})
}

func (r *runner) writeResults() error {
	payload := map[string]any{
		"run_id":     r.runID,
		"started_at": r.startedAt.Format(time.RFC3339),
		"output_dir": r.outputDir,
		"checks":     r.checks,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(r.outputDir, "results.json"), data, 0644)
}

func apiOK(resp map[string]any) bool {
	return resp != nil && fmt.Sprint(resp["result"]) == "ok"
}

func responseDetail(resp map[string]any, err error) string {
	if err != nil {
		return err.Error()
	}
	data, marshalErr := json.Marshal(resp)
	if marshalErr != nil {
		return fmt.Sprint(resp)
	}
	const max = 600
	if len(data) > max {
		return string(data[:max]) + "..."
	}
	return string(data)
}

func nestedString(value map[string]any, keys ...string) string {
	var current any = value
	for _, key := range keys {
		m, ok := current.(map[string]any)
		if !ok {
			return ""
		}
		current = m[key]
	}
	return fmt.Sprint(current)
}

func nestedInt64(value map[string]any, keys ...string) int64 {
	raw := nestedString(value, keys...)
	number, _ := strconv.ParseFloat(raw, 64)
	return int64(number)
}

func waitForPort(address string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", address, 250*time.Millisecond)
		if err == nil {
			_ = conn.Close()
			return nil
		}
		time.Sleep(100 * time.Millisecond)
	}
	return errors.New("test API ishga tushmadi")
}
