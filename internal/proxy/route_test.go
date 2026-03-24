package proxy

import (
	"os"
	"testing"
)

func TestLoadRoutes(t *testing.T) {
	yaml := `routes:
  anthropic:
    upstream: https://api.anthropic.com
    provider: anthropic
  openai:
    upstream: https://api.openai.com
    provider: openai
    inject_stream_options: true
  chatgpt:
    upstream: https://chatgpt.com/backend-api/codex
    provider: openai
    chrome_transport: true
`
	f, err := os.CreateTemp(t.TempDir(), "routes-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(yaml)
	f.Close()

	routes, err := LoadRoutes(f.Name())
	if err != nil {
		t.Fatalf("LoadRoutes: %v", err)
	}
	if len(routes) != 3 {
		t.Fatalf("got %d routes, want 3", len(routes))
	}

	anth := routes["anthropic"]
	if anth.Provider != "anthropic" {
		t.Errorf("anthropic provider = %q", anth.Provider)
	}
	if anth.Upstream.Host != "api.anthropic.com" {
		t.Errorf("anthropic host = %q", anth.Upstream.Host)
	}
	if anth.InjectStreamOptions {
		t.Error("anthropic should not inject stream_options")
	}
	if anth.ChromeTransport {
		t.Error("anthropic should not use chrome transport")
	}

	oai := routes["openai"]
	if oai.Provider != "openai" {
		t.Errorf("openai provider = %q", oai.Provider)
	}
	if !oai.InjectStreamOptions {
		t.Error("openai should inject stream_options")
	}

	cg := routes["chatgpt"]
	if cg.Provider != "openai" {
		t.Errorf("chatgpt provider = %q, want openai", cg.Provider)
	}
	if !cg.ChromeTransport {
		t.Error("chatgpt should use chrome transport")
	}
	if cg.Upstream.Path != "/backend-api/codex" {
		t.Errorf("chatgpt upstream path = %q", cg.Upstream.Path)
	}
}

func TestLoadRoutesMissingProvider(t *testing.T) {
	yaml := `routes:
  bad:
    upstream: https://example.com
`
	f, err := os.CreateTemp(t.TempDir(), "routes-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(yaml)
	f.Close()

	_, err = LoadRoutes(f.Name())
	if err == nil {
		t.Error("expected error for missing provider")
	}
}

func TestLoadRoutesInvalidURL(t *testing.T) {
	yaml := `routes:
  bad:
    upstream: "://invalid"
    provider: test
`
	f, err := os.CreateTemp(t.TempDir(), "routes-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	f.WriteString(yaml)
	f.Close()

	_, err = LoadRoutes(f.Name())
	if err == nil {
		t.Error("expected error for invalid URL")
	}
}

func TestLoadRoutesFileNotFound(t *testing.T) {
	_, err := LoadRoutes("/nonexistent/routes.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}
