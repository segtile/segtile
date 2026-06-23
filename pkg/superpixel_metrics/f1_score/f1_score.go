package f1_score

import (
	"fmt"
	"image"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpMetricF1Score struct {
	superpixel_metrics.SpMetricsData
}

func (e *SpMetricF1Score) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricF1Score) GetScore() (f1Score float64) {

	const maxIslands = 5

	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(e.GetGroundTruthSegmentation())

	// Prepare metrics
	f1Score = 0.0

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

		// F1 Score
		currentF1Score, _ := F1Score(currentGroundTruthIsland, currentApproximatedMask)
		f1Score += currentF1Score

	}

	f1Score /= float64(len(islandLabels))
	return f1Score
}

func F1Score(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
	if maskOriginal.Bounds() != maskApproximated.Bounds() {
		return 0, fmt.Errorf("Masks must have the same dimensions")
	}

	distanceMap := superpixel_utils.DistanceToNextBoundaryPixel(maskOriginal)

	var intersection, union int
	for y := 0; y < maskOriginal.Bounds().Dy(); y++ {
		for x := 0; x < maskOriginal.Bounds().Dx(); x++ {
			pOriginal := maskOriginal.Gray16At(x, y)
			pApproximated := maskApproximated.Gray16At(x, y)
			if pOriginal.Y > 0 && pApproximated.Y > 0 {
				intersection++
				union += 2
			} else if pOriginal.Y > 0 || pApproximated.Y > 0 {
				union += /*1 +*/ int(distanceMap.Gray16At(x, y).Y)
			}
		}
	}

	if union == 0 {
		return 0, nil
	}

	return float64(2*intersection) / float64(union), nil
}
