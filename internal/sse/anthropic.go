package sse

import "encoding/json"

type AnthropicExtractor struct {
	usage UsageData
}

func NewAnthropicExtractor() *AnthropicExtractor {
	return &AnthropicExtractor{}
}

func (e *AnthropicExtractor) ProcessEvent(ev Event) {
	switch ev.Type {
	case "message_start":
		var msg struct {
			Message struct {
				Model string `json:"model"`
				Usage struct {
					InputTokens              int64 `json:"input_tokens"`
					CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
					CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
				} `json:"usage"`
			} `json:"message"`
		}
		if json.Unmarshal([]byte(ev.Data), &msg) == nil {
			e.usage.Model = msg.Message.Model
			e.usage.InputTokens = msg.Message.Usage.InputTokens
			e.usage.CacheCreationTokens = msg.Message.Usage.CacheCreationInputTokens
			e.usage.CacheReadTokens = msg.Message.Usage.CacheReadInputTokens
		}
	case "message_delta":
		var delta struct {
			Usage struct {
				OutputTokens int64 `json:"output_tokens"`
			} `json:"usage"`
		}
		if json.Unmarshal([]byte(ev.Data), &delta) == nil {
			e.usage.OutputTokens = delta.Usage.OutputTokens
		}
	}
}

func (e *AnthropicExtractor) Usage() UsageData {
	return e.usage
}
