package sse

import "encoding/json"

type OpenAIExtractor struct {
	usage UsageData
}

func NewOpenAIExtractor() *OpenAIExtractor {
	return &OpenAIExtractor{}
}

func (e *OpenAIExtractor) ProcessEvent(ev Event) {
	if ev.Data == "[DONE]" {
		return
	}
	if ev.Type == "response.completed" {
		var resp struct {
			Response struct {
				Model string `json:"model"`
				Usage struct {
					InputTokens        int64 `json:"input_tokens"`
					OutputTokens       int64 `json:"output_tokens"`
					InputTokensDetails *struct {
						CachedTokens int64 `json:"cached_tokens"`
					} `json:"input_tokens_details"`
				} `json:"usage"`
			} `json:"response"`
		}
		if json.Unmarshal([]byte(ev.Data), &resp) == nil && resp.Response.Model != "" {
			e.usage.Model = resp.Response.Model
			e.usage.OutputTokens = resp.Response.Usage.OutputTokens
			if d := resp.Response.Usage.InputTokensDetails; d != nil && d.CachedTokens > 0 {
				e.usage.CacheReadTokens = d.CachedTokens
				e.usage.InputTokens = resp.Response.Usage.InputTokens - d.CachedTokens
			} else {
				e.usage.InputTokens = resp.Response.Usage.InputTokens
			}
		}
		return
	}
	var chunk struct {
		Model   string `json:"model"`
		Choices []struct {
			Delta json.RawMessage `json:"delta"`
		} `json:"choices"`
		Usage *struct {
			PromptTokens        int64 `json:"prompt_tokens"`
			CompletionTokens    int64 `json:"completion_tokens"`
			PromptTokensDetails *struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}
	if json.Unmarshal([]byte(ev.Data), &chunk) != nil {
		return
	}
	if e.usage.Model == "" && chunk.Model != "" {
		e.usage.Model = chunk.Model
	}
	if chunk.Usage != nil {
		e.usage.OutputTokens = chunk.Usage.CompletionTokens
		if d := chunk.Usage.PromptTokensDetails; d != nil && d.CachedTokens > 0 {
			e.usage.CacheReadTokens = d.CachedTokens
			e.usage.InputTokens = chunk.Usage.PromptTokens - d.CachedTokens
		} else {
			e.usage.InputTokens = chunk.Usage.PromptTokens
		}
	}
}

func (e *OpenAIExtractor) Usage() UsageData {
	return e.usage
}
