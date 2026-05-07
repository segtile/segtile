package main

import (
	"testing"

	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/util"
)

func TestBuildRunConfigurations_SingleRange(t *testing.T) {
	ranges := []util.ParameterRange{
		{Name: "count", Start: 100, End: 300, Step: 100},
	}
	configs, err := buildRunParameterSets(ranges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(configs) != 3 {
		t.Fatalf("expected 3 configurations, got %d", len(configs))
	}

	expected := []float64{100, 200, 300}
	for i, cfg := range configs {
		if cfg[superpixel.Count] != expected[i] {
			t.Errorf("config[%d][Count] = %v, want %v", i, cfg[superpixel.Count], expected[i])
		}
	}
}

func TestBuildRunConfigurations_MultipleRanges(t *testing.T) {
	ranges := []util.ParameterRange{
		{Name: "count", Start: 100, End: 200, Step: 20},
		{Name: "compactness", Start: 0, End: 10, Step: 2},
	}
	configs, err := buildRunParameterSets(ranges)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// 6 count values × 6 compactness values = 36 configurations
	if len(configs) != 36 {
		t.Fatalf("expected 36 configurations, got %d", len(configs))
	}

	// Verify all combinations are present
	type pair struct{ count, compactness float64 }
	seen := make(map[pair]bool)
	for _, cfg := range configs {
		seen[pair{cfg[superpixel.Count], cfg[superpixel.Compactness]}] = true
	}
	expected := []pair{
		{100, 0}, {100, 2}, {100, 4}, {100, 6}, {100, 8}, {100, 10},
		{120, 0}, {120, 2}, {120, 4}, {120, 6}, {120, 8}, {120, 10},
		{140, 0}, {140, 2}, {140, 4}, {140, 6}, {140, 8}, {140, 10},
		{160, 0}, {160, 2}, {160, 4}, {160, 6}, {160, 8}, {160, 10},
		{180, 0}, {180, 2}, {180, 4}, {180, 6}, {180, 8}, {180, 10},
		{200, 0}, {200, 2}, {200, 4}, {200, 6}, {200, 8}, {200, 10},
	}
	for _, e := range expected {
		if !seen[e] {
			t.Errorf("missing combination count=%.0f compactness=%.0f", e.count, e.compactness)
		}
	}
}

func TestBuildRunConfigurations_EmptyRanges(t *testing.T) {
	configs, err := buildRunParameterSets([]util.ParameterRange{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(configs) != 1 {
		t.Fatalf("expected 1 (empty) configuration, got %d", len(configs))
	}
	if len(configs[0]) != 0 {
		t.Errorf("expected empty map, got %v", configs[0])
	}
}
