package proxy

import (
	"io"

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
