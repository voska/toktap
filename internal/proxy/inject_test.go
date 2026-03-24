package proxy

import (
	"encoding/json"
	"testing"
)

func TestInjectStreamOptions(t *testing.T) {
	input := `{"model":"gpt-5.4","messages":[{"role":"user","content":"hi"}],"stream":true}`
	result, modified, err := InjectStreamOptions([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !modified {
		t.Error("expected modified=true")
	}
	var parsed map[string]interface{}
	json.Unmarshal(result, &parsed)
	so, ok := parsed["stream_options"].(map[string]interface{})
	if !ok {
		t.Fatal("stream_options not found")
	}
	if so["include_usage"] != true {
		t.Error("include_usage not true")
	}
}

func TestInjectStreamOptionsAlreadySet(t *testing.T) {
	input := `{"model":"gpt-5.4","stream":true,"stream_options":{"include_usage":true}}`
	_, modified, err := InjectStreamOptions([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Error("expected modified=false when already set")
	}
}

func TestInjectStreamOptionsNotStreaming(t *testing.T) {
	input := `{"model":"gpt-5.4","messages":[{"role":"user","content":"hi"}]}`
	_, modified, err := InjectStreamOptions([]byte(input))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if modified {
		t.Error("expected modified=false when not streaming")
	}
}
