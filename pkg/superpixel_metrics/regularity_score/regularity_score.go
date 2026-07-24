package regularity_score

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
			Description: "Maximum number of islands to consider for the regularity score calculation. If set to 0, all islands will be considered.",
			Default:     5,
			Min:         0,
			Max:         math.MaxInt,
		},
	}
	superpixel_metrics.RegisterSpMetric(
		"regularity_score",
		"Regularity Score",
		"Computes the regularity score of the superpixel segmentation.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth, uint16(params["max_islands"]))
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16, maxIslands uint16) (regularityCalibrationScore float64, err error) {

	// Check that images are valid
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Find islands in GT
	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruth)

	// Prepare metrics
	cumulativeRegularityCalibrationScore := 0.0

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
		currentGroundTruthIsland := superpixel_utils.SelectIsland(groundTruthIslands, i)

		// Regularity Calibration Score
		groundTruthFillRatio := fillRatioOfOwnBoundingBox(currentGroundTruthIsland)
		superpixelFillRatio := fillRatioOfOwnBoundingBoxPerSuperpixel(result.LabelMap, result.SpCount)
		groundTruthAspectRatio := aspectRatioOfBoundingBox(currentGroundTruthIsland)
		superpixelAspectRatio := aspectRatioOfBoundingBoxPerSuperpixel(result.LabelMap, result.SpCount)

		if groundTruthFillRatio > 0.0 || superpixelFillRatio > 0.0 {
			currentScore := math.Min(groundTruthFillRatio, superpixelFillRatio) / math.Max(groundTruthFillRatio, superpixelFillRatio)
			currentScore *= math.Min(groundTruthAspectRatio, superpixelAspectRatio) / math.Max(groundTruthAspectRatio, superpixelAspectRatio)
			cumulativeRegularityCalibrationScore += currentScore
		}
	}

	// Calculate the final metric
	regularityCalibrationScore = cumulativeRegularityCalibrationScore / float64(maxIslands)

	return regularityCalibrationScore, nil

}
