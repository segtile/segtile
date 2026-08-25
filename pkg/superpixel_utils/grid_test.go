package superpixel_utils

import (
	"testing"
)

func TestHexagonalGridPoints(t *testing.T) {
	width := 64
	height := 64
	targetCount := 16

	points, gridInterval := HexagonalGridPoints(width, height, targetCount)

	if len(points) == 0 {
		t.Fatalf("expected non-empty points, got 0")
	}

	if gridInterval <= 0 {
		t.Fatalf("expected positive gridInterval, got %d", gridInterval)
	}

	for _, pt := range points {
		if pt.X < 0 || pt.X >= width || pt.Y < 0 || pt.Y >= height {
			t.Errorf("point %v out of bounds (%d, %d)", pt, width, height)
		}
	}
}

func TestHexagonalGridPoints_EdgeCases(t *testing.T) {
	// Zero / negative bounds
	points, _ := HexagonalGridPoints(0, 0, 10)
	if len(points) != 0 {
		t.Errorf("expected 0 points for 0x0 image, got %d", len(points))
	}

	points, _ = HexagonalGridPoints(10, 10, 0)
	if len(points) != 0 {
		t.Errorf("expected 0 points for targetCount 0, got %d", len(points))
	}
}
