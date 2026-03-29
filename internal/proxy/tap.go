package proxy

import (
	"io"

	"github.com/voska/toktap/internal/recorder"
	"github.com/voska/toktap/internal/sse"
)

type usageExtractor interface {
	ProcessEvent(sse.Event)
	Usage() sse.UsageData
}

type TapReader struct {
	inner     io.ReadCloser
	scanner   *sse.Scanner
	extractor usageExtractor
	onUsage   func(sse.UsageData)
}

func NewTapReader(inner io.ReadCloser, provider string, onUsage func(sse.UsageData)) *TapReader {
	var ex usageExtractor
	switch provider {
	case "anthropic":
		ex = sse.NewAnthropicExtractor()
	default:
		ex = sse.NewOpenAIExtractor()
	}
	return &TapReader{
		inner:     inner,
		scanner:   sse.NewScanner(),
		extractor: ex,
		onUsage:   onUsage,
	}
}

func (t *TapReader) Read(p []byte) (int, error) {
	n, err := t.inner.Read(p)
	if n > 0 {
		for _, ev := range t.scanner.Feed(p[:n]) {
			t.extractor.ProcessEvent(ev)
		}
	}
	return n, err
}

func (t *TapReader) Close() error {
	if t.onUsage != nil {
		t.onUsage(t.extractor.Usage())
	}
	return t.inner.Close()
}

// RecordingTapReader extends TapReader to also capture all SSE events for the recorder.
type RecordingTapReader struct {
	inner     io.ReadCloser
	scanner   *sse.Scanner
	extractor usageExtractor
	onDone    func(sse.UsageData, []recorder.SSEEvent)
	events    []recorder.SSEEvent
}

func NewRecordingTapReader(inner io.ReadCloser, provider string, onDone func(sse.UsageData, []recorder.SSEEvent)) *RecordingTapReader {
	var ex usageExtractor
	switch provider {
	case "anthropic":
		ex = sse.NewAnthropicExtractor()
	default:
		ex = sse.NewOpenAIExtractor()
	}
	return &RecordingTapReader{
		inner:     inner,
		scanner:   sse.NewScanner(),
		extractor: ex,
		onDone:    onDone,
	}
}

func (t *RecordingTapReader) Read(p []byte) (int, error) {
	n, err := t.inner.Read(p)
	if n > 0 {
		for _, ev := range t.scanner.Feed(p[:n]) {
			t.extractor.ProcessEvent(ev)
			t.events = append(t.events, recorder.SSEEvent{
				Type: ev.Type,
				Data: ev.Data,
			})
		}
	}
	return n, err
}

func (t *RecordingTapReader) Close() error {
	if t.onDone != nil {
		t.onDone(t.extractor.Usage(), t.events)
	}
	return t.inner.Close()
}
