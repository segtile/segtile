package achievable_segmentation_accuracy

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{}
	superpixel_metrics.RegisterSpMetric(
		"achievable_segmentation_accuracy",
		"Achievable Segmentation Accuracy",
		"Computes the achievable segmentation accuracy of the superpixel segmentation against the ground truth segmentation.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth)
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16) (score float64, err error) {

	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Tally up the overlap between each superpixel and each ground truth label
	overlapPerSp := make(map[uint16]map[uint16]int)

	bounds := result.LabelMap.Bounds()
	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			spId := result.LabelMap.Gray16At(x, y).Y
			gtLabel := groundTruth.Gray16At(x, y).Y

			if overlapPerSp[spId] == nil {
				overlapPerSp[spId] = make(map[uint16]int)
			}
			overlapPerSp[spId][gtLabel]++
		}
	}

	// Assign each superpixel to the ground truth label it overlaps with the most
	var correctlyLabeledPixels int
	for _, overlapPerLabel := range overlapPerSp {
		var maxOverlap int
		for _, overlap := range overlapPerLabel {
			if overlap > maxOverlap {
				maxOverlap = overlap
			}
		}
		correctlyLabeledPixels += maxOverlap
	}

	totalPixels := bounds.Dx() * bounds.Dy()
	if totalPixels == 0 {
		return 0, nil
	}

	return float64(correctlyLabeledPixels) / float64(totalPixels), nil
}
