package labeling_calibration_score

import (
	"image"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/weighted_f1_score"
)

func init() {
	parameterInfo := map[string]superpixel_metrics.SpMetricParameterInfo{
		"max_islands": {
			Kind:        superpixel_metrics.SpMetricParameterKindInt,
			Title:       "Max Islands",
			Description: "Maximum number of islands to consider for density and regularity score calculations. If set to 0, all islands will be considered.",
			Default:     5,
			Min:         0,
			Max:         math.MaxInt,
		},
	}
	superpixel_metrics.RegisterSpMetric(
		"labeling_calibration_score",
		"Labeling Calibration Score",
		"Computes the labeling calibration score as a product of density, weighted F1, and regularity scores.",
		1.0,
		Run,
		parameterInfo,
	)
}

func Run(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]any) (score float64, err error) {
	maxIslands := uint16(params["max_islands"].(int))

	densityScore, _ := density_score.Run(result, groundTruth, maxIslands)
	weightedF1Score, _ := weighted_f1_score.Run(result, groundTruth)
	regularityScore, _ := regularity_score.Run(result, groundTruth, maxIslands)

	return densityScore * weightedF1Score * regularityScore, nil
}
