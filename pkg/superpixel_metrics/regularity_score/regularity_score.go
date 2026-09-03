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

func Run(result superpixel.SpResult, groundTruth *image.Gray16) (regularityScore float64, err error) {

	// Check that images are valid
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Approximate gt using superpixels
	selection := result.SelectFromMask(groundTruth, 0.5)

	// Count SP used to approximate GT
	usedSuperpixelCount := 0
	for i := 0; i < len(selection); i++ {
		if selection[i] {
			usedSuperpixelCount++
		}
	}

	// Build selection mask
	selectionMask := result.SelectionToMask(selection)

	// Build GT distance map
	gtDistance := superpixel_utils.DistanceToNextBoundaryPixel(groundTruth)

	// Determine average distance to border in GT selection
	averageDistance := 0.0
	count := 0
	for y := 0; y < groundTruth.Bounds().Dy(); y++ {
		for x := 0; x < groundTruth.Bounds().Dx(); x++ {
			spId := result.LabelMap.Gray16At(x, y).Y
			if selection[spId] {
				averageDistance += float64(gtDistance.Gray16At(x, y).Y)
				count++
			}
		}
	}
	if count > 0 {
		averageDistance /= float64(count)
	}

	// Blur approx. mask using determined radius
	blurredSelectionMask := superpixel_utils.SmoothMask(selectionMask, 1+int(averageDistance/2.0))

	// Blur GT mask using determined radius
	blurredGtMask := superpixel_utils.SmoothMask(groundTruth, 1+int(averageDistance/2.0))

	// Measure difference between blurred and original masks
	iouSelection := IntersectionOverUnion(selectionMask, blurredSelectionMask)
	iouGt := IntersectionOverUnion(groundTruth, blurredGtMask)

	if iouSelection == 0 || iouGt == 0 {
		return 0, nil
	}

	return math.Min((iouSelection / iouGt), (iouGt / iouSelection)), nil

}

func IntersectionOverUnion(mask1, mask2 *image.Gray16) (iou float64) {
	intersection := 0
	union := 0
	for y := 0; y < mask1.Bounds().Dy(); y++ {
		for x := 0; x < mask1.Bounds().Dx(); x++ {
			p1 := mask1.Gray16At(x, y)
			p2 := mask2.Gray16At(x, y)
			if p1.Y > 0 && p2.Y > 0 {
				intersection++
			}
			if p1.Y > 0 || p2.Y > 0 {
				union++
			}
		}
	}
	if union == 0 {
		return 0.0
	}
	return float64(intersection) / float64(union)
}
