package f1_score

import (
	"fmt"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
)

type SpMetricF1Score struct {
	superpixel_metrics.SpMetricsData
}

func (e *SpMetricF1Score) SetParameter(parameter superpixel_metrics.SpMetricParameter, value float64) error {
	return fmt.Errorf("No parameters available for F1Score metric.")
}

func (e *SpMetricF1Score) GetParameter(parameter superpixel_metrics.SpMetricParameter) (float64, error) {
	return 0, fmt.Errorf("No parameters available for F1Score metric.")
}

func (e *SpMetricF1Score) GetSupportedParameters() []superpixel_metrics.SpMetricParameter {
	return []superpixel_metrics.SpMetricParameter{}
}

func (e *SpMetricF1Score) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricF1Score) GetScore() (f1Score float64, err error) {

	maskOriginal := e.GetGroundTruthSegmentation()
	selectionFromGroundTruth := e.GetSpResult().SelectFromMask(maskOriginal, 0.5)
	maskApproximated := e.GetSpResult().SelectionToMask(selectionFromGroundTruth)

	if maskOriginal.Bounds() != maskApproximated.Bounds() {
		return 0, fmt.Errorf("Masks must have the same dimensions")
	}
	var intersection, union int
	for y := 0; y < maskOriginal.Bounds().Dy(); y++ {
		for x := 0; x < maskOriginal.Bounds().Dx(); x++ {
			pOriginal := maskOriginal.Gray16At(x, y)
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
