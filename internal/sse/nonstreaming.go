package sse

import (
	"encoding/json"
	"fmt"
)

func ExtractNonStreamingUsage(body []byte, provider string) (UsageData, error) {
	switch provider {
	case "anthropic":
		return extractAnthropicNonStreaming(body)
	case "openai", "openrouter", "xai":
		return extractOpenAINonStreaming(body)
	default:
		return UsageData{}, fmt.Errorf("unknown provider: %s", provider)
	}
}

func extractAnthropicNonStreaming(body []byte) (UsageData, error) {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			InputTokens              int64 `json:"input_tokens"`
			OutputTokens             int64 `json:"output_tokens"`
			CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
			CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return UsageData{}, err
	}
	return UsageData{
		Model:               resp.Model,
		InputTokens:         resp.Usage.InputTokens,
		OutputTokens:        resp.Usage.OutputTokens,
		CacheReadTokens:     resp.Usage.CacheReadInputTokens,
		CacheCreationTokens: resp.Usage.CacheCreationInputTokens,
	}, nil
}

func extractOpenAINonStreaming(body []byte) (UsageData, error) {
	var resp struct {
		Model string `json:"model"`
		Usage struct {
			PromptTokens        int64 `json:"prompt_tokens"`
			CompletionTokens    int64 `json:"completion_tokens"`
			PromptTokensDetails *struct {
				CachedTokens int64 `json:"cached_tokens"`
			} `json:"prompt_tokens_details"`
		} `json:"usage"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return UsageData{}, err
	}
	u := UsageData{
		Model:        resp.Model,
		OutputTokens: resp.Usage.CompletionTokens,
	}
	if d := resp.Usage.PromptTokensDetails; d != nil && d.CachedTokens > 0 {
		u.CacheReadTokens = d.CachedTokens
		u.InputTokens = resp.Usage.PromptTokens - d.CachedTokens
	} else {
		u.InputTokens = resp.Usage.PromptTokens
	}
	return u, nil
}
