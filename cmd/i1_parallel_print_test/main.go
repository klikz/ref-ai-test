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
	"sort"
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
	testPort    = "18082"
	parallelN   = 50
	testModelID = 107
	i1PrinterID = 13
	i1LineID    = 7
)

type printOutcome struct {
	Index   int
	Serial  string
	GSCode  string
	Error   string
	APIOK   bool
}

type serialRow struct {
	Serial string
	GSCode string
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
		"i1-parallel-"+now.Format("20060102-150405"),
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

	var freeGS int
	if err := db.QueryRow(`
		SELECT COUNT(*) FROM production.gscodes
		WHERE model_id = $1 AND status = true`, testModelID).Scan(&freeGS); err != nil {
		panic(err)
	}
	if freeGS < parallelN {
		panic(fmt.Sprintf("yetarli GSCODE yo'q: kerak %d, mavjud %d", parallelN, freeGS))
	}

	appStore := store.New(db)
	runID := "ZZPAR_" + now.Format("20060102_150405")
	userID, token, err := prepareAuth(db, runID)
	if err != nil {
		panic(err)
	}

	logger := zerolog.Nop()
	go func() {
		if err := api.StartSrv(testPort, "debug", &logger, *appStore); err != nil {
			fmt.Fprintln(os.Stderr, "test API:", err)
		}
	}()
	if err := waitForPort("127.0.0.1:"+testPort, 30*time.Second); err != nil {
		panic(err)
	}

	if err := ensurePlanLocked("http://127.0.0.1:"+testPort, token); err != nil {
		panic(err)
	}

	startedAt := time.Now()
	outcomes := runParallelPrints(
		"http://127.0.0.1:"+testPort,
		token,
		parallelN,
	)

	dbRows, err := loadDBRows(db, startedAt)
	if err != nil {
		panic(err)
	}

	report := verify(outcomes, dbRows, outputDir)
	report["run_id"] = runID
	report["user_id"] = userID
	report["output_dir"] = outputDir
	report["started_at"] = startedAt.Format(time.RFC3339)
	report["parallel_n"] = parallelN

	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		panic(err)
	}
	resultsPath := filepath.Join(outputDir, "results.json")
	if err := os.WriteFile(resultsPath, data, 0644); err != nil {
		panic(err)
	}

	summary, _ := report["summary"].(map[string]any)
	fmt.Printf("RESULTS_FILE=%s\n", resultsPath)
	fmt.Printf("PDF_DIR=%s\n", outputDir)
	fmt.Printf("SUMMARY=%v\n", summary)
	if fails, ok := summary["fail"].(int); ok && fails > 0 {
		os.Exit(2)
	}
}

func prepareAuth(db *sql.DB, runID string) (int, string, error) {
	var userID int
	if err := db.QueryRow(`SELECT id FROM auth.users WHERE status = true ORDER BY id LIMIT 1`).Scan(&userID); err != nil {
		return 0, "", err
	}
	routes := []string{
		"/api/production/plan/save",
		"/api/production/plan/unlock-day",
		"/api/production/plan/lock-day",
		"/api/lines/ichki/v2/print",
	}
	for _, route := range routes {
		if _, err := db.Exec(`
			INSERT INTO auth.routes (route)
			SELECT $1::text
			WHERE NOT EXISTS (SELECT 1 FROM auth.routes WHERE route = $1::text)`, route); err != nil {
			return 0, "", fmt.Errorf("route %s: %w", route, err)
		}
		if _, err := db.Exec(`
			INSERT INTO auth.permissions (user_id, route_id)
			SELECT $1, id FROM auth.routes ar
			WHERE ar.route = $2
			  AND NOT EXISTS (
				SELECT 1 FROM auth.permissions p
				WHERE p.user_id = $1 AND p.route_id = ar.id
			  )`, userID, route); err != nil {
			return 0, "", fmt.Errorf("permission %s: %w", route, err)
		}
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"login":   runID,
		"user_id": userID,
		"nbf":     time.Date(2015, 10, 10, 12, 0, 0, 0, time.UTC).Unix(),
	})
	signed, err := token.SignedString([]byte(os.Getenv("SECRET_KEY")))
	return userID, signed, err
}

