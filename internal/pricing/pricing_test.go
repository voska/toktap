package pricing

import (
	"math"
	"testing"
)

func TestLoadPricing(t *testing.T) {
	table, err := LoadFromFile("../../deploy/config/pricing.yaml")
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	if len(table.Models) == 0 {
		t.Fatal("expected models to be loaded")
	}
	opus, ok := table.Models["claude-opus-4-6"]
	if !ok {
		t.Fatal("claude-opus-4-6 not found")
	}
	if opus.InputPerM != 15.0 {
		t.Errorf("InputPerM = %f, want 15.0", opus.InputPerM)
	}
}

func TestCalculateCost(t *testing.T) {
	table := &Table{
		Models: map[string]ModelPricing{
			"claude-opus-4-6": {
				InputPerM:         15.0,
				OutputPerM:        75.0,
				CacheReadPerM:     1.875,
				CacheCreationPerM: 18.75,
			},
		},
	}

	cost := table.Calculate("claude-opus-4-6", 1000, 500, 2000, 100)
	// (1000/1e6)*15 + (500/1e6)*75 + (2000/1e6)*1.875 + (100/1e6)*18.75
	// = 0.015 + 0.0375 + 0.00375 + 0.001875 = 0.058125
	expected := 0.058125
	if math.Abs(cost-expected) > 0.0001 {
		t.Errorf("cost = %f, want %f", cost, expected)
	}
}

func TestCalculateCostUnknownModel(t *testing.T) {
	table := &Table{Models: map[string]ModelPricing{}}
	cost := table.Calculate("unknown-model", 1000, 500, 0, 0)
	if cost != 0 {
		t.Errorf("cost = %f, want 0 for unknown model", cost)
	}
}

func TestReload(t *testing.T) {
	table, err := LoadFromFile("../../deploy/config/pricing.yaml")
	if err != nil {
		t.Fatalf("LoadFromFile: %v", err)
	}
	origCost := table.Calculate("claude-opus-4-6", 1000000, 0, 0, 0)
	if origCost == 0 {
		t.Fatal("expected non-zero cost before reload")
	}
	err = table.Reload("../../deploy/config/pricing.yaml")
	if err != nil {
		t.Fatalf("Reload: %v", err)
	}
	reloadedCost := table.Calculate("claude-opus-4-6", 1000000, 0, 0, 0)
	if math.Abs(origCost-reloadedCost) > 0.0001 {
		t.Errorf("cost changed after reload: %f vs %f", origCost, reloadedCost)
	}
}
