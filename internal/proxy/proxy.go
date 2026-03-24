package proxy

import (
	"bytes"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/http/httputil"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/voska/toktap/internal/pricing"
	"github.com/voska/toktap/internal/sse"
)

type UsageWriter interface {
	WriteUsage(sse.UsageData, Metadata, float64, int64, int, time.Time, string)
}

type Proxy struct {
	routes          map[string]*Route
	writer          UsageWriter
	pricing         *pricing.Table
	chromeTransport http.RoundTripper
}

func New(routes map[string]*Route, writer UsageWriter, pricingTable *pricing.Table) *Proxy {
	return &Proxy{
		routes:          routes,
		writer:          writer,
		pricing:         pricingTable,
		chromeTransport: NewChromeTransport(),
	}
}

// Headers to strip so they don't leak to upstream services.
// Cloudflare Tunnel headers cause RFC 8586 loop detection failures.
// Accept-Encoding must be stripped so SSE arrives uncompressed.
var stripHeaders = []string{
	"X-Device",
	"Cdn-Loop", "Cf-Ray", "Cf-Connecting-Ip", "Cf-Visitor",
	"Cf-Ipcountry", "Cf-Warp-Tag-Id",
	"X-Forwarded-For", "X-Forwarded-Host", "X-Forwarded-Port",
	"X-Forwarded-Proto", "X-Forwarded-Server", "X-Real-Ip",
	"Accept-Encoding",
}

func (p *Proxy) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path == "/healthz" {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
		return
	}

	routeName, routePath := splitRoutePath(r.URL.Path)
	route, ok := p.routes[routeName]
	if !ok {
		http.Error(w, "unknown route", http.StatusBadGateway)
		return
	}

	meta := ExtractMetadata(r, routeName, route.Provider)
	requestID := uuid.New().String()
	start := time.Now()
	requestMethod := r.Method
	requestHeaders := SanitizeHeaders(r.Header)

	// Build upstream path: strip route prefix, prepend upstream base path.
	r.URL.Path = routePath
	if route.Upstream.Path != "" {
		r.URL.Path = strings.TrimSuffix(route.Upstream.Path, "/") + r.URL.Path
	}

	var requestBodyPreview string
	if r.Method == "POST" && r.Body != nil && r.ContentLength != 0 {
		body, err := io.ReadAll(r.Body)
		_ = r.Body.Close()
		if err == nil {
			requestBodyPreview = TruncateBody(body)
			if route.InjectStreamOptions {
				if modified, changed, _ := InjectStreamOptions(body); changed {
					body = modified
				}
			}
			r.Body = io.NopCloser(bytes.NewReader(body))
			r.ContentLength = int64(len(body))
		}
	}

	var responseHeaders map[string]string

	var transport http.RoundTripper
	if route.ChromeTransport {
		transport = p.chromeTransport
	}

	record := func(usage sse.UsageData, statusCode int, streaming bool, respBodyPreview string) {
		elapsed := time.Since(start).Milliseconds()
		cost := p.calculateCost(usage)
		p.writer.WriteUsage(usage, meta, cost, elapsed, statusCode, time.Now(), requestID)
		LogRequest(RequestLog{
			Type:                "request_log",
			RequestID:           requestID,
			Timestamp:           time.Now(),
			Provider:            meta.Provider,
			Model:               usage.Model,
			Device:              meta.Device,
			Harness:             meta.Harness,
			AuthType:            meta.AuthType,
			StatusCode:          statusCode,
			ResponseTimeMs:      elapsed,
			InputTokens:         usage.InputTokens,
			OutputTokens:        usage.OutputTokens,
			CacheReadTokens:     usage.CacheReadTokens,
			CacheCreationTokens: usage.CacheCreationTokens,
			CostUSD:             cost,
			RequestMethod:       requestMethod,
			RequestPath:         routePath,
			RequestHeaders:      requestHeaders,
			ResponseHeaders:     responseHeaders,
			RequestBodyPreview:  requestBodyPreview,
			ResponseBodyPreview: respBodyPreview,
			IsStreaming:         streaming,
		})
	}

	rp := &httputil.ReverseProxy{
		Transport: transport,
		Director: func(req *http.Request) {
			req.URL.Scheme = route.Upstream.Scheme
			req.URL.Host = route.Upstream.Host
			req.Host = route.Upstream.Host
			for _, h := range stripHeaders {
				req.Header.Del(h)
			}
		},
		ModifyResponse: func(resp *http.Response) error {
			if resp.StatusCode == http.StatusSwitchingProtocols {
				log.Printf("websocket upgrade [%s]: %s", route.Provider, r.URL.Path)
				return nil
			}

			responseHeaders = SanitizeHeaders(resp.Header)

			ct := resp.Header.Get("Content-Type")
			isSSE := strings.Contains(ct, "text/event-stream") ||
				(ct == "" && resp.StatusCode == 200)

			if isSSE {
				resp.Body = NewTapReader(resp.Body, route.Provider, func(usage sse.UsageData) {
					record(usage, resp.StatusCode, true, "")
				})
			} else {
				body, err := io.ReadAll(resp.Body)
				_ = resp.Body.Close()
				if err != nil {
					return fmt.Errorf("reading response body: %w", err)
				}
				usage, extractErr := sse.ExtractNonStreamingUsage(body, route.Provider)
				if extractErr != nil {
					log.Printf("non-streaming extraction failed [%s]: %v (body_len=%d)", route.Provider, extractErr, len(body))
				}
				if extractErr == nil {
					record(usage, resp.StatusCode, false, TruncateBody(body))
				}
				resp.Body = io.NopCloser(bytes.NewReader(body))
			}
			return nil
		},
		ErrorHandler: func(w http.ResponseWriter, r *http.Request, err error) {
			log.Printf("proxy error [%s]: %v", route.Provider, err)
			http.Error(w, "upstream error", http.StatusBadGateway)
		},
	}

	rp.ServeHTTP(w, r)
}

func (p *Proxy) calculateCost(usage sse.UsageData) float64 {
	if p.pricing == nil {
		return 0
	}
	return p.pricing.Calculate(
		usage.Model,
		usage.InputTokens,
		usage.OutputTokens,
		usage.CacheReadTokens,
		usage.CacheCreationTokens,
	)
}
