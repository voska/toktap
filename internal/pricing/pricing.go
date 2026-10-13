package pricing

import (
	"log"
	"os"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type ModelPricing struct {
	InputPerM         float64 `yaml:"input_per_m"`
	OutputPerM        float64 `yaml:"output_per_m"`
	CacheReadPerM     float64 `yaml:"cache_read_per_m"`
	CacheCreationPerM float64 `yaml:"cache_creation_per_m"`
}

type Table struct {
	Models map[string]ModelPricing `yaml:"models"`
	mu     sync.RWMutex
}

func LoadFromFile(path string) (*Table, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Table
	if err := yaml.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (t *Table) Calculate(model string, input, output, cacheRead, cacheCreation int64) float64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	realModelName := model

	lastDash := strings.LastIndex(model, "-")
	if lastDash != -1 {
		lenOfLastSegment := len(model) - lastDash - 1
		switch lenOfLastSegment {
		case 2, 4: // model format is gpt-4-2024-08-06 or gpt-4-32k-06-08-2024
			modelLength := len(model) - 11 // length of date segment in model name (e.g., "-2024-08-06")
			model = model[:modelLength]
		case 8: // model format is claude-2-20240806
			model = model[:lastDash]
		}
	}

	log.Printf("Model name transformed from '%s' to '%s' for cost calculation", realModelName, model)

	p, ok := t.Models[model]

	if !ok {
		return 0
	}
	return (float64(input)/1e6)*p.InputPerM +
		(float64(output)/1e6)*p.OutputPerM +
		(float64(cacheRead)/1e6)*p.CacheReadPerM +
		(float64(cacheCreation)/1e6)*p.CacheCreationPerM
}

func (t *Table) Reload(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var updated Table
	if err := yaml.Unmarshal(data, &updated); err != nil {
		return err
	}
	t.mu.Lock()
	t.Models = updated.Models
	t.mu.Unlock()
	return nil
}
