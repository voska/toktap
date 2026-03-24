package proxy

import (
	"io"
	"strings"
	"testing"

	"github.com/voska/toktap/internal/sse"
)

func TestTapReaderAnthropicStream(t *testing.T) {
	stream := "event: message_start\ndata: {\"type\":\"message_start\",\"message\":{\"model\":\"claude-opus-4-6\",\"usage\":{\"input_tokens\":25,\"cache_read_input_tokens\":500}}}\n\n" +
		"event: content_block_delta\ndata: {\"delta\":{\"text\":\"hello\"}}\n\n" +
		"event: message_delta\ndata: {\"type\":\"message_delta\",\"usage\":{\"output_tokens\":15}}\n\n" +
		"event: message_stop\ndata: {\"type\":\"message_stop\"}\n\n"

	var gotUsage sse.UsageData
	reader := NewTapReader(
		io.NopCloser(strings.NewReader(stream)),
		"anthropic",
		func(u sse.UsageData) { gotUsage = u },
	)

	out, err := io.ReadAll(reader)
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	reader.Close()

	if string(out) != stream {
		t.Error("output bytes differ from input")
	}
	if gotUsage.Model != "claude-opus-4-6" {
		t.Errorf("model = %q", gotUsage.Model)
	}
	if gotUsage.InputTokens != 25 {
		t.Errorf("input = %d, want 25", gotUsage.InputTokens)
	}
	if gotUsage.OutputTokens != 15 {
		t.Errorf("output = %d, want 15", gotUsage.OutputTokens)
	}
	if gotUsage.CacheReadTokens != 500 {
		t.Errorf("cache_read = %d, want 500", gotUsage.CacheReadTokens)
	}
}

func TestTapReaderOpenAIStream(t *testing.T) {
	stream := "data: {\"model\":\"gpt-5.4\",\"choices\":[{\"delta\":{\"content\":\"hi\"}}]}\n\n" +
		"data: {\"model\":\"gpt-5.4\",\"choices\":[],\"usage\":{\"prompt_tokens\":100,\"completion_tokens\":50}}\n\n" +
		"data: [DONE]\n\n"

	var gotUsage sse.UsageData
	reader := NewTapReader(
		io.NopCloser(strings.NewReader(stream)),
		"openai",
		func(u sse.UsageData) { gotUsage = u },
	)

	out, _ := io.ReadAll(reader)
	reader.Close()

	if string(out) != stream {
		t.Error("output bytes differ from input")
	}
	if gotUsage.InputTokens != 100 || gotUsage.OutputTokens != 50 {
		t.Errorf("tokens: input=%d output=%d", gotUsage.InputTokens, gotUsage.OutputTokens)
	}
}
