package actual_superpixel_count

import (
	"image"

	"github.com/segtile/segtile/pkg/superpixel"
	"github.com/segtile/segtile/pkg/superpixel_map"
	"github.com/segtile/segtile/pkg/superpixel_metrics"
)

func init() {
	superpixel_metrics.RegisterSpMetric(
		"actual_superpixel_count",
		"Actual Superpixel Count",
		"Returns the number of superpixels in the segmentation result.",
		0.0,
		Run,
		[]superpixel_metrics.SpMetricParameterInfo{},
	)
}

func Run(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error) {
	return float64(result.SpCount), nil
}
