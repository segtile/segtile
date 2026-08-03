package outer_boundary_pixel_count

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	superpixel_metrics.RegisterSpMetric(
		"outer_boundary_pixel_count",
		"Outer Boundary Pixel Count",
		"Returns the number of outer boundary pixels of the selection.",
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
	maskFromSelection := result.SelectionToMask(selection)
	boundaryLength := len(superpixel_utils.BoundaryPoints(maskFromSelection, true))
	return float64(boundaryLength), nil
}
