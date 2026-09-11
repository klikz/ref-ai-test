package utils

import (
	"bytes"
	"testing"
)

func TestRawPrintWorkerFrameRoundTrip(t *testing.T) {
	payload := []byte("SIZE 72 mm,105 mm\r\nPRINT 1\r\n")
	var buf bytes.Buffer
	if err := writeRawPrintWorkerFrame(&buf, rawPrintWorkerHeader{
		ReqID:    77,
		Printer:  "AC-FILE-TSPL",
		Doc:      "SN123",
		Datatype: "RAW",
	}, payload); err != nil {
		t.Fatalf("write: %v", err)
	}
	header, gotPayload, err := readRawPrintWorkerFrame(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if header.Printer != "AC-FILE-TSPL" || header.Doc != "SN123" || header.Datatype != "RAW" {
		t.Fatalf("header mismatch: %+v", header)
	}
	if header.ReqID != 77 {
		t.Fatalf("req_id=%d want 77", header.ReqID)
	}
	if header.PayloadLen != len(payload) {
		t.Fatalf("payload_len=%d want %d", header.PayloadLen, len(payload))
	}
	if !bytes.Equal(gotPayload, payload) {
		t.Fatalf("payload mismatch")
	}
}

func TestRawPrintWorkerFrameExit(t *testing.T) {
	var buf bytes.Buffer
	if err := writeRawPrintWorkerFrame(&buf, rawPrintWorkerHeader{Cmd: "exit"}, nil); err != nil {
		t.Fatalf("write: %v", err)
	}
	header, payload, err := readRawPrintWorkerFrame(&buf)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if header.Cmd != "exit" {
		t.Fatalf("cmd=%q", header.Cmd)
	}
	if len(payload) != 0 {
		t.Fatalf("exit payload should be empty")
	}
}

func TestParseRawWorkerReply(t *testing.T) {
	tests := []struct {
		line      string
		wantValid bool
		wantOK    bool
		wantReq   uint64
		wantJob   uint32
		wantMsg   string
	}{
		{line: "OK 12 3456", wantValid: true, wantOK: true, wantReq: 12, wantJob: 3456},
		{line: "OK 12 0", wantValid: true, wantOK: true, wantReq: 12, wantJob: 0},
		{line: "ERR 12 StartDocPrinter: access denied", wantValid: true, wantReq: 12, wantMsg: "StartDocPrinter: access denied"},
		{line: "  OK 7 9  ", wantValid: true, wantOK: true, wantReq: 7, wantJob: 9},
		{line: "READY", wantValid: false},
		{line: "", wantValid: false},
		{line: "garbage line", wantValid: false},
	}

	for _, tt := range tests {
		reply, valid := parseRawWorkerReply(tt.line)
		if valid != tt.wantValid {
			t.Errorf("parseRawWorkerReply(%q) valid=%v, kutilgan %v", tt.line, valid, tt.wantValid)
			continue
		}
		if !valid {
			continue
		}
		if reply.ok != tt.wantOK || reply.reqID != tt.wantReq || reply.jobID != tt.wantJob {
			t.Errorf("parseRawWorkerReply(%q) = %+v, kutilgan ok=%v req=%d job=%d",
				tt.line, reply, tt.wantOK, tt.wantReq, tt.wantJob)
		}
		if tt.wantMsg != "" && reply.msg != tt.wantMsg {
			t.Errorf("parseRawWorkerReply(%q) msg=%q, kutilgan %q", tt.line, reply.msg, tt.wantMsg)
		}
	}
}

// An error reply must never come back with an empty message.
func TestParseRawWorkerReplyErrorAlwaysHasMessage(t *testing.T) {
	reply, valid := parseRawWorkerReply("ERR 5")
	if !valid || reply.ok || reply.msg == "" {
		t.Fatalf("ERR javobida xabar bo'lishi kerak, olindi %+v (valid=%v)", reply, valid)
	}
}
