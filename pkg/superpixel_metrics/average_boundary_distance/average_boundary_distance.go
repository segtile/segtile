package average_boundary_distance

import (
	"fmt"
	"image"
	"math"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpMetricAverageBoundaryDistance struct {
	superpixel_metrics.SpMetricsData
}

func (e *SpMetricAverageBoundaryDistance) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricAverageBoundaryDistance) SetParameter(parameter superpixel_metrics.SpMetricParameter, value float64) error {
	return fmt.Errorf("No parameters available for AverageBoundaryDistance metric.")
}

func (e *SpMetricAverageBoundaryDistance) GetParameter(parameter superpixel_metrics.SpMetricParameter) (float64, error) {
	return 0, fmt.Errorf("No parameters available for AverageBoundaryDistance metric.")
}

func (e *SpMetricAverageBoundaryDistance) GetSupportedParameters() []superpixel_metrics.SpMetricParameter {
	return []superpixel_metrics.SpMetricParameter{}
}

func (e *SpMetricAverageBoundaryDistance) GetScore() (averageBoundaryDistance float64, err error) {

	const maxIslands = 5

	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(e.GetGroundTruthSegmentation())

	// Prepare metrics
	averageBoundaryDistance = 0.0

	// Build a list of all island labels
	islandLabels := make([]uint16, groundTruthIslandCount)
	for i := uint16(0); i < groundTruthIslandCount; i++ {
		islandLabels[i] = i + 1
	}

	// Shuffle the island labels and pick up to 10 for evaluation
	rand.Shuffle(len(islandLabels), func(i, j int) {
		islandLabels[i], islandLabels[j] = islandLabels[j], islandLabels[i]
	})
	if len(islandLabels) > maxIslands {
		islandLabels = islandLabels[:maxIslands]
	}

	for _, islandLabel := range islandLabels {

		currentGroundTruthIsland := superpixel_utils.SelectIsland(groundTruthIslands, islandLabel)

		selectionFromCurrentIsland := e.GetSpResult().SelectFromMask(currentGroundTruthIsland, 0.5)
		currentApproximatedMask := e.GetSpResult().SelectionToMask(selectionFromCurrentIsland)

		currentAverageBoundaryDistance, _ := AverageBoundaryDistance(currentGroundTruthIsland, currentApproximatedMask)
		averageBoundaryDistance += currentAverageBoundaryDistance

	}

	averageBoundaryDistance /= float64(len(islandLabels))

	return averageBoundaryDistance, nil
}

// AverageBoundaryDistance computes the average distance from each boundary pixel in maskOriginal to the nearest boundary pixel in maskApproximated.
func AverageBoundaryDistance(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
	if maskOriginal.Bounds() != maskApproximated.Bounds() {
		return 0, fmt.Errorf("Masks must have the same dimensions")
	}

	bounds := maskOriginal.Bounds()

	maxDistance2 := float64(bounds.Dx()*bounds.Dx() + bounds.Dy()*bounds.Dy()) // Max possible distance squared

	// Collect boundary pixel positions for both masks
	type point struct{ x, y int }
	var boundariesOriginal, boundariesApproximated []point
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if maskOriginal.Gray16At(x, y).Y > 0 && (maskOriginal.Gray16At(x-1, y).Y == 0 || maskOriginal.Gray16At(x+1, y).Y == 0 || maskOriginal.Gray16At(x, y-1).Y == 0 || maskOriginal.Gray16At(x, y+1).Y == 0) {
				boundariesOriginal = append(boundariesOriginal, point{x, y})
			}
			if maskApproximated.Gray16At(x, y).Y > 0 && (maskApproximated.Gray16At(x-1, y).Y == 0 || maskApproximated.Gray16At(x+1, y).Y == 0 || maskApproximated.Gray16At(x, y-1).Y == 0 || maskApproximated.Gray16At(x, y+1).Y == 0) {
				boundariesApproximated = append(boundariesApproximated, point{x, y})
			}
		}
	}

	if len(boundariesOriginal) == 0 || len(boundariesApproximated) == 0 {
		return 0, nil
	}

	// Check each boundary pixel in maskOriginal against all boundary pixels in maskApproximated
	bestDistancesSum := 0.0
	for _, p := range boundariesOriginal {
		bestDistance2 := maxDistance2 // Max possible distance squared
		for _, q := range boundariesApproximated {
			deltaX := float64(p.x - q.x)
			deltaY := float64(p.y - q.y)
			distance2 := deltaX*deltaX + deltaY*deltaY
			if distance2 < bestDistance2 {
				bestDistance2 = distance2
			}
		}
		bestDistancesSum += math.Sqrt(bestDistance2)
	}

	return bestDistancesSum / float64(len(boundariesOriginal)), nil
}
