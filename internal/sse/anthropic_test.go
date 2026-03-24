package sse

import "testing"

func TestAnthropicExtractMessageStart(t *testing.T) {
	ex := NewAnthropicExtractor()
	ex.ProcessEvent(Event{
		Type: "message_start",
		Data: `{"type":"message_start","message":{"id":"msg_1","model":"claude-opus-4-6","usage":{"input_tokens":25,"cache_creation_input_tokens":100,"cache_read_input_tokens":500}}}`,
	})
	u := ex.Usage()
	if u.Model != "claude-opus-4-6" {
		t.Errorf("model = %q, want %q", u.Model, "claude-opus-4-6")
	}
	if u.InputTokens != 25 {
		t.Errorf("input = %d, want 25", u.InputTokens)
	}
	if u.CacheCreationTokens != 100 {
		t.Errorf("cache_creation = %d, want 100", u.CacheCreationTokens)
	}
	if u.CacheReadTokens != 500 {
		t.Errorf("cache_read = %d, want 500", u.CacheReadTokens)
	}
}

func TestAnthropicExtractMessageDelta(t *testing.T) {
	ex := NewAnthropicExtractor()
	ex.ProcessEvent(Event{
		Type: "message_start",
		Data: `{"type":"message_start","message":{"model":"claude-opus-4-6","usage":{"input_tokens":25}}}`,
	})
	ex.ProcessEvent(Event{
		Type: "message_delta",
		Data: `{"type":"message_delta","usage":{"output_tokens":150}}`,
	})
	u := ex.Usage()
	if u.OutputTokens != 150 {
		t.Errorf("output = %d, want 150", u.OutputTokens)
	}
}

func TestAnthropicIgnoresOtherEvents(t *testing.T) {
	ex := NewAnthropicExtractor()
	ex.ProcessEvent(Event{Type: "content_block_start", Data: `{}`})
	ex.ProcessEvent(Event{Type: "content_block_delta", Data: `{"delta":{"text":"hello"}}`})
	u := ex.Usage()
	if u.InputTokens != 0 || u.OutputTokens != 0 {
		t.Error("expected zero usage for non-usage events")
	}
}
