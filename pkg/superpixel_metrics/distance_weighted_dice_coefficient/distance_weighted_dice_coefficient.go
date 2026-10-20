package distance_weighted_dice_coefficient

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{}
	superpixel_metrics.RegisterSpMetric(
		"distance_weighted_dice_coefficient",
		"Distance Weighted Dice Coefficient",
		"Computes the Dice coefficient of the superpixel segmentation against the ground truth mask, weighted by distance to the nearest boundary pixel.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth)
}

func Run(result superpixel.SpResult, groundTruth *superpixel_map.Mask) (score float64, err error) {

	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Approximate the ground truth mask using the superpixel segmentation
	selectionFromGroundTruth := result.SelectFromMask(groundTruth, 0.5)
	maskApproximated := result.SelectionToMask(selectionFromGroundTruth)

	// Create the boundary distance map which dictates the weight for the union calculation
	distanceMap := superpixel_utils.DistanceToNextBoundaryPixel(groundTruth)

	// Tally up the weighted F1 score
	var intersectionPixels, sumPixels, distancePenalty, gtPixels int
	var avgBorderDistance float64

	for y := 0; y < groundTruth.Bounds().Dy(); y++ {
		for x := 0; x < groundTruth.Bounds().Dx(); x++ {

			// Read the maps
			gtMaskValue := groundTruth.PositiveAt(x, y)
			approxMaskValue := maskApproximated.PositiveAt(x, y)
			distance := int(distanceMap.ValueAt(x, y))

			// Tally up area and average distance
			if gtMaskValue {
				gtPixels++
				avgBorderDistance += float64(distance)
			}

			// Test for a mismatch
			if gtMaskValue != approxMaskValue {

				// Use the distance to the nearest boundary pixel as a weight for the union
				distancePenalty += distance
				sumPixels += 1

				continue
			}

			// Test for a positive match (Both masks are true)
			if gtMaskValue && approxMaskValue {
				intersectionPixels++
				sumPixels += 2

				continue
			}

		}
	}

	if sumPixels == 0 || gtPixels == 0 {
		return 0, nil
	}

	avgBorderDistance = float64(avgBorderDistance) / float64(gtPixels)
	if avgBorderDistance == 0 {
		distancePenalty = 0
	} else {
		distancePenalty = int(float64(distancePenalty) / avgBorderDistance)
	}

	return float64(2*intersectionPixels) / float64(sumPixels+distancePenalty), nil
}
