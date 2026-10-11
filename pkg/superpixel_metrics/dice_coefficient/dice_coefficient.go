package dice_coefficient

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{}
	superpixel_metrics.RegisterSpMetric(
		"dice_coefficient",
		"Dice Coefficient",
		"Computes the Dice coefficient of the superpixel segmentation against the ground truth mask.",
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

	// Tally up the Dice score
	var intersection, union int
	for y := 0; y < groundTruth.Bounds().Dy(); y++ {
		for x := 0; x < groundTruth.Bounds().Dx(); x++ {
			pOriginal := groundTruth.ValueAt(x, y)
			pApproximated := maskApproximated.ValueAt(x, y)
			if pOriginal > 0 && pApproximated > 0 {
				intersection++
				union += 2
			} else if pOriginal > 0 || pApproximated > 0 {
				union += 1
			}
		}
	}

	if union == 0 {
		return 0, nil
	}

	return float64(2*intersection) / float64(union), nil
}
