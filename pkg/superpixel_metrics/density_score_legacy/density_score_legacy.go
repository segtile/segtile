package density_score_legacy

import (
	"fmt"
	"image"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{
		{
			Id:          "max_islands",
			Kind:        superpixel_metrics.SpMetricParameterKindInt,
			Title:       "Max Islands",
			Description: "Maximum number of islands to consider for the density score calculation. If set to 0, all islands will be considered.",
			Default:     5,
			Min:         0,
			Max:         math.MaxInt,
		},
	}
	superpixel_metrics.RegisterSpMetric(
		"density_score_legacy",
		"Density Score (Legacy)",
		"Computes the density score of the superpixel segmentation using the legacy boundary SP counting method.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth, uint16(params["max_islands"]))
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16, maxIslands uint16) (densityCalibrationScore float64, err error) {

	// Check that images are valid
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Find islands in GT
	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruth)

	// Prepare metrics
	cumulativeDensityCalibrationScore := 0.0
	pixelCount := groundTruth.Bounds().Dx() * groundTruth.Bounds().Dy()

	// Choose the number of islands to evaluate
	if maxIslands <= 0 {
		maxIslands = groundTruthIslandCount
	}
	if maxIslands > groundTruthIslandCount {
		maxIslands = groundTruthIslandCount
	}

	// Return early
	if maxIslands == 0 {
		return 0.0, nil
	}

	// Evaluate islands
	for i := uint16(1); i <= maxIslands; i++ {

		// Make a selection of the current island and get its area
		currentGroundTruthIsland := superpixel_utils.SelectIsland(groundTruthIslands, i)
		currentGroundTruthIslandArea := superpixel_utils.Area(currentGroundTruthIsland)

		// Make a selection of the superpixels that overlap with the current island and get its area
		selectionFromCurrentIsland := result.SelectFromMask(currentGroundTruthIsland, 0.5)
		currentApproximatedMask := result.SelectionToMask(selectionFromCurrentIsland)
		currentApproximatedMaskArea := superpixel_utils.Area(currentApproximatedMask)

		// Count the number of superpixels that are bordering superpixels in the current selection
		_, currentSelectedBorderingSpCount := superpixel_utils.CountBorderingSuperpixels(result.LabelMap, selectionFromCurrentIsland)

		// Get the mask of the bordering superpixels and its area
		currentBorderingSpSelection := superpixel_utils.SelectBorderingSuperpixels(result.LabelMap, selectionFromCurrentIsland)
		currentBorderingSpMask := result.SelectionToMask(currentBorderingSpSelection)
		currentBorderingSpMaskArea := superpixel_utils.Area(currentBorderingSpMask)

		// Density Calibration Score
		if currentSelectedBorderingSpCount > 0 {
			// Get the area portion of the selection that is a bordering superpixel
			borderPortionOfSelection := float64(currentBorderingSpMaskArea) / float64(currentApproximatedMaskArea)

			// Get the desired area portion of the selection that is a bordering superpixel based on the area of the current island and the total pixel count
			desiredBorderPortionOfSelection := math.Pow(1-(float64(currentGroundTruthIslandArea)/float64(pixelCount)), 2)

			// Add the ratio of the actual border portion to the desired border portion to the density calibration score
			cumulativeDensityCalibrationScore += math.Min(borderPortionOfSelection, desiredBorderPortionOfSelection) / math.Max(borderPortionOfSelection, desiredBorderPortionOfSelection)
		}

	}

	// Calculate the final metric
	densityCalibrationScore = cumulativeDensityCalibrationScore / float64(maxIslands)

	return densityCalibrationScore, nil
}
