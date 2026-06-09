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
	selectedBorderingSpCount = 0
	selectedSpCount = 0

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

		currentGroundTruthIsland := SelectIsland(groundTruthIslands, islandLabel)
		currentGroundTruthIslandArea := Area(currentGroundTruthIsland)

		selectionFromCurrentIsland := spResult.SelectFromMask(currentGroundTruthIsland, 0.5)
		currentApproximatedMask := spResult.SelectionToMask(selectionFromCurrentIsland)
		currentApproximatedMaskArea := Area(currentApproximatedMask)

		currentSelectedSpCount, currentSelectedBorderingSpCount := CountBorderingSuperpixels(spResult.LabelMap, selectionFromCurrentIsland)
		selectedSpCount += currentSelectedSpCount
		selectedBorderingSpCount += currentSelectedBorderingSpCount

		currentBorderingSpSelection := SelectBorderingSuperpixels(spResult.LabelMap, selectionFromCurrentIsland)
		currentBorderingSpMask := spResult.SelectionToMask(currentBorderingSpSelection)
		currentBorderingSpMaskArea := Area(currentBorderingSpMask)

		// F1 Score
		currentF1Score, _ := F1Score(currentGroundTruthIsland, currentApproximatedMask)
		f1Score += currentF1Score

		// Density Calibration Score
		if currentSelectedBorderingSpCount > 0 {
			borderPortionOfSelection := float64(currentBorderingSpMaskArea) / float64(currentApproximatedMaskArea)
			desiredBorderPortionOfSelection := math.Pow(1-(float64(currentGroundTruthIslandArea)/float64(groundTruthMaskArea)), 2)
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
	selectedSpCount /= len(islandLabels)
	selectedBorderingSpCount /= len(islandLabels)

	// Final Labeling Calibration Score
	labelingCalibrationScore = f1Score * f1Score * densityCalibrationScore * regularityCalibrationScore

	return labelingCalibrationScore, f1Score, densityCalibrationScore, regularityCalibrationScore, averageBoundaryDistance, selectedSpCount, selectedBorderingSpCount
}
