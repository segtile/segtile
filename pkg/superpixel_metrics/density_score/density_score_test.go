package density_score

import (
	"math"
	"testing"

	"github.com/segtile/segtile/pkg/superpixel"
	"github.com/segtile/segtile/pkg/superpixel_map"
)

func TestRunHandlesThreeGroundTruthComponents(t *testing.T) {
	labelMap := superpixel_map.NewLabelMap(3, 1)
	labelMap.SetValueAt(1, 0, 1)
	labelMap.SetValueAt(2, 0, 2)
	result := superpixel.NewSpResult(labelMap)

	groundTruth := superpixel_map.NewMask(3, 1)
	groundTruth.SetValueAt(0, 0, true)
	groundTruth.SetValueAt(2, 0, true)

	score, err := Run(result, groundTruth)
	if err != nil {
		t.Fatalf("Run() returned an error: %v", err)
	}
	if math.IsNaN(score) || math.IsInf(score, 0) {
		t.Fatalf("Run() returned a non-finite score: %v", score)
	}
	want := 1 - math.Pow(1.0/3, 0.25)
	if math.Abs(score-want) > 1e-12 {
		t.Fatalf("Run() score = %v, want %v", score, want)
	}
}
