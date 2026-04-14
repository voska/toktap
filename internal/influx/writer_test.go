package influx

import (
	"strings"
	"testing"
	"time"

	"github.com/voska/toktap/internal/proxy"
	"github.com/voska/toktap/internal/sse"
)

type mockWriteAPI struct {
	points []string
}

func (m *mockWriteAPI) WriteRecord(line string) {
	m.points = append(m.points, line)
}

func (m *mockWriteAPI) Flush() {}

func (m *mockWriteAPI) Errors() <-chan error {
	return make(chan error)
}

func TestWriteUsage(t *testing.T) {
	mock := &mockWriteAPI{}
	w := &Writer{api: mock}

	w.WriteUsage(sse.UsageData{
		Model:               "claude-opus-4-6",
		InputTokens:         100,
		OutputTokens:        50,
		CacheReadTokens:     200,
		CacheCreationTokens: 10,
	}, proxy.Metadata{
		Provider: "anthropic",
		Device:   "raptor",
		Harness:  "claude-code",
		AuthType: "oauth",
	}, 0.058, 1500, 200, time.Now(), "test-request-id")

	if len(mock.points) != 1 {
		t.Fatalf("got %d points, want 1", len(mock.points))
	}
	line := mock.points[0]
	if line == "" {
		t.Error("empty line protocol")
	}
	for _, want := range []string{
		"llm_usage",
		"provider=anthropic",
		"model=claude-opus-4-6",
		"device=raptor",
		"harness=claude-code",
		`request_id="test-request-id"`,
		"input_tokens=100i",
		"output_tokens=50i",
		"cost_usd=0.058",
	} {
		if !strings.Contains(line, want) {
			t.Errorf("line protocol missing %q:\n  %s", want, line)
		}
	}
}
