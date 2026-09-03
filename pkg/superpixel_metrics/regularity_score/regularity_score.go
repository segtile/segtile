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

	// Return early if there is nothing in the image
	if groundTruthIslandCount == 0 {
		return 0.0, nil
	}

	// Build metrics
	superpixelFillRatio := fillRatioOfOwnConvexHullPerSuperpixel(result.LabelMap, result.SpCount)
	islandFillRatio := fillRatioOfOwnConvexHullPerIsland(groundTruthIslands, groundTruthIslandCount)

	return math.Min(islandFillRatio, superpixelFillRatio) / math.Max(islandFillRatio, superpixelFillRatio), nil

}

func fillRatioOfOwnConvexHullPerIsland(islandMap *image.Gray16, islandCount uint16) float64 {
	histogram := superpixel_utils.Histogram(islandMap)

	boundaryPoints := superpixel_utils.BoundaryPointsPerValue(islandMap)
	var totalFillRatio float64
	for i := uint16(1); i <= islandCount; i++ {
		if histogram[i] == 0 {
			continue
		}
		hullArea := superpixel_utils.PolygonArea(superpixel_utils.ConvexHull(boundaryPoints[i]))
		if hullArea > 0 {
			totalFillRatio += float64(histogram[i]) / hullArea
		} else {
			totalFillRatio += 1.0
		}
	}
	return totalFillRatio / float64(islandCount)
}

func fillRatioOfOwnConvexHullPerSuperpixel(labelMap *image.Gray16, spCount int) float64 {
	histogram := superpixel_utils.Histogram(labelMap)

	boundaryPoints := superpixel_utils.BoundaryPointsPerValue(labelMap)
	var totalFillRatio float64
	for i := 0; i < spCount; i++ {
		if histogram[i] == 0 {
			continue
		}
		hullArea := superpixel_utils.PolygonArea(superpixel_utils.ConvexHull(boundaryPoints[uint16(i)]))
		if hullArea > 0 {
			totalFillRatio += float64(histogram[i]) / hullArea
		} else {
			totalFillRatio += 1.0
		}
	}
	return totalFillRatio / float64(spCount)
}
