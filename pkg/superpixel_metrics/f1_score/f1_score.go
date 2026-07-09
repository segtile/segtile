package f1_score

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{}
	superpixel_metrics.RegisterSpMetric(
		"f1_score",
		"F1 Score",
		"Computes the F1 score of the superpixel segmentation against the ground truth mask.",
		1.0,
		Run,
		parameterInfo,
	)
}

func Run(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {

	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Approximate the ground truth mask using the superpixel segmentation
	selectionFromGroundTruth := result.SelectFromMask(groundTruth, 0.5)
	maskApproximated := result.SelectionToMask(selectionFromGroundTruth)

	// Tally up the F1 score
	var intersection, union int
	for y := 0; y < groundTruth.Bounds().Dy(); y++ {
		for x := 0; x < groundTruth.Bounds().Dx(); x++ {
			pOriginal := groundTruth.Gray16At(x, y)
			pApproximated := maskApproximated.Gray16At(x, y)
			if pOriginal.Y > 0 && pApproximated.Y > 0 {
				intersection++
				union += 2
			} else if pOriginal.Y > 0 || pApproximated.Y > 0 {
				union += 1
			}
		}
	}

	if union == 0 {
		return 0, nil
	}

	return float64(2*intersection) / float64(union), nil
}
