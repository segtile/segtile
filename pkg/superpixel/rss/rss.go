package rss

/*

Rooted Spanning Superpixels implementation based on:

@article{chai2020rooted,
  title={Rooted Spanning Superpixels},
  author={Chai, Dengfeng},
  journal={International Journal of Computer Vision},
  year={2020}
}

*/

import (
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/emirpasic/gods/v2/queues/priorityqueue"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameters := []superpixel.SpParameterInfo{
		{
			Id:          "count",
			Title:       "Superpixel Count",
			Description: "The target number of superpixels to generate.",
			Kind:        superpixel.SpParameterKindInt,
			Default:     500,
			Min:         2,
			Max:         32768,
		},
		{
			Id:          "compactness",
			Title:       "Compactness",
			Description: "The normalized spatial weight. Higher values produce more compact superpixels; values from 0 to 5 are a useful starting range.",
			Kind:        superpixel.SpParameterKindFloat,
			Default:     3,
			Min:         0,
			Max:         512,
		},
	}
	runFromMap := func(img image.Image, params map[string]float64) (superpixel.SpResult, error) {
		return Run(img, int(params["count"]), params["compactness"])
	}
	superpixel.RegisterSpAlgorithm("rss", "RSS", "Rooted Spanning Superpixels (Chai, 2020), using a maximum-difference path cost.", runFromMap, parameters)
}

type queueItem struct {
	pixelIndex int
	label      int
	cost       float64
	order      uint64
}

func compareQueueItems(a, b queueItem) int {
	switch {
	case a.cost < b.cost:
		return -1
	case a.cost > b.cost:
		return 1
	case a.order < b.order:
		return -1
	case a.order > b.order:
		return 1
	default:
		return 0
	}
}

// Run segments img into rooted spanning superpixels. Compactness weights
// normalized spatial distance against the 8-bit RGB maximum-difference cost.
func Run(img image.Image, count int, compactness float64) (superpixel.SpResult, error) {
	if img == nil {
		return superpixel.SpResult{}, errors.New("Image is nil.")
	}
	if count < 2 {
		return superpixel.SpResult{}, errors.New("Superpixel count must be at least 2.")
	}
	if compactness < 0 || math.IsNaN(compactness) || math.IsInf(compactness, 0) {
		return superpixel.SpResult{}, errors.New("Compactness must be a finite, non-negative value.")
	}

	imageBounds := img.Bounds()
	width, height := imageBounds.Dx(), imageBounds.Dy()
	pixelCount := width * height
	if pixelCount == 0 {
		return superpixel.SpResult{}, errors.New("Image has zero dimensions.")
	}
	if count > pixelCount {
		return superpixel.SpResult{}, errors.New("Superpixel count cannot exceed the number of pixels.")
	}

	seeds, gridInterval := superpixel_utils.RegularGridPoints(width, height, count)
	if len(seeds) == 0 {
		seeds = []image.Point{{X: width / 2, Y: height / 2}}
	}
	if len(seeds) > math.MaxUint16 {
		return superpixel.SpResult{}, errors.New("Seed grid exceeds the maximum number of label values.")
	}
	if gridInterval < 1 {
		gridInterval = 1
	}

	features := make([][3]uint8, pixelCount)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := color.NRGBAModel.Convert(img.At(imageBounds.Min.X+x, imageBounds.Min.Y+y)).(color.NRGBA)
			features[y*width+x] = [3]uint8{pixel.R, pixel.G, pixel.B}
		}
	}

	labels := make([]int, pixelCount)
	bestCosts := make([]float64, pixelCount)
	lockedSeeds := make([]bool, pixelCount)
	visited := make([]bool, pixelCount)
	for index := range labels {
		labels[index] = -1
		bestCosts[index] = math.Inf(1)
	}

	queue := priorityqueue.NewWith(compareQueueItems)
	var queueOrder uint64
	for label, seed := range seeds {
		seedIndex := seed.Y*width + seed.X
		labels[seedIndex] = label
		bestCosts[seedIndex] = 0
		lockedSeeds[seedIndex] = true
		queue.Enqueue(queueItem{pixelIndex: seedIndex, label: label, cost: 0, order: queueOrder})
		queueOrder++
	}

	imageRect := image.Rect(0, 0, width, height)
	for !queue.Empty() {
		item, _ := queue.Dequeue()
		if visited[item.pixelIndex] || labels[item.pixelIndex] != item.label || bestCosts[item.pixelIndex] != item.cost {
			continue
		}
		visited[item.pixelIndex] = true

		seed := seeds[item.label]
		seedIndex := seed.Y*width + seed.X
		seedColor := features[seedIndex]
		point := image.Point{X: item.pixelIndex % width, Y: item.pixelIndex / width}
		for _, neighbor := range superpixel_utils.PixelNeighbours(point, imageRect, true, false) {
			neighborIndex := neighbor.Y*width + neighbor.X
			if visited[neighborIndex] || lockedSeeds[neighborIndex] {
				continue
			}

			neighborColor := features[neighborIndex]
			colorDifference := maxChannelDifference(seedColor, neighborColor)
			spatialDifference := compactness * math.Max(
				math.Abs(float64(seed.X-neighbor.X)),
				math.Abs(float64(seed.Y-neighbor.Y)),
			) / float64(gridInterval)
			candidateCost := math.Max(item.cost, math.Max(colorDifference, spatialDifference))
			if candidateCost < bestCosts[neighborIndex] {
				bestCosts[neighborIndex] = candidateCost
				labels[neighborIndex] = item.label
				queue.Enqueue(queueItem{
					pixelIndex: neighborIndex,
					label:      item.label,
					cost:       candidateCost,
					order:      queueOrder,
				})
				queueOrder++
			}
		}
	}

	labelMap := image.NewGray16(image.Rect(0, 0, width, height))
	for pixelIndex, label := range labels {
		if label < 0 {
			return superpixel.SpResult{}, errors.New("Rooted spanning growth did not cover every pixel.")
		}
		labelMap.SetGray16(pixelIndex%width, pixelIndex/width, color.Gray16{Y: uint16(label)})
	}
	return superpixel.NewSpResult(labelMap, len(seeds), true), nil
}

func maxChannelDifference(a, b [3]uint8) float64 {
	difference := int(a[0]) - int(b[0])
	if difference < 0 {
		difference = -difference
	}
	for channel := 1; channel < len(a); channel++ {
		channelDifference := int(a[channel]) - int(b[channel])
		if channelDifference < 0 {
			channelDifference = -channelDifference
		}
		if channelDifference > difference {
			difference = channelDifference
		}
	}
	return float64(difference)
}
