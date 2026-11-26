package labeling_calibration_score

import (
	"image"

	"github.com/segtile/segtile/pkg/superpixel"
	"github.com/segtile/segtile/pkg/superpixel_map"
	"github.com/segtile/segtile/pkg/superpixel_metrics"
	"github.com/segtile/segtile/pkg/superpixel_metrics/density_score"
	"github.com/segtile/segtile/pkg/superpixel_metrics/distance_weighted_dice_coefficient"
	"github.com/segtile/segtile/pkg/superpixel_metrics/regularity_score"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{}
	superpixel_metrics.RegisterSpMetric(
		"labeling_calibration_score",
		"Labeling Calibration Score",
		"Computes the labeling calibration score as a product of density, F1, and regularity scores.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error) {
	return Run(result, img, groundTruth)
}

func Run(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask) (score float64, err error) {

	densityScore, _ := density_score.Run(result, groundTruth)
	dwdc, _ := distance_weighted_dice_coefficient.Run(result, groundTruth)
	regularityScore, _ := regularity_score.Run(result, groundTruth)

	return densityScore * dwdc * regularityScore, nil
}
