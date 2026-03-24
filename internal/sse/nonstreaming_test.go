package sse

import "testing"

func TestExtractAnthropicNonStreaming(t *testing.T) {
	body := `{"id":"msg_1","model":"claude-opus-4-6","content":[{"text":"hello"}],"usage":{"input_tokens":100,"output_tokens":50,"cache_creation_input_tokens":10,"cache_read_input_tokens":200}}`
	u, err := ExtractNonStreamingUsage([]byte(body), "anthropic")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Model != "claude-opus-4-6" {
		t.Errorf("model = %q", u.Model)
	}
	if u.InputTokens != 100 || u.OutputTokens != 50 {
		t.Errorf("tokens: input=%d output=%d", u.InputTokens, u.OutputTokens)
	}
	if u.CacheReadTokens != 200 || u.CacheCreationTokens != 10 {
		t.Errorf("cache: read=%d creation=%d", u.CacheReadTokens, u.CacheCreationTokens)
	}
}

func TestExtractOpenAINonStreaming(t *testing.T) {
	body := `{"id":"chatcmpl-1","model":"gpt-5.4","choices":[{}],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`
	u, err := ExtractNonStreamingUsage([]byte(body), "openai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if u.Model != "gpt-5.4" {
		t.Errorf("model = %q", u.Model)
	}
	if u.InputTokens != 100 || u.OutputTokens != 50 {
		t.Errorf("tokens: input=%d output=%d", u.InputTokens, u.OutputTokens)
	}
}

func TestExtractInvalidJSON(t *testing.T) {
	_, err := ExtractNonStreamingUsage([]byte("not json"), "anthropic")
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
