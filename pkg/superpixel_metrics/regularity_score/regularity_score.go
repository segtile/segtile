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

	islandFillRatios := fillRatioOfOwnBoundingBoxPerIsland(groundTruthIslands, groundTruthIslandCount)

	// Evaluate islands
	for i := uint16(1); i <= groundTruthIslandCount; i++ {
		// Only consider the superpixels that approximate this specific island, not the whole image.
		currentGroundTruthIsland := superpixel_utils.SelectIsland(groundTruthIslands, i)
		currentSelection := result.SelectFromMask(currentGroundTruthIsland, 0.5)
		currentSuperpixelFillRatio := fillRatioOfOwnBoundingBoxPerSuperpixel(result.LabelMap, result.SpCount, currentSelection)

		// Regularity Calibration Score
		groundTruthFillRatio := islandFillRatios[i]
		if groundTruthFillRatio > 0.0 || currentSuperpixelFillRatio > 0.0 {
			cumulativeRegularityCalibrationScore += math.Min(groundTruthFillRatio, math.Sqrt(currentSuperpixelFillRatio)) / math.Max(groundTruthFillRatio, math.Sqrt(currentSuperpixelFillRatio))
		}
	}

	// Calculate the final metric
	regularityCalibrationScore = cumulativeRegularityCalibrationScore / float64(groundTruthIslandCount)

	return regularityCalibrationScore, nil

}

// fillRatioOfOwnBoundingBoxPerIsland computes, in a single pass over islandMap, the fill ratio of
// every ground truth island (label 0 is background and is ignored). The returned slice is indexed
// by island label, so index 0 is always 0.
func fillRatioOfOwnBoundingBoxPerIsland(islandMap *image.Gray16, islandCount uint16) []float64 {
	islandPixelCount := make([]int, islandCount+1)
	islandBoundingBox := make([]superpixel_utils.BoundingBox, islandCount+1)
	bounds := islandMap.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			islandLabel := islandMap.Gray16At(x, y).Y
			if islandLabel == 0 {
				continue
			}
			islandPixelCount[islandLabel]++
			islandBoundingBox[islandLabel].AddPoint(x, y)
		}
	}

	fillRatios := make([]float64, islandCount+1)
	for i := uint16(1); i <= islandCount; i++ {
		if islandPixelCount[i] > 0 {
			fillRatios[i] = float64(islandPixelCount[i]) / float64(islandBoundingBox[i].Area())
		}
	}
	return fillRatios
}

// fillRatioOfOwnBoundingBoxPerSuperpixel computes, in a single pass over labelMap, the average fill
// ratio of the superpixels marked in selection. Superpixels not in selection are ignored entirely,
// both when accumulating pixel/bounding box data and when averaging the result.
func fillRatioOfOwnBoundingBoxPerSuperpixel(labelMap *image.Gray16, spCount int, selection []bool) float64 {
	spPixelCount := make([]int, spCount)
	spBoundingBox := make([]superpixel_utils.BoundingBox, spCount)
	bounds := labelMap.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			spLabel := labelMap.Gray16At(x, y).Y
			if !selection[spLabel] {
				continue
			}
			spPixelCount[spLabel]++
			spBoundingBox[spLabel].AddPoint(x, y)
		}
	}

	var totalFillRatio float64
	var selectedCount int
	for i := 0; i < spCount; i++ {
		if !selection[i] {
			continue
		}
		selectedCount++
		if spPixelCount[i] > 0 {
			totalFillRatio += float64(spPixelCount[i]) / float64(spBoundingBox[i].Area())
		}
	}
	if selectedCount == 0 {
		return 0.0
	}
	return totalFillRatio / float64(selectedCount)
}
