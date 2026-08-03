package total_boundary_pixel_count

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	superpixel_metrics.RegisterSpMetric(
		"total_boundary_pixel_count",
		"Total Boundary Pixel Count",
		"Returns the total number of boundary pixels of the selection.",
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

	maskWithLabels := image.NewGray16(result.LabelMap.Bounds())
	for y := 0; y < result.LabelMap.Bounds().Dy(); y++ {
		for x := 0; x < result.LabelMap.Bounds().Dx(); x++ {
			if maskFromSelection.Gray16At(x, y).Y > 0 {
				maskWithLabels.SetGray16(x, y, result.LabelMap.Gray16At(x, y))
			}
		}
	}

	boundaryLength := len(superpixel_utils.BoundaryPoints(maskWithLabels, true))

	return float64(boundaryLength), nil
}
