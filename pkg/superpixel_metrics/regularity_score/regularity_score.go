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
	superpixel_metrics.RegisterSpMetric(
		"regularity_score",
		"Regularity Score",
		"Computes the regularity score of the superpixel segmentation.",
		1.0,
		runWithParams,
		nil,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth)
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16) (regularityCalibrationScore float64, err error) {

	// Check that images are valid
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Find islands in GT
	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruth)

	// Prepare metrics
	cumulativeRegularityCalibrationScore := 0.0

	// Return early
	if groundTruthIslandCount == 0 {
		return 0.0, nil
	}

	// The superpixel fill ratio and aspect ratio do not depend on any particular island, so compute
	// them once instead of redundantly recomputing them on every loop iteration.
	superpixelFillRatio := fillRatioOfOwnBoundingBoxPerSuperpixel(result.LabelMap, result.SpCount)
	superpixelAspectRatio := aspectRatioOfBoundingBoxPerSuperpixel(result.LabelMap, result.SpCount)

	// Compute the fill ratio and aspect ratio of every ground truth island in a single pass over
	// the image, instead of re-scanning the whole image (via SelectIsland + fillRatioOfOwnBoundingBox)
	// once per island.
	islandFillRatios := fillRatioOfOwnBoundingBoxPerIsland(groundTruthIslands, groundTruthIslandCount)
	islandAspectRatios := aspectRatioOfBoundingBoxPerIsland(groundTruthIslands, groundTruthIslandCount)

	// Evaluate islands
	for i := uint16(1); i <= groundTruthIslandCount; i++ {
		// Regularity Calibration Score
		groundTruthFillRatio := islandFillRatios[i]
		groundTruthAspectRatio := islandAspectRatios[i]
		if groundTruthFillRatio > 0.0 || superpixelFillRatio > 0.0 {
			currentScore := math.Min(groundTruthFillRatio, superpixelFillRatio) / math.Max(groundTruthFillRatio, superpixelFillRatio)
			currentScore *= math.Min(groundTruthAspectRatio, superpixelAspectRatio) / math.Max(groundTruthAspectRatio, superpixelAspectRatio)
			cumulativeRegularityCalibrationScore += currentScore
		}
	}

	// Calculate the final metric
	regularityCalibrationScore = cumulativeRegularityCalibrationScore / float64(groundTruthIslandCount)

	return regularityCalibrationScore, nil

}
