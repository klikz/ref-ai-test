package utils

import (
	"bufio"
	"encoding/binary"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"sync/atomic"
	"time"
)

// printRequestSeq numbers worker requests so replies can be matched to them.
var printRequestSeq atomic.Uint64

func nextPrintRequestID() uint64 { return printRequestSeq.Add(1) }

const printWorkerArgPrefix = "--print-worker="

// errRawWorkerUnavailable marks failures that happened before anything could
// reach the spooler, so resending cannot produce a duplicate label.
var errRawWorkerUnavailable = errors.New("RAW worker mavjud emas")

// Binary RAW worker frame (after READY):
//
//	u32be headerLen | headerJSON | payload bytes (header.payload_len)
//
// Exit: header {"cmd":"exit"} with payload_len 0.
//
// Replies are one line, "OK <reqID> <jobID>" or "ERR <reqID> <message>". The
// request id lets the client discard a reply left over from an abandoned
// request instead of restarting the worker over a single desynced line.
type rawPrintWorkerHeader struct {
	Cmd        string `json:"cmd,omitempty"`
	ReqID      uint64 `json:"req_id,omitempty"`
	Printer    string `json:"printer,omitempty"`
	Doc        string `json:"doc,omitempty"`
	Datatype   string `json:"datatype,omitempty"`
	PayloadLen int    `json:"payload_len,omitempty"`
}

type rawWorkerReply struct {
	ok    bool
	reqID uint64
	jobID uint32
	msg   string
}

// parseRawWorkerReply understands "OK <reqID> <jobID>" and
// "ERR <reqID> <message>". Malformed lines report valid=false so the caller can
// treat them as a protocol break.
func parseRawWorkerReply(line string) (rawWorkerReply, bool) {
	fields := strings.Fields(strings.TrimSpace(line))
	if len(fields) == 0 {
		return rawWorkerReply{}, false
	}

	var reply rawWorkerReply
	switch fields[0] {
	case "OK":
		reply.ok = true
	case "ERR":
	default:
		return rawWorkerReply{}, false
	}

	if len(fields) >= 2 {
		if id, err := strconv.ParseUint(fields[1], 10, 64); err == nil {
			reply.reqID = id
		} else if !reply.ok {
			reply.msg = strings.Join(fields[1:], " ")
			return reply, true
		}
	}
	if len(fields) >= 3 {
		if reply.ok {
			if id, err := strconv.ParseUint(fields[2], 10, 32); err == nil {
				reply.jobID = uint32(id)
			}
		} else {
			reply.msg = strings.Join(fields[2:], " ")
		}
	}
	if !reply.ok && reply.msg == "" {
		reply.msg = "noma'lum worker xatosi"
	}
	return reply, true
}

func isPrintWorkerProcess() bool {
	for _, a := range os.Args[1:] {
		if strings.HasPrefix(a, printWorkerArgPrefix) {
			return true
		}
	}
	return false
}

// MaybeRunPrintWorker enters TSPL/ZPL RAW worker mode when argv contains
// --print-worker=tspl|zpl. Returns true if this process should exit after the call.
func MaybeRunPrintWorker(args []string) bool {
	for _, a := range args[1:] {
		if !strings.HasPrefix(a, printWorkerArgPrefix) {
			continue
		}
		lang := NormalizePrintLanguage(strings.TrimPrefix(a, printWorkerArgPrefix))
		if lang != PrintLanguageTSPL && lang != PrintLanguageZPL {
			fmt.Fprintf(os.Stderr, "noma'lum print worker: %s\n", lang)
			os.Exit(2)
		}
		runRawPrintWorkerLoop(lang)
		return true
	}
	return false
}

func runRawPrintWorkerLoop(language string) {
	fmt.Println("READY")
	in := bufio.NewReaderSize(os.Stdin, 1024*1024)

	for {
		header, payload, err := readRawPrintWorkerFrame(in)
		if err != nil {
			if errors.Is(err, io.EOF) {
				return
			}
			fmt.Fprintf(os.Stderr, "print worker stdin: %v\n", err)
			os.Exit(1)
		}
		if strings.EqualFold(strings.TrimSpace(header.Cmd), "exit") {
			return
		}
		datatype := header.Datatype
		if strings.TrimSpace(datatype) == "" {
			datatype = RawSpoolDatatype(language)
		}
		jobID, err := rawPrintWindows(header.Printer, header.Doc, datatype, payload)
		if err != nil {
			msg := strings.ReplaceAll(err.Error(), "\n", " ")
			fmt.Printf("ERR %d %s\n", header.ReqID, msg)
			continue
		}
		fmt.Printf("OK %d %d\n", header.ReqID, jobID)
	}
}

