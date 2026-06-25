package regularity_score

import (
	"math"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpMetricRegularityScore struct {
	superpixel_metrics.SpMetricsData
}

func (e *SpMetricRegularityScore) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricRegularityScore) GetScore() (regularityCalibrationScore float64) {

	const maxIslands = 5

	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(e.GetGroundTruthSegmentation())

	// Prepare metrics
	regularityCalibrationScore = 0.0

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

		// Regularity Calibration Score
		groundTruthFillRatio := FillRatioOfOwnBoundingBox(currentGroundTruthIsland)
		superpixelFillRatio := FillRatioOfOwnBoundingBoxPerSuperpixel(e.GetSpResult().LabelMap, e.GetSpResult().SpCount)
		if groundTruthFillRatio > 0.0 || superpixelFillRatio > 0.0 {
			regularityCalibrationScore += math.Min(groundTruthFillRatio, superpixelFillRatio) / math.Max(groundTruthFillRatio, superpixelFillRatio)
		}

	}

	regularityCalibrationScore /= float64(len(islandLabels))

	return regularityCalibrationScore
}
