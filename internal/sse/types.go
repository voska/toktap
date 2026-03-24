package sse

type UsageData struct {
	Model               string
	InputTokens         int64
	OutputTokens        int64
	CacheReadTokens     int64
	CacheCreationTokens int64
}

type Event struct {
	Type string
	Data string
}