func readRawPrintWorkerFrame(r io.Reader) (rawPrintWorkerHeader, []byte, error) {
	var headerLenBuf [4]byte
	if _, err := io.ReadFull(r, headerLenBuf[:]); err != nil {
		return rawPrintWorkerHeader{}, nil, err
	}
	headerLen := binary.BigEndian.Uint32(headerLenBuf[:])
	if headerLen == 0 || headerLen > 1024*1024 {
		return rawPrintWorkerHeader{}, nil, fmt.Errorf("invalid header length %d", headerLen)
	}
	headerBytes := make([]byte, headerLen)
	if _, err := io.ReadFull(r, headerBytes); err != nil {
		return rawPrintWorkerHeader{}, nil, err
	}
	var header rawPrintWorkerHeader
	if err := json.Unmarshal(headerBytes, &header); err != nil {
		return rawPrintWorkerHeader{}, nil, fmt.Errorf("header json: %w", err)
	}
	if strings.EqualFold(strings.TrimSpace(header.Cmd), "exit") {
		return header, nil, nil
	}
	if header.PayloadLen < 0 || header.PayloadLen > 32*1024*1024 {
		return rawPrintWorkerHeader{}, nil, fmt.Errorf("invalid payload length %d", header.PayloadLen)
	}
	payload := make([]byte, header.PayloadLen)
	if header.PayloadLen > 0 {
		if _, err := io.ReadFull(r, payload); err != nil {
			return rawPrintWorkerHeader{}, nil, err
		}
	}
	return header, payload, nil
}

func writeRawPrintWorkerFrame(w io.Writer, header rawPrintWorkerHeader, payload []byte) error {
	if len(payload) > 0 {
		header.PayloadLen = len(payload)
	}
	headerBytes, err := json.Marshal(header)
	if err != nil {
		return err
	}
	var lenBuf [4]byte
	binary.BigEndian.PutUint32(lenBuf[:], uint32(len(headerBytes)))
	if _, err := w.Write(lenBuf[:]); err != nil {
		return err
	}
	if _, err := w.Write(headerBytes); err != nil {
		return err
	}
	if len(payload) > 0 {
		if _, err := w.Write(payload); err != nil {
			return err
		}
	}
	return nil
}

// printWorkerCount is how many worker processes serve one print language.
//
// A single process per language meant every printer in the plant queued behind
// the same mutex, so one slow printer stalled every other line.
func printWorkerCount(language string) int {
	switch NormalizePrintLanguage(language) {
	case PrintLanguageTSPL:
		return clampWorkerCount(envIntDefault("PRINT_V2_TSPL_WORKERS", 4))
	case PrintLanguageZPL:
		return clampWorkerCount(envIntDefault("PRINT_V2_ZPL_WORKERS", 4))
	default:
		// Each GDI worker is a PowerShell process holding GDI+ resources, so
		// this stays deliberately small.
		return clampWorkerCount(envIntDefault("PRINT_V2_GDI_WORKERS", 2))
	}
}

func clampWorkerCount(n int) int {
	if n < 1 {
		return 1
	}
	if n > 16 {
		return 16
	}
	return n
}

// submitQueueTimeout bounds the wait for a free worker so an overloaded system
// returns a clear error instead of blocking the HTTP handler indefinitely.
func submitQueueTimeout() time.Duration {
	ms := envIntDefault("PRINT_V2_SUBMIT_QUEUE_TIMEOUT_MS", 15000)
	if ms < 1000 {
		ms = 1000
	}
	if ms > 120000 {
		ms = 120000
	}
	return time.Duration(ms) * time.Millisecond
}

func printWorkerReplyTimeout() time.Duration {
	ms := envIntDefault("PRINT_V2_WORKER_TIMEOUT_MS", 30000)
	if ms < 5000 {
		ms = 5000
	}
	if ms > 120000 {
		ms = 120000
	}
	return time.Duration(ms) * time.Millisecond
}

func readLineWithTimeout(r *bufio.Reader, timeout time.Duration) (string, error) {
	type result struct {
		line string
		err  error
	}
	ch := make(chan result, 1)
	go func() {
		line, err := r.ReadString('\n')
		ch <- result{line: line, err: err}
	}()
	select {
	case res := <-ch:
		return res.line, res.err
	case <-time.After(timeout):
		return "", errors.New("worker javob vaqti tugadi")
	}
}
