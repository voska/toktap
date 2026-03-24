package proxy

import (
	"net/http"
	"regexp"
	"strings"
)

type Metadata struct {
	Route    string
	Provider string
	Device   string
	Harness  string
	AuthType string
}

func ExtractMetadata(r *http.Request, route, provider string) Metadata {
	return Metadata{
		Route:    route,
		Provider: provider,
		Device:   extractDevice(r),
		Harness:  extractHarness(r),
		AuthType: extractAuthType(r),
	}
}

var validDeviceName = regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)

func extractDevice(r *http.Request) string {
	if d := r.Header.Get("X-Device"); d != "" && validDeviceName.MatchString(d) {
		return d
	}
	return "unknown"
}

func extractHarness(r *http.Request) string {
	ua := strings.ToLower(r.Header.Get("User-Agent"))
	switch {
	case strings.HasPrefix(ua, "claude-code"), strings.HasPrefix(ua, "claude-cli"):
		return "claude-code"
	case strings.HasPrefix(ua, "codex") || strings.Contains(ua, "codex"):
		return "codex"
	case strings.HasPrefix(ua, "anthropic-"):
		return "sdk"
	case strings.HasPrefix(ua, "openai-"):
		return "sdk"
	default:
		return "unknown"
	}
}

func extractAuthType(r *http.Request) string {
	if r.Header.Get("x-api-key") != "" {
		return "api_key"
	}
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		if strings.HasPrefix(token, "sk-") {
			return "api_key"
		}
		return "oauth"
	}
	return "unknown"
}
