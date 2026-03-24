package proxy

import (
	"net/http"
	"testing"
)

func TestExtractMetadataSetsRouteAndProvider(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://localhost/anthropic/v1/messages", nil)
	m := ExtractMetadata(req, "anthropic", "anthropic")
	if m.Route != "anthropic" {
		t.Errorf("route = %q, want anthropic", m.Route)
	}
	if m.Provider != "anthropic" {
		t.Errorf("provider = %q, want anthropic", m.Provider)
	}
}

func TestExtractMetadataChatGPTRouteOpenAIProvider(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://localhost/chatgpt/codex-responses", nil)
	m := ExtractMetadata(req, "chatgpt", "openai")
	if m.Route != "chatgpt" {
		t.Errorf("route = %q, want chatgpt", m.Route)
	}
	if m.Provider != "openai" {
		t.Errorf("provider = %q, want openai", m.Provider)
	}
}

func TestExtractDevice(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://localhost/anthropic/v1/messages", nil)
	req.Header.Set("X-Device", "raptor")
	m := ExtractMetadata(req, "anthropic", "anthropic")
	if m.Device != "raptor" {
		t.Errorf("device = %q, want raptor", m.Device)
	}
}

func TestExtractDeviceDefault(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://localhost/anthropic/v1/messages", nil)
	m := ExtractMetadata(req, "anthropic", "anthropic")
	if m.Device != "unknown" {
		t.Errorf("device = %q, want unknown", m.Device)
	}
}

func TestExtractHarness(t *testing.T) {
	tests := []struct {
		ua      string
		harness string
	}{
		{"claude-code/1.2.3", "claude-code"},
		{"claude-cli/2.1.83 (external, cli)", "claude-code"},
		{"Codex/0.99.0", "codex"},
		{"codex-rs/0.1.0", "codex"},
		{"anthropic-python/0.30.0", "sdk"},
		{"openai-python/1.0.0", "sdk"},
		{"Mozilla/5.0", "unknown"},
		{"", "unknown"},
	}
	for _, tt := range tests {
		req, _ := http.NewRequest("POST", "http://localhost/anthropic/v1/messages", nil)
		if tt.ua != "" {
			req.Header.Set("User-Agent", tt.ua)
		}
		m := ExtractMetadata(req, "anthropic", "anthropic")
		if m.Harness != tt.harness {
			t.Errorf("ua %q: harness = %q, want %q", tt.ua, m.Harness, tt.harness)
		}
	}
}

func TestExtractAuthType(t *testing.T) {
	req, _ := http.NewRequest("POST", "http://localhost/anthropic/v1/messages", nil)
	req.Header.Set("x-api-key", "sk-ant-123")
	m := ExtractMetadata(req, "anthropic", "anthropic")
	if m.AuthType != "api_key" {
		t.Errorf("auth = %q, want api_key", m.AuthType)
	}

	req2, _ := http.NewRequest("POST", "http://localhost/anthropic/v1/messages", nil)
	req2.Header.Set("Authorization", "Bearer eyJ...")
	m2 := ExtractMetadata(req2, "anthropic", "anthropic")
	if m2.AuthType != "oauth" {
		t.Errorf("auth = %q, want oauth", m2.AuthType)
	}

	req3, _ := http.NewRequest("POST", "http://localhost/openai/v1/chat/completions", nil)
	req3.Header.Set("Authorization", "Bearer sk-proj-abc123")
	m3 := ExtractMetadata(req3, "openai", "openai")
	if m3.AuthType != "api_key" {
		t.Errorf("auth = %q, want api_key for Bearer sk- token", m3.AuthType)
	}
}

func TestSplitRoutePath(t *testing.T) {
	tests := []struct {
		path      string
		wantRoute string
		wantRest  string
	}{
		{"/anthropic/v1/messages", "anthropic", "/v1/messages"},
		{"/openai/v1/chat/completions", "openai", "/v1/chat/completions"},
		{"/chatgpt/codex-responses", "chatgpt", "/codex-responses"},
		{"/openclaw", "openclaw", "/"},
		{"/", "", "/"},
	}
	for _, tt := range tests {
		route, rest := splitRoutePath(tt.path)
		if route != tt.wantRoute || rest != tt.wantRest {
			t.Errorf("splitRoutePath(%q) = (%q, %q), want (%q, %q)",
				tt.path, route, rest, tt.wantRoute, tt.wantRest)
		}
	}
}
