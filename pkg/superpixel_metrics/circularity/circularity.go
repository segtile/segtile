package circularity

import (
	"image"
	"math"

	"github.com/segtile/segtile/pkg/superpixel"
	"github.com/segtile/segtile/pkg/superpixel_map"
	"github.com/segtile/segtile/pkg/superpixel_metrics"
	"github.com/segtile/segtile/pkg/superpixel_utils"
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

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error) {
	return Run(result)
}

func Run(result superpixel.SpResult) (circularity float64, err error) {

	areaPerSp := make(map[uint32]float64)
	boundaryLengthPerSp := make(map[uint32]float64)

	bounds := result.LabelMap.Bounds()

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			spId := result.LabelMap.ValueAt(x, y)

			// Tally up area
			areaPerSp[spId]++

			// Count bounding pixels per SP.
			for _, neighbour := range superpixel_utils.PixelNeighbours(image.Point{X: x, Y: y}, bounds, false, true) {
				if !neighbour.In(bounds) {
					boundaryLengthPerSp[spId]++
					continue
				}
				neighbourSpId := result.LabelMap.ValueAt(neighbour.X, neighbour.Y)
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
			tmp := (4 * math.Pi * area) / (boundaryLength * boundaryLength)
			circularity += tmp
		}
	}

	// Average circularity over all superpixels
	if len(areaPerSp) > 0 {
		circularity /= float64(len(areaPerSp))
	}

	return circularity, nil
}
