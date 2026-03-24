package sse

import "testing"

func TestOpenAIExtractUsage(t *testing.T) {
	ex := NewOpenAIExtractor()
	ex.ProcessEvent(Event{
		Data: `{"id":"chatcmpl-1","model":"gpt-5.4","choices":[{"delta":{"content":"hi"}}]}`,
	})
	ex.ProcessEvent(Event{
		Data: `{"id":"chatcmpl-1","model":"gpt-5.4","choices":[],"usage":{"prompt_tokens":100,"completion_tokens":50,"total_tokens":150}}`,
	})
	u := ex.Usage()
	if u.Model != "gpt-5.4" {
		t.Errorf("model = %q, want %q", u.Model, "gpt-5.4")
	}
	if u.InputTokens != 100 {
		t.Errorf("input = %d, want 100", u.InputTokens)
	}
	if u.OutputTokens != 50 {
		t.Errorf("output = %d, want 50", u.OutputTokens)
	}
}

func TestOpenAISkipsDone(t *testing.T) {
	ex := NewOpenAIExtractor()
	ex.ProcessEvent(Event{Data: "[DONE]"})
	u := ex.Usage()
	if u.InputTokens != 0 {
		t.Error("expected zero usage for [DONE]")
	}
}

func TestOpenAIExtractFromResponsesAPI(t *testing.T) {
	ex := NewOpenAIExtractor()
	ex.ProcessEvent(Event{
		Type: "response.completed",
		Data: `{"type":"response.completed","response":{"model":"gpt-5.4","usage":{"input_tokens":200,"output_tokens":80,"total_tokens":280}}}`,
	})
	u := ex.Usage()
	if u.Model != "gpt-5.4" {
		t.Errorf("model = %q, want %q", u.Model, "gpt-5.4")
	}
	if u.InputTokens != 200 {
		t.Errorf("input = %d, want 200", u.InputTokens)
	}
	if u.OutputTokens != 80 {
		t.Errorf("output = %d, want 80", u.OutputTokens)
	}
}
