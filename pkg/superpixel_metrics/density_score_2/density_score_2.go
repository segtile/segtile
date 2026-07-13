package density_score_2

import (
	"image"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{
		{
			Id:          "max_islands",
			Kind:        superpixel_metrics.SpMetricParameterKindInt,
			Title:       "Max Islands",
			Description: "Maximum number of islands to consider for the density score calculation. If set to 0, all islands will be considered.",
			Default:     5,
			Min:         0,
			Max:         math.MaxInt,
		},
	}
	superpixel_metrics.RegisterSpMetric(
		"density_score_2",
		"Density Score",
		"Computes the density score of the superpixel segmentation.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth, uint16(params["max_islands"]))
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16, maxIslands uint16) (densityCalibrationScore float64, err error) {

	labelMapBoundaries := superpixel_utils.BoundaryPoints(result.LabelMap, true)

	selectionFromCurrentIsland := result.SelectFromMask(currentGroundTruthIsland, 0.5)
	currentApproximatedMask := result.SelectionToMask(selectionFromCurrentIsland)

	for _, boundaryPoint := range labelMapBoundaries {

	}

	// For every boundary point:

	// Is it in selection?

	// Does it have a neighbor pixel not in the selection?

	// Boundary with no neighboor outside selection -> Inside Boundary
	// Boudary with neighbour outside selection -> Outside Boundary
	return 0, nil
}
