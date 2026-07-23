package utils

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
)

// PrintPool limits concurrent BarTender HTTP calls so print requests
// queue instead of spawning unbounded goroutines on Gin workers.
type PrintPool struct {
	slots chan struct{}
}

var (
	globalPrintPool *PrintPool
	printPoolOnce   sync.Once
)

func InitPrintPool(maxWorkers int) {
	printPoolOnce.Do(func() {
		if maxWorkers <= 0 {
			maxWorkers = 8
		}
		globalPrintPool = &PrintPool{
			slots: make(chan struct{}, maxWorkers),
		}
	})
}

func getPrintPool() *PrintPool {
	if globalPrintPool == nil {
		InitPrintPool(8)
	}
	return globalPrintPool
}

func (p *PrintPool) acquire() {
	p.slots <- struct{}{}
}

func (p *PrintPool) release() {
	<-p.slots
}

func executePrintHTTP(jsonStr []byte, printerURL string) string {
	count := 0
	for {
		if count > 3 {
			return "qaytadan urinib ko'ring"
		}

		req, err := http.NewRequest("POST", printerURL, bytes.NewBuffer(jsonStr))
		if err != nil {
			return err.Error()
		}
		req.Header.Set("X-Custom-Header", "myvalue")
		req.Header.Set("Content-Type", "application/json")

		resp, err := printerHTTPClient.Do(req)
		if err != nil {
			return "printer bilan aloqa yo'q"
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		var jsonMap map[string]any
		_ = json.Unmarshal(body, &jsonMap)

		if boolSuccess, ok := jsonMap["success"].(bool); ok && boolSuccess {
			return "ok"
		}

		count++
		if count >= 3 {
			fmt.Println("Print error:", string(body))
		}
	}
}

// PrintLabelAndWait sends a label to BarTender through the shared print pool.
// Returns "ok" on success or an error message string.
func (u *UtilsStruct) PrintLabelAndWait(jsonStr []byte, printerURL string) string {
	pool := getPrintPool()
	pool.acquire()
	defer pool.release()

	result := executePrintHTTP(jsonStr, printerURL)
	if result == "ok" {
		u.DebugLogAny("print ok", "")
	} else {
		u.DebugLogAny("print error: ", result)
	}
	return result
}
