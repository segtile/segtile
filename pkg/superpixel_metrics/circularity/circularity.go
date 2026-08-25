package circularity

import (
	"image"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{}
	superpixel_metrics.RegisterSpMetric(
		"circularity",
		"Circularity",
		"Computes the circularity of the superpixel segmentation.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result)
}

func Run(result superpixel.SpResult) (circularity float64, err error) {

	areaPerSp := make(map[uint16]float64)
	boundaryLengthPerSp := make(map[uint16]float64)

	for y := 0; y < result.LabelMap.Bounds().Dy(); y++ {
		for x := 0; x < result.LabelMap.Bounds().Dx(); x++ {
			spId := result.LabelMap.Gray16At(x, y).Y

			// Tally up area
			areaPerSp[spId]++

			// Count bounding pixels per SP
			for _, neighbour := range superpixel_utils.PixelNeighbours(image.Point{X: x, Y: y}, result.LabelMap.Bounds(), false) {
				neighbourSpId := result.LabelMap.Gray16At(neighbour.X, neighbour.Y).Y
				if neighbourSpId != spId {
					boundaryLengthPerSp[spId]++
				}

			}
		}
	}

	// Remove data for background superpixel (id 0)
	delete(areaPerSp, 0)
	delete(boundaryLengthPerSp, 0)

	// Compute circularity per superpixel
	for spId := range areaPerSp {
		area := areaPerSp[spId]
		boundaryLength := boundaryLengthPerSp[spId]
		if boundaryLength > 0 {
			circularity += (4 * math.Pi * area) / (boundaryLength * boundaryLength)
		}
	}

	// Average circularity over all superpixels
	if len(areaPerSp) > 0 {
		circularity /= float64(len(areaPerSp))
	}

	return circularity, nil
}
