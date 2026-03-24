package proxy

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

type RequestLog struct {
	Type                string            `json:"type"`
	RequestID           string            `json:"request_id"`
	Timestamp           time.Time         `json:"timestamp"`
	Provider            string            `json:"provider"`
	Model               string            `json:"model"`
	Device              string            `json:"device"`
	Harness             string            `json:"harness"`
	AuthType            string            `json:"auth_type"`
	StatusCode          int               `json:"status_code"`
	ResponseTimeMs      int64             `json:"response_time_ms"`
	InputTokens         int64             `json:"input_tokens"`
	OutputTokens        int64             `json:"output_tokens"`
	CacheReadTokens     int64             `json:"cache_read_tokens"`
	CacheCreationTokens int64             `json:"cache_creation_tokens"`
	CostUSD             float64           `json:"cost_usd"`
	RequestMethod       string            `json:"request_method"`
	RequestPath         string            `json:"request_path"`
	RequestHeaders      map[string]string `json:"request_headers"`
	ResponseHeaders     map[string]string `json:"response_headers"`
	RequestBodyPreview  string            `json:"request_body_preview"`
	ResponseBodyPreview string            `json:"response_body_preview,omitempty"`
	IsStreaming         bool              `json:"is_streaming"`
}

const bodyPreviewMaxLen = 500

func SanitizeHeaders(h http.Header) map[string]string {
	out := make(map[string]string, len(h))
	for k, vals := range h {
		v := strings.Join(vals, ", ")
		kl := strings.ToLower(k)
		if kl == "authorization" || kl == "x-api-key" {
			out[k] = redactAuthValue(v)
		} else {
			out[k] = v
		}
	}
	return out
}

func redactAuthValue(v string) string {
	if strings.HasPrefix(v, "Bearer ") {
		token := strings.TrimPrefix(v, "Bearer ")
		hash := sha256.Sum256([]byte(token))
		return fmt.Sprintf("Bearer [REDACTED sha256:%x]", hash[:4])
	}
	hash := sha256.Sum256([]byte(v))
	return fmt.Sprintf("[REDACTED sha256:%x]", hash[:4])
}

func TruncateBody(body []byte) string {
	if len(body) <= bodyPreviewMaxLen {
		return string(body)
	}
	return string(body[:bodyPreviewMaxLen]) + "..."
}

func LogRequest(rl RequestLog) {
	slog.LogAttrs(context.TODO(), slog.LevelInfo, "request_log",
		slog.String("provider", rl.Provider),
		slog.String("model", rl.Model),
		slog.String("device", rl.Device),
		slog.String("harness", rl.Harness),
		slog.String("auth_type", rl.AuthType),
		slog.Int("status_code", rl.StatusCode),
		slog.Int64("response_time_ms", rl.ResponseTimeMs),
		slog.Int64("input_tokens", rl.InputTokens),
		slog.Int64("output_tokens", rl.OutputTokens),
		slog.Int64("cache_read_tokens", rl.CacheReadTokens),
		slog.Int64("cache_creation_tokens", rl.CacheCreationTokens),
		slog.Float64("cost_usd", rl.CostUSD),
		slog.String("request_path", rl.RequestPath),
		slog.Bool("is_streaming", rl.IsStreaming),
	)

	// Debug-level: full headers and body previews for troubleshooting
	slog.LogAttrs(context.TODO(), slog.LevelDebug, "request_detail",
		slog.String("request_id", rl.RequestID),
		slog.String("request_method", rl.RequestMethod),
		slog.String("request_path", rl.RequestPath),
		slog.Any("request_headers", rl.RequestHeaders),
		slog.Any("response_headers", rl.ResponseHeaders),
		slog.String("request_body_preview", rl.RequestBodyPreview),
		slog.String("response_body_preview", rl.ResponseBodyPreview),
	)
}
