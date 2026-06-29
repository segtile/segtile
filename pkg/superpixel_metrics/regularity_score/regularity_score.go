package regularity_score

import (
	"fmt"
	"image"
	"math"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpMetricRegularityScore struct {
	superpixel_metrics.SpMetricsData
	maxIslands int
}

func (e *SpMetricRegularityScore) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricRegularityScore) SetParameter(parameter superpixel_metrics.SpMetricParameter, value float64) error {
	switch parameter {
	case MaxIslands:
		e.maxIslands = int(value)
		return nil
	default:
		return fmt.Errorf("Unknown parameter for Regularity Score metric: %s", parameter)
	}
}

func (e *SpMetricRegularityScore) GetScore() (regularityCalibrationScore float64, err error) {

	islandMap, islandCount := superpixel_utils.FindIslands(e.GetGroundTruthSegmentation())

	return e.GetScoreWithCachedIslands(islandMap, islandCount)
}

func (e *SpMetricRegularityScore) GetScoreWithCachedIslands(islandMap *image.Gray16, islandCount uint16) (regularityCalibrationScore float64, err error) {

	// Prepare metrics
	regularityCalibrationScore = 0.0

	// Build a list of all island labels
	islandLabels := make([]uint16, islandCount)
	for i := uint16(0); i < islandCount; i++ {
		islandLabels[i] = i + 1
	}

	// Shuffle the island labels and pick up to e.maxIslands for evaluation
	rand.Shuffle(len(islandLabels), func(i, j int) {
		islandLabels[i], islandLabels[j] = islandLabels[j], islandLabels[i]
	})
	if len(islandLabels) > e.maxIslands {
		islandLabels = islandLabels[:e.maxIslands]
	}

	for _, islandLabel := range islandLabels {

		currentGroundTruthIsland := superpixel_utils.SelectIsland(islandMap, islandLabel)

		// Regularity Calibration Score
		groundTruthFillRatio := FillRatioOfOwnBoundingBox(currentGroundTruthIsland)
		superpixelFillRatio := FillRatioOfOwnBoundingBoxPerSuperpixel(e.GetSpResult().LabelMap, e.GetSpResult().SpCount)
		if groundTruthFillRatio > 0.0 || superpixelFillRatio > 0.0 {
			regularityCalibrationScore += math.Min(groundTruthFillRatio, superpixelFillRatio) / math.Max(groundTruthFillRatio, superpixelFillRatio)
		}

	}

	regularityCalibrationScore /= float64(len(islandLabels))

	return regularityCalibrationScore, nil
}
