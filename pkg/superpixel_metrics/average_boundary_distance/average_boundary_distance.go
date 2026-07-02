package average_boundary_distance

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameterInfo := map[string]superpixel_metrics.SpMetricParameterInfo{}
	superpixel_metrics.RegisterSpMetric(
		"average_boundary_distance",
		"Average Boundary Distance",
		"Computes the average distance from each boundary pixel in the ground truth mask to the nearest boundary pixel in the superpixel segmentation mask.",
		0.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]any) (score float64, err error) {
	return Run(result, groundTruth)
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16) (score float64, err error) {

	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	cumulativeBoundaryDistance := 0.0

	// Find all boundary pixels in the ground truth segmentation
	boundariesGroundTruth := superpixel_utils.BoundaryPoints(groundTruth, false)
	if len(boundariesGroundTruth) == 0 {
		return 0, fmt.Errorf("No boundary pixels found in ground truth segmentation")
	}

	// Create the boundary distance map for the SP segmentation
	boundaryDistanceMap := superpixel_utils.DistanceToNextBoundaryPixel(result.LabelMap)

	// For every boundary in the grount truth segmentation, get the distance to the nearest boundary in the SP segmentation and add it to the average
	for _, p := range boundariesGroundTruth {
		distance := boundaryDistanceMap.Gray16At(p.X, p.Y).Y
		cumulativeBoundaryDistance += float64(distance)
	}

	// Divide the cumulative boundary distance by the number of boundary pixels in the ground truth segmentation to get the average
	averageBoundaryDistance := cumulativeBoundaryDistance / float64(len(boundariesGroundTruth))

	return averageBoundaryDistance, nil

}
