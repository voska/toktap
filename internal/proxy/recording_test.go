package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/voska/toktap/internal/recorder"
	"github.com/voska/toktap/internal/sse"
)

type nopWriter struct{}

func (nopWriter) WriteUsage(_ sse.UsageData, _ Metadata, _ float64, _ int64, _ int, _ time.Time, _ string) {
}

func TestRecordingSSECapture(t *testing.T) {
	// Mock upstream that returns an Anthropic-style SSE stream
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		flusher, _ := w.(http.Flusher)

		events := []string{
			"event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"id\":\"msg_test\",\"model\":\"claude-sonnet-4-20250514\",\"usage\":{\"input_tokens\":25,\"cache_creation_input_tokens\":0,\"cache_read_input_tokens\":0}}}\n\n",
			"event: content_block_start\ndata: {\"type\":\"content_block_start\",\"index\":0,\"content_block\":{\"type\":\"text\",\"text\":\"\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\"Hello\"}}\n\n",
			"event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"index\":0,\"delta\":{\"type\":\"text_delta\",\"text\":\" world\"}}\n\n",
			"event: content_block_stop\ndata: {\"type\":\"content_block_stop\",\"index\":0}\n\n",
			"event: message_delta\ndata: {\"type\":\"message_delta\",\"delta\":{\"stop_reason\":\"end_turn\"},\"usage\":{\"output_tokens\":10}}\n\n",
			"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n",
		}
		for _, ev := range events {
			_, _ = w.Write([]byte(ev))
			flusher.Flush()
		}
	}))
	defer upstream.Close()

	// Set up recorder
	dir := t.TempDir()
	rec, err := recorder.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()

	// Build proxy with single route pointing at mock upstream
	routes := map[string]*Route{
		"anthropic": {
			Upstream: mustParseURL(upstream.URL),
			Provider: "anthropic",
		},
	}
	p := New(routes, nopWriter{}, nil)
	p.SetRecorder(rec)

	// Make request
	body := `{"model":"claude-sonnet-4-20250514","max_tokens":100,"stream":true,"messages":[{"role":"user","content":"Say hello"}]}`
	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Device", "test")
	req.ContentLength = int64(len(body))

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	// Read response to trigger Close() on the TapReader
	resp := w.Result()
	respBody, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Fatalf("status = %d, want 200; body = %s", resp.StatusCode, string(respBody))
	}

	// Close recorder to flush
	rec.Close()

	// Check the JSONL file
	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(files) == 0 {
		t.Fatal("no recording files created")
	}

	data, err := os.ReadFile(files[0])
	if err != nil {
		t.Fatal(err)
	}

	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	if len(lines) != 1 {
		t.Fatalf("expected 1 record, got %d", len(lines))
	}

	var record recorder.Record
	if err := json.Unmarshal([]byte(lines[0]), &record); err != nil {
		t.Fatalf("unmarshal record: %v", err)
	}

	if record.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", record.Provider)
	}
	if record.Model != "claude-sonnet-4-20250514" {
		t.Errorf("model = %q, want claude-sonnet-4-20250514", record.Model)
	}
	if !record.IsStreaming {
		t.Error("is_streaming = false, want true")
	}
	if record.InputTokens != 25 {
		t.Errorf("input_tokens = %d, want 25", record.InputTokens)
	}
	if record.OutputTokens != 10 {
		t.Errorf("output_tokens = %d, want 10", record.OutputTokens)
	}
	if len(record.SSEEvents) == 0 {
		t.Error("sse_events is empty, want captured events")
	}
	if record.SSEEvents[0].Type != "message_start" {
		t.Errorf("first SSE event type = %q, want message_start", record.SSEEvents[0].Type)
	}

	// Check request body was captured
	if record.RequestBody == nil {
		t.Error("request_body is nil")
	} else {
		var reqBody map[string]interface{}
		if err := json.Unmarshal(record.RequestBody, &reqBody); err != nil {
			t.Errorf("request_body not valid JSON: %v", err)
		}
		if reqBody["model"] != "claude-sonnet-4-20250514" {
			t.Errorf("request_body model = %v, want claude-sonnet-4-20250514", reqBody["model"])
		}
	}

	t.Logf("Record captured: model=%s, input=%d, output=%d, sse_events=%d, streaming=%v",
		record.Model, record.InputTokens, record.OutputTokens, len(record.SSEEvents), record.IsStreaming)
}

func TestRecordingNonStreamingCapture(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		_, _ = w.Write([]byte(`{"id":"msg_test","model":"claude-sonnet-4-20250514","content":[{"type":"text","text":"Hello"}],"usage":{"input_tokens":15,"output_tokens":5}}`))
	}))
	defer upstream.Close()

	dir := t.TempDir()
	rec, err := recorder.New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()

	routes := map[string]*Route{
		"anthropic": {Upstream: mustParseURL(upstream.URL), Provider: "anthropic"},
	}
	p := New(routes, nopWriter{}, nil)
	p.SetRecorder(rec)

	body := `{"model":"claude-sonnet-4-20250514","max_tokens":100,"messages":[{"role":"user","content":"Hi"}]}`
	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.ContentLength = int64(len(body))

	w := httptest.NewRecorder()
	p.ServeHTTP(w, req)

	resp := w.Result()
	io.ReadAll(resp.Body)
	resp.Body.Close()
	rec.Close()

	files, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	if len(files) == 0 {
		t.Fatal("no recording files created")
	}

	data, _ := os.ReadFile(files[0])
	var record recorder.Record
	json.Unmarshal([]byte(strings.TrimSpace(string(data))), &record)

	if record.IsStreaming {
		t.Error("is_streaming = true, want false")
	}
	if record.ResponseBody == nil {
		t.Error("response_body is nil for non-streaming")
	}
	if record.InputTokens != 15 {
		t.Errorf("input_tokens = %d, want 15", record.InputTokens)
	}
	t.Logf("Non-streaming record: model=%s, input=%d, output=%d, has_response_body=%v",
		record.Model, record.InputTokens, record.OutputTokens, record.ResponseBody != nil)
}

func mustParseURL(s string) *url.URL {
	u, err := url.Parse(s)
	if err != nil {
		panic(err)
	}
	return u
}
