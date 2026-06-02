package evaluation

import (
	"image"
	"math"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel"
)

func SuperpixelEvaluation(groundTruthMask *image.Gray16, spResult *superpixel.SpResult, maxIslands int) (labelingCalibrationScore float64,
	f1Score float64, densityCalibrationScore float64, regularityCalibrationScore float64, averageBoundaryDistance float64, selectedSpCount int, selectedBorderingSpCount int) {

	groundTruthIslands, groundTruthIslandCount := FindIslands(groundTruthMask)

	// Prepare metrics
	f1Score = 0.0
	densityCalibrationScore = 0.0
	regularityCalibrationScore = 0.0
	labelingCalibrationScore = 0.0
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

		currentGroundTruthIsland := SelectIsland(groundTruthIslands, islandLabel)

		selectionFromCurrentIsland := spResult.SelectFromMask(currentGroundTruthIsland, 0.5)
		currentApproximatedMask := spResult.SelectionToMask(selectionFromCurrentIsland)

		selectedSpCount, selectedBorderingSpCount = CountBorderingSuperpixels(spResult.LabelMap, selectionFromCurrentIsland)

		// F1 Score
		currentF1Score, _ := F1Score(currentGroundTruthIsland, currentApproximatedMask)
		f1Score += currentF1Score

		// Density Calibration Score
		if selectedSpCount > 0 {
			borderPortionOfSelection := float64(selectedBorderingSpCount) / float64(selectedSpCount)
			desiredBorderPortionOfSelection := 1 - math.Pow(float64(selectedSpCount)/float64(spResult.SpCount), 0.5)
			densityCalibrationScore += math.Min(borderPortionOfSelection, desiredBorderPortionOfSelection) / math.Max(borderPortionOfSelection, desiredBorderPortionOfSelection)
		}

		// Regularity Calibration Score
		groundTruthFillRatio := FillRatioOfOwnBoundingBox(currentGroundTruthIsland)
		superpixelFillRatio := FillRatioOfOwnBoundingBoxPerSuperpixel(spResult.LabelMap, spResult.SpCount)
		if groundTruthFillRatio > 0.0 || superpixelFillRatio > 0.0 {
			regularityCalibrationScore += math.Min(groundTruthFillRatio, superpixelFillRatio) / math.Max(groundTruthFillRatio, superpixelFillRatio)
		}

		// Bonus: Average Boundary Distance
		currentAverageBoundaryDistance, _ := AverageBoundaryDistance(currentGroundTruthIsland, currentApproximatedMask)
		averageBoundaryDistance += currentAverageBoundaryDistance

	}

	f1Score /= float64(len(islandLabels))
	densityCalibrationScore /= float64(len(islandLabels))
	regularityCalibrationScore /= float64(len(islandLabels))
	averageBoundaryDistance /= float64(len(islandLabels))

	// Final Labeling Calibration Score
	labelingCalibrationScore = f1Score * densityCalibrationScore * regularityCalibrationScore

	return labelingCalibrationScore, f1Score, densityCalibrationScore, regularityCalibrationScore, averageBoundaryDistance, selectedSpCount, selectedBorderingSpCount
}