func ensurePlanLocked(baseURL, token string) error {
	today := time.Now().Format("2006-01-02")
	client := &http.Client{Timeout: 30 * time.Second}
	post := func(path string, payload map[string]any) (map[string]any, error) {
		body, err := json.Marshal(payload)
		if err != nil {
			return nil, err
		}
		req, err := http.NewRequest(http.MethodPost, baseURL+path, bytes.NewReader(body))
		if err != nil {
			return nil, err
		}
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", token)
		resp, err := client.Do(req)
		if err != nil {
			return nil, err
		}
		defer resp.Body.Close()
		raw, err := io.ReadAll(resp.Body)
		if err != nil {
			return nil, err
		}
		var decoded map[string]any
		if err := json.Unmarshal(raw, &decoded); err != nil {
			return nil, err
		}
		return decoded, nil
	}
	_, _ = post("/api/production/plan/unlock-day", map[string]any{
		"line_id": i1LineID, "plan_date": today,
	})
	saveResp, err := post("/api/production/plan/save", map[string]any{
		"line_id": i1LineID,
		"cells": []map[string]any{{
			"plan_date": today, "model_id": testModelID, "component_id": 0,
			"planned_qty": 100000, "allow_overplan": true,
		}},
	})
	if err != nil {
		return fmt.Errorf("plan save: %w", err)
	}
	if fmt.Sprint(saveResp["result"]) != "ok" {
		return fmt.Errorf("plan save: %s", saveResp["error"])
	}
	lockResp, err := post("/api/production/plan/lock-day", map[string]any{
		"line_id": i1LineID, "plan_date": today,
	})
	if err != nil {
		return err
	}
	if fmt.Sprint(lockResp["result"]) != "ok" {
		return fmt.Errorf("plan lock: %s", lockResp["error"])
	}
	return nil
}

func runParallelPrints(baseURL, token string, n int) []printOutcome {
	client := &http.Client{Timeout: 300 * time.Second}
	outcomes := make([]printOutcome, n)
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(index int) {
			defer wg.Done()
			outcome := printOutcome{Index: index}
			body, _ := json.Marshal(map[string]any{
				"model_id":      testModelID,
				"printer_v2_id": i1PrinterID,
				"quantity":      1,
			})
			req, err := http.NewRequest(http.MethodPost, baseURL+"/api/lines/ichki/v2/print", bytes.NewReader(body))
			if err != nil {
				outcome.Error = err.Error()
				outcomes[index] = outcome
				return
			}
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", token)
			resp, err := client.Do(req)
			if err != nil {
				outcome.Error = err.Error()
				outcomes[index] = outcome
				return
			}
			defer resp.Body.Close()
			raw, err := io.ReadAll(resp.Body)
			if err != nil {
				outcome.Error = err.Error()
				outcomes[index] = outcome
				return
			}
			var decoded map[string]any
			if err := json.Unmarshal(raw, &decoded); err != nil {
				outcome.Error = string(raw)
				outcomes[index] = outcome
				return
			}
			if fmt.Sprint(decoded["result"]) != "ok" {
				outcome.Error = fmt.Sprint(decoded["error"])
				outcomes[index] = outcome
				return
			}
			data, _ := decoded["data"].(map[string]any)
			outcome.APIOK = true
			outcome.Serial = strings.TrimSpace(fmt.Sprint(data["serial"]))
			outcome.GSCode = strings.TrimSpace(fmt.Sprint(data["gscode"]))
			outcomes[index] = outcome
		}(i)
	}
	wg.Wait()
	return outcomes
}

func loadDBRows(db *sql.DB, since time.Time) ([]serialRow, error) {
	rows, err := db.Query(`
		SELECT p.serial, g.data
		FROM lines.products p
		JOIN production.gscodes g ON g.product_id = p.id
		WHERE p.line_id = $1
		  AND p.model_id = $2
		  AND p.time >= $3
		ORDER BY p.id`, i1LineID, testModelID, since)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []serialRow
	for rows.Next() {
		var row serialRow
		if err := rows.Scan(&row.Serial, &row.GSCode); err != nil {
			return nil, err
		}
		result = append(result, row)
	}
	return result, rows.Err()
}

