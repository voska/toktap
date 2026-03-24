package proxy

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/voska/toktap/internal/sse"
)

var ssePayload = buildSSEPayload(100) // 100 SSE events

func buildSSEPayload(events int) string {
	var b strings.Builder
	b.WriteString("event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-opus-4-6\",\"usage\":{\"input_tokens\":25}}}\n\n")
	for i := range events {
		fmt.Fprintf(&b, "event: content_block_delta\ndata: {\"type\":\"content_block_delta\",\"delta\":{\"text\":\"word%d \"}}\n\n", i)
	}
	b.WriteString("event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":100}}\n\n")
	b.WriteString("event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n")
	return b.String()
}

func BenchmarkProxyStreaming(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		io.WriteString(w, ssePayload)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	routes := map[string]*Route{
		"anthropic": {Provider: "anthropic", Upstream: u},
	}
	p := New(routes, &noopWriter{}, nil)

	body := strings.NewReader(`{"stream":true}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		body.Reset(`{"stream":true}`)
		req := httptest.NewRequest("POST", "/anthropic/v1/messages", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
		io.Copy(io.Discard, rec.Result().Body)
	}
}

func BenchmarkProxyNonStreaming(b *testing.B) {
	respBody := `{"id":"msg_1","model":"claude-opus-4-6","content":[{"text":"hello"}],"usage":{"input_tokens":50,"output_tokens":20}}`
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(200)
		io.WriteString(w, respBody)
	}))
	defer upstream.Close()

	u, _ := url.Parse(upstream.URL)
	routes := map[string]*Route{
		"anthropic": {Provider: "anthropic", Upstream: u},
	}
	p := New(routes, &noopWriter{}, nil)

	body := strings.NewReader(`{}`)

	b.ResetTimer()
	b.ReportAllocs()

	for range b.N {
		body.Reset(`{}`)
		req := httptest.NewRequest("POST", "/anthropic/v1/messages", body)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		p.ServeHTTP(rec, req)
	}
}

func BenchmarkDirectVsProxy(b *testing.B) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		io.WriteString(w, ssePayload)
	}))
	defer upstream.Close()

	b.Run("direct", func(b *testing.B) {
		client := upstream.Client()
		b.ResetTimer()
		for range b.N {
			resp, _ := client.Post(upstream.URL, "application/json", strings.NewReader(`{}`))
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})

	b.Run("proxied", func(b *testing.B) {
		u, _ := url.Parse(upstream.URL)
		routes := map[string]*Route{
			"test": {Provider: "anthropic", Upstream: u},
		}
		p := New(routes, &noopWriter{}, nil)
		proxyServer := httptest.NewServer(p)
		defer proxyServer.Close()

		client := proxyServer.Client()
		b.ResetTimer()
		for range b.N {
			resp, _ := client.Post(proxyServer.URL+"/test/v1/messages", "application/json", strings.NewReader(`{}`))
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

func BenchmarkSSEPayloadSizes(b *testing.B) {
	for _, events := range []int{10, 100, 1000} {
		payload := buildSSEPayload(events)
		b.Run(fmt.Sprintf("events_%d", events), func(b *testing.B) {
			upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "text/event-stream")
				w.WriteHeader(200)
				io.WriteString(w, payload)
			}))
			defer upstream.Close()

			u, _ := url.Parse(upstream.URL)
			routes := map[string]*Route{
				"test": {Provider: "anthropic", Upstream: u},
			}
			p := New(routes, &noopWriter{}, nil)

			b.ResetTimer()
			b.ReportAllocs()

			for range b.N {
				req := httptest.NewRequest("POST", "/test/v1/messages", strings.NewReader(`{"stream":true}`))
				req.Header.Set("Content-Type", "application/json")
				rec := httptest.NewRecorder()
				p.ServeHTTP(rec, req)
				io.Copy(io.Discard, rec.Result().Body)
			}
		})
	}
}

type noopWriter struct{}

func (n *noopWriter) WriteUsage(_ sse.UsageData, _ Metadata, _ float64, _ int64, _ int, _ time.Time, _ string) {
}
