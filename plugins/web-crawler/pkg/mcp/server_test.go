package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"testing"
)

func TestMCPServer(t *testing.T) {
	inBuf := bytes.NewBuffer(nil)
	outBuf := bytes.NewBuffer(nil)

	// Send initialize request
	initReq := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{}}` + "\n"
	inBuf.WriteString(initReq)

	// Send tools/list request
	listReq := `{"jsonrpc":"2.0","id":2,"method":"tools/list","params":{}}` + "\n"
	inBuf.WriteString(listReq)

	server := NewServer(inBuf, outBuf)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := server.Run(ctx); err != nil {
		t.Fatalf("server run error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(outBuf.String()), "\n")
	if len(lines) < 2 {
		t.Fatalf("expected at least 2 response lines, got %d", len(lines))
	}

	var initResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[0]), &initResp); err != nil {
		t.Fatalf("failed to unmarshal init response: %v", err)
	}

	if initResp.Error != nil {
		t.Errorf("init response returned error: %v", initResp.Error)
	}

	var listResp JSONRPCResponse
	if err := json.Unmarshal([]byte(lines[1]), &listResp); err != nil {
		t.Fatalf("failed to unmarshal list response: %v", err)
	}

	resMap, ok := listResp.Result.(map[string]interface{})
	if !ok {
		t.Fatalf("invalid result type: %T", listResp.Result)
	}

	tools, ok := resMap["tools"].([]interface{})
	if !ok || len(tools) != 3 {
		t.Errorf("expected 3 tools in list, got %v", tools)
	}
}