func verify(outcomes []printOutcome, dbRows []serialRow, outputDir string) map[string]any {
	checks := []map[string]any{}
	pass, fail := 0, 0
	record := func(name string, ok bool, details string) {
		status := "PASS"
		if !ok {
			status = "FAIL"
			fail++
		} else {
			pass++
		}
		checks = append(checks, map[string]any{
			"name": name, "status": status, "details": details,
		})
	}

	apiOK := 0
	printBySerial := map[string]string{}
	printGSCodes := []string{}
	var apiErrors []string
	for _, o := range outcomes {
		if o.APIOK {
			apiOK++
			printBySerial[o.Serial] = o.GSCode
			printGSCodes = append(printGSCodes, o.GSCode)
		} else if o.Error != "" {
			apiErrors = append(apiErrors, fmt.Sprintf("#%d: %s", o.Index, o.Error))
		}
	}
	record("50 parallel API success", apiOK == parallelN, fmt.Sprintf("ok=%d errors=%v", apiOK, apiErrors))
	record("50 DB rows with GSCODE", len(dbRows) == parallelN, fmt.Sprintf("rows=%d", len(dbRows)))

	onlyDB := []string{}
	for _, row := range dbRows {
		if _, ok := printBySerial[row.Serial]; !ok {
			onlyDB = append(onlyDB, row.Serial)
		}
	}
	sort.Strings(onlyDB)
	record("API timeout bo'lsa ham DB da GSCODE", len(onlyDB)+apiOK == parallelN, fmt.Sprintf("api_ok=%d db_only=%v", apiOK, onlyDB))

	dbBySerial := map[string]string{}
	dbGSCodes := make([]string, 0, len(dbRows))
	for _, row := range dbRows {
		dbBySerial[row.Serial] = row.GSCode
		dbGSCodes = append(dbGSCodes, row.GSCode)
	}

	mismatches := []string{}
	for serial, printGS := range printBySerial {
		dbGS, ok := dbBySerial[serial]
		if !ok {
			mismatches = append(mismatches, serial+": DB da yo'q")
			continue
		}
		if printGS != dbGS {
			mismatches = append(mismatches, fmt.Sprintf("%s: print=%s db=%s", serial, truncate(printGS, 40), truncate(dbGS, 40)))
		}
	}
	sort.Strings(mismatches)
	record("print GSCODE == DB GSCODE (har serial)", len(mismatches) == 0, strings.Join(mismatches, "; "))

	printUnique := uniqueCount(printGSCodes)
	dbUnique := uniqueCount(dbGSCodes)
	record("print GSCODE lar xil", printUnique == parallelN, fmt.Sprintf("unique=%d total=%d", printUnique, len(printGSCodes)))
	record("DB GSCODE lar xil", dbUnique == parallelN, fmt.Sprintf("unique=%d total=%d", dbUnique, len(dbGSCodes)))

	dupPrint := duplicateValues(printGSCodes)
	dupDB := duplicateValues(dbGSCodes)
	record("print duplicate GSCODE yo'q", len(dupPrint) == 0, strings.Join(dupPrint, ", "))
	record("DB duplicate GSCODE yo'q", len(dupDB) == 0, strings.Join(dupDB, ", "))

	pdfFiles, _ := filepath.Glob(filepath.Join(outputDir, "*.pdf"))
	pdfBySerial := map[string]int{}
	for _, path := range pdfFiles {
		name := filepath.Base(path)
		for serial := range printBySerial {
			if strings.HasPrefix(name, serial+"-") {
				pdfBySerial[serial]++
			}
		}
	}
	missingPDF := []string{}
	for serial := range printBySerial {
		if pdfBySerial[serial] == 0 {
			missingPDF = append(missingPDF, serial)
		}
	}
	sort.Strings(missingPDF)
	record("har serial uchun PDF mavjud", len(missingPDF) == 0, fmt.Sprintf("pdfs=%d missing=%v", len(pdfFiles), missingPDF))

	perSerial := make([]map[string]string, 0, len(printBySerial))
	serials := make([]string, 0, len(printBySerial))
	for s := range printBySerial {
		serials = append(serials, s)
	}
	sort.Strings(serials)
	for _, serial := range serials {
		perSerial = append(perSerial, map[string]string{
			"serial":      serial,
			"print_gscode": truncate(printBySerial[serial], 80),
			"db_gscode":    truncate(dbBySerial[serial], 80),
			"match":        fmt.Sprint(printBySerial[serial] == dbBySerial[serial]),
			"pdf_count":    fmt.Sprint(pdfBySerial[serial]),
		})
	}

	return map[string]any{
		"summary": map[string]any{"pass": pass, "fail": fail},
		"checks":  checks,
		"serials": perSerial,
	}
}

func uniqueCount(values []string) int {
	seen := map[string]struct{}{}
	for _, v := range values {
		if v == "" {
			continue
		}
		seen[v] = struct{}{}
	}
	return len(seen)
}

func duplicateValues(values []string) []string {
	counts := map[string]int{}
	for _, v := range values {
		if v != "" {
			counts[v]++
		}
	}
	var dups []string
	for v, c := range counts {
		if c > 1 {
			dups = append(dups, fmt.Sprintf("%s(x%d)", truncate(v, 30), c))
		}
	}
	sort.Strings(dups)
	return dups
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
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
