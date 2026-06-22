package regularity_score

import (
	"image"
	"math"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpEvaluatorDensityCalibrationScore struct {
	spResult *superpixel.SpResult
}

func (e *SpEvaluatorDensityCalibrationScore) SetSpResult(spResult *superpixel.SpResult) {
	e.spResult = spResult
}

func (e *SpEvaluatorDensityCalibrationScore) GetSpResult() *superpixel.SpResult {
	return e.spResult
}

func (e *SpEvaluatorDensityCalibrationScore) PerfectScore() float64 {
	return 1.0
}

func (e *SpEvaluatorDensityCalibrationScore) GetScore(groundTruthMask *image.Gray16, spResult *superpixel.SpResult, maxIslands int) (regularityCalibrationScore float64) {

	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruthMask)

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
		superpixelFillRatio := FillRatioOfOwnBoundingBoxPerSuperpixel(spResult.LabelMap, spResult.SpCount)
		if groundTruthFillRatio > 0.0 || superpixelFillRatio > 0.0 {
			regularityCalibrationScore += math.Min(groundTruthFillRatio, superpixelFillRatio) / math.Max(groundTruthFillRatio, superpixelFillRatio)
		}

	}

	regularityCalibrationScore /= float64(len(islandLabels))

	return regularityCalibrationScore
}
