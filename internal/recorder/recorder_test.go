package recorder

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestRecorderWritesJSONL(t *testing.T) {
	dir := t.TempDir()
	rec, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()

	ts := time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC)
	rec.Write(Record{
		ID:          "test-1",
		Timestamp:   ts,
		Provider:    "anthropic",
		Model:       "claude-opus-4-6",
		RequestBody: json.RawMessage(`{"messages":[{"role":"user","content":"hello"}]}`),
		IsStreaming: true,
		SSEEvents: []SSEEvent{
			{Type: "message_start", Data: `{"message":{"model":"claude-opus-4-6"}}`},
			{Type: "content_block_delta", Data: `{"delta":{"text":"Hi"}}`},
		},
		InputTokens:  100,
		OutputTokens: 50,
		DurationMs:   1500,
	})

	rec.Close()

	path := filepath.Join(dir, "2026-03-29.jsonl")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("reading file: %v", err)
	}

	var got Record
	if err := json.Unmarshal(data[:len(data)-1], &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if got.ID != "test-1" {
		t.Errorf("ID = %q, want test-1", got.ID)
	}
	if got.Provider != "anthropic" {
		t.Errorf("Provider = %q, want anthropic", got.Provider)
	}
	if len(got.SSEEvents) != 2 {
		t.Errorf("SSEEvents len = %d, want 2", len(got.SSEEvents))
	}
	if got.InputTokens != 100 {
		t.Errorf("InputTokens = %d, want 100", got.InputTokens)
	}
}

func TestRecorderRotatesDaily(t *testing.T) {
	dir := t.TempDir()
	rec, err := New(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer rec.Close()

	rec.Write(Record{
		ID:        "day1",
		Timestamp: time.Date(2026, 3, 28, 12, 0, 0, 0, time.UTC),
	})
	rec.Write(Record{
		ID:        "day2",
		Timestamp: time.Date(2026, 3, 29, 12, 0, 0, 0, time.UTC),
	})

	rec.Close()

	for _, day := range []string{"2026-03-28.jsonl", "2026-03-29.jsonl"} {
		if _, err := os.Stat(filepath.Join(dir, day)); err != nil {
			t.Errorf("missing file %s: %v", day, err)
		}
	}
}
