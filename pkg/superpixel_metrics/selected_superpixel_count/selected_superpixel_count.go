package selected_superpixel_count

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	superpixel_metrics.RegisterSpMetric(
		"selected_superpixel_count",
		"Selected Superpixel Count",
		"Returns the number of superpixels selected by the ground truth mask approximation.",
		0.0,
		Run,
		[]superpixel_metrics.SpMetricParameterInfo{},
	)
}

func Run(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("ground truth and superpixel label map must have the same dimensions")
	}

	selection := result.SelectFromMask(groundTruth, 0.5)
	selectedSpCount, _ := superpixel_utils.CountBorderingSuperpixels(result.LabelMap, selection)
	return float64(selectedSpCount), nil
}
