package density_score

import (
	"image"
	"math"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpMetricDensityCalibrationScore struct {
	spResult *superpixel.SpResult
}

func (e *SpMetricDensityCalibrationScore) SetSpResult(spResult *superpixel.SpResult) {
	e.spResult = spResult
}

func (e *SpMetricDensityCalibrationScore) GetSpResult() *superpixel.SpResult {
	return e.spResult
}

func (e *SpMetricDensityCalibrationScore) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricDensityCalibrationScore) GetScore(groundTruthMask *image.Gray16, spResult *superpixel.SpResult, maxIslands int) (densityCalibrationScore float64) {

	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruthMask)

	// Prepare metrics
	densityCalibrationScore = 0.0

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

	groundTruthMaskArea := groundTruthMask.Bounds().Dx() * groundTruthMask.Bounds().Dy()

	for _, islandLabel := range islandLabels {

		currentGroundTruthIsland := superpixel_utils.SelectIsland(groundTruthIslands, islandLabel)
		currentGroundTruthIslandArea := superpixel_metrics.Area(currentGroundTruthIsland)

		selectionFromCurrentIsland := spResult.SelectFromMask(currentGroundTruthIsland, 0.5)
		currentApproximatedMask := spResult.SelectionToMask(selectionFromCurrentIsland)
		currentApproximatedMaskArea := superpixel_metrics.Area(currentApproximatedMask)

		_, currentSelectedBorderingSpCount := superpixel_utils.CountBorderingSuperpixels(spResult.LabelMap, selectionFromCurrentIsland)

		currentBorderingSpSelection := superpixel_utils.SelectBorderingSuperpixels(spResult.LabelMap, selectionFromCurrentIsland)
		currentBorderingSpMask := spResult.SelectionToMask(currentBorderingSpSelection)
		currentBorderingSpMaskArea := superpixel_metrics.Area(currentBorderingSpMask)

		// Density Calibration Score
		if currentSelectedBorderingSpCount > 0 {
			borderPortionOfSelection := float64(currentBorderingSpMaskArea) / float64(currentApproximatedMaskArea)
			desiredBorderPortionOfSelection := math.Pow(1-(float64(currentGroundTruthIslandArea)/float64(groundTruthMaskArea)), 2)
			densityCalibrationScore += math.Min(borderPortionOfSelection, desiredBorderPortionOfSelection) / math.Max(borderPortionOfSelection, desiredBorderPortionOfSelection)
		}

	}

	densityCalibrationScore /= float64(len(islandLabels))

	return densityCalibrationScore
}
