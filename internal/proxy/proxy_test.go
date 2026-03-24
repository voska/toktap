package proxy

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/voska/toktap/internal/sse"
)

type mockWriter struct {
	calls []writeCall
}

type writeCall struct {
	usage      sse.UsageData
	meta       Metadata
	costUSD    float64
	statusCode int
}

func (m *mockWriter) WriteUsage(u sse.UsageData, meta Metadata, cost float64, respMs int64, status int, ts time.Time, requestID string) {
	m.calls = append(m.calls, writeCall{usage: u, meta: meta, costUSD: cost, statusCode: status})
}

func testRoute(upstream string, provider string) map[string]*Route {
	u, _ := url.Parse(upstream)
	return map[string]*Route{
		provider: {
			Provider: provider,
			Upstream: u,
		},
	}
}

func TestProxyAnthropicStreaming(t *testing.T) {
	sseBody := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-opus-4-6\",\"usage\":{\"input_tokens\":25}}}\n\n" +
		"event: content_block_delta\ndata: {\"delta\":{\"text\":\"hi\"}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":10}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/messages" {
			t.Errorf("upstream path = %q, want /v1/messages", r.URL.Path)
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte(sseBody))
	}))
	defer upstream.Close()

	writer := &mockWriter{}
	p := New(testRoute(upstream.URL, "anthropic"), writer, nil)

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(`{"stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if rec.Code != 200 {
		t.Errorf("status = %d, want 200", rec.Code)
	}
	if rec.Body.String() != sseBody {
		t.Error("response body modified by proxy")
	}

	time.Sleep(50 * time.Millisecond)

	if len(writer.calls) != 1 {
		t.Fatalf("got %d writes, want 1", len(writer.calls))
	}
	if writer.calls[0].usage.Model != "claude-opus-4-6" {
		t.Errorf("model = %q", writer.calls[0].usage.Model)
	}
	if writer.calls[0].usage.InputTokens != 25 {
		t.Errorf("input = %d", writer.calls[0].usage.InputTokens)
	}
	if writer.calls[0].meta.Provider != "anthropic" {
		t.Errorf("provider = %q", writer.calls[0].meta.Provider)
	}
}

func TestProxyOpenAIInjectsStreamOptions(t *testing.T) {
	var gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte("data: {\"model\":\"gpt-5.4\",\"choices\":[],\"usage\":{\"prompt_tokens\":10,\"completion_tokens\":5}}\n\ndata: [DONE]\n\n"))
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	routes := map[string]*Route{
		"openai": {
			Provider:            "openai",
			Upstream:            u,
			InjectStreamOptions: true,
		},
	}
	p := New(routes, &mockWriter{}, nil)

	req := httptest.NewRequest("POST", "/openai/v1/chat/completions", strings.NewReader(`{"model":"gpt-5.4","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if !strings.Contains(gotBody, "include_usage") {
		t.Error("stream_options not injected into request body")
	}
}

func TestProxyNonStreamingResponse(t *testing.T) {
	respBody := `{"id":"msg_1","model":"claude-opus-4-6","content":[{"text":"hi"}],"usage":{"input_tokens":50,"output_tokens":20}}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(respBody))
	}))
	defer upstream.Close()

	writer := &mockWriter{}
	p := New(testRoute(upstream.URL, "anthropic"), writer, nil)

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if rec.Body.String() != respBody {
		t.Error("non-streaming response body modified")
	}

	time.Sleep(50 * time.Millisecond)

	if len(writer.calls) != 1 {
		t.Fatalf("got %d writes, want 1", len(writer.calls))
	}
	if writer.calls[0].usage.InputTokens != 50 {
		t.Errorf("input = %d, want 50", writer.calls[0].usage.InputTokens)
	}
}

func TestProxyUnknownRouteReturns502(t *testing.T) {
	p := New(map[string]*Route{}, &mockWriter{}, nil)

	req := httptest.NewRequest("GET", "/unknown/path", nil)
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if rec.Code != 502 {
		t.Errorf("status = %d, want 502", rec.Code)
	}
}

func TestProxyChatGPTRouteMapsToOpenAIProvider(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"model":"gpt-5.3-codex","usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	routes := map[string]*Route{
		"chatgpt": {
			Provider: "openai",
			Upstream: u,
		},
	}

	writer := &mockWriter{}
	p := New(routes, writer, nil)

	req := httptest.NewRequest("POST", "/chatgpt/codex-responses", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	time.Sleep(50 * time.Millisecond)

	if len(writer.calls) != 1 {
		t.Fatalf("got %d writes, want 1", len(writer.calls))
	}
	if writer.calls[0].meta.Provider != "openai" {
		t.Errorf("provider = %q, want openai", writer.calls[0].meta.Provider)
	}
}

func TestProxyUpstreamPathPrepended(t *testing.T) {
	var gotPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"model":"gpt-5.3-codex","usage":{"prompt_tokens":10,"completion_tokens":5}}`))
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL + "/backend-api/codex")
	routes := map[string]*Route{
		"chatgpt": {Provider: "openai", Upstream: u},
	}
	p := New(routes, &mockWriter{}, nil)

	req := httptest.NewRequest("POST", "/chatgpt/responses", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if gotPath != "/backend-api/codex/responses" {
		t.Errorf("upstream path = %q, want /backend-api/codex/responses", gotPath)
	}
}

func TestProxySkipsStreamOptionsWhenNotConfigured(t *testing.T) {
	var gotBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		gotBody = string(body)
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		w.Write([]byte(`{"model":"claude-opus-4-6","usage":{"input_tokens":10,"output_tokens":5}}`))
	}))
	defer upstream.Close()

	p := New(testRoute(upstream.URL, "anthropic"), &mockWriter{}, nil)

	req := httptest.NewRequest("POST", "/anthropic/v1/messages", strings.NewReader(`{"model":"claude-opus-4-6","stream":true}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	p.ServeHTTP(rec, req)

	if strings.Contains(gotBody, "stream_options") {
		t.Error("stream_options should not be injected for anthropic route")
	}
}
