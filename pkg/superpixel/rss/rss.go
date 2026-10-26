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
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
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
			Description: "The normalized spatial weight. Higher values produce more compact superpixels.",
			Kind:        superpixel.SpParameterKindFloat,
			Default:     3,
			Min:         0,
			Max:         512,
		},
	}
	runFromMap := func(img image.Image, params map[string]float64) (superpixel.SpResult, error) {
		return Run(img, int(params["count"]), params["compactness"])
	}
	superpixel.RegisterSpAlgorithm("rss", "Rooted Spanning Superpixels", "Rooted Spanning Superpixels (Chai, 2020).", runFromMap, parameters)
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

func Run(img image.Image, count int, compactness float64) (superpixel.SpResult, error) {

	// Validate input parameters
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

	// Place starting points
	seeds, gridInterval := superpixel_utils.RegularGridPoints(width, height, count)
	if len(seeds) == 0 {
		seeds = []image.Point{{X: width / 2, Y: height / 2}}
	}
	if uint64(len(seeds)-1) > uint64(^uint32(0)) {
		return superpixel.SpResult{}, errors.New("Seed grid exceeds the maximum uint32 label value.")
	}
	if gridInterval < 1 {
		gridInterval = 1
	}

	// Extract color features from the image
	features := superpixel_map.NewThreeChannelUint8Map(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			pixel := color.NRGBAModel.Convert(img.At(imageBounds.Min.X+x, imageBounds.Min.Y+y)).(color.NRGBA)
			features.SetValueAt(x, y, [3]uint8{pixel.R, pixel.G, pixel.B})
		}
	}

	// Initialize label, cost, and state maps for the superpixel algorithm

	labels := superpixel_map.NewIntMap(width, height)
	bestCosts := superpixel_map.NewFloatMap(width, height)
	lockedSeeds := superpixel_map.NewMask(width, height)
	visited := superpixel_map.NewMask(width, height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			labels.SetValueAt(x, y, -1)
			bestCosts.SetValueAt(x, y, math.Inf(1))
		}
	}

	queue := priorityqueue.NewWith(compareQueueItems)
	var queueOrder uint64
	for label, seed := range seeds {
		seedIndex := seed.Y*width + seed.X
		labels.SetValueAt(seed.X, seed.Y, label)
		bestCosts.SetValueAt(seed.X, seed.Y, 0)
		lockedSeeds.SetValueAt(seed.X, seed.Y, true)
		queue.Enqueue(queueItem{pixelIndex: seedIndex, label: label, cost: 0, order: queueOrder})
		queueOrder++
	}

	// Perform the rooted spanning growth to assign pixels to superpixels
	imageRect := image.Rect(0, 0, width, height)
	for !queue.Empty() {

		// Dequeue the next pixel to process
		item, _ := queue.Dequeue()
		point := image.Point{X: item.pixelIndex % width, Y: item.pixelIndex / width}
		if visited.PositiveAt(point.X, point.Y) ||
			labels.ValueAt(point.X, point.Y) != item.label ||
			bestCosts.ValueAt(point.X, point.Y) != item.cost {
			continue
		}
		visited.SetValueAt(point.X, point.Y, true)

		// Retrieve the seed corresponding to the current label
		seed := seeds[item.label]
		seedColor := features.ValueAt(seed.X, seed.Y)

		// Iterate over the neighboring pixels to update their costs and labels
		for _, neighbor := range superpixel_utils.PixelNeighbours(point, imageRect, true, false) {

			// Compute the index of the neighboring pixel
			neighborIndex := neighbor.Y*width + neighbor.X
			if visited.PositiveAt(neighbor.X, neighbor.Y) || lockedSeeds.PositiveAt(neighbor.X, neighbor.Y) {
				continue
			}

			// Compute the cost of assigning the neighboring pixel to the current superpixel
			neighborColor := features.ValueAt(neighbor.X, neighbor.Y)
			colorDifference := maxChannelDifference(seedColor, neighborColor)
			spatialDifference := compactness * math.Max(
				math.Abs(float64(seed.X-neighbor.X)),
				math.Abs(float64(seed.Y-neighbor.Y)),
			) / float64(gridInterval)
			candidateCost := math.Max(item.cost, math.Max(colorDifference, spatialDifference))
			if candidateCost < bestCosts.ValueAt(neighbor.X, neighbor.Y) {

				// Update the best cost and label for the neighboring pixel
				bestCosts.SetValueAt(neighbor.X, neighbor.Y, candidateCost)
				labels.SetValueAt(neighbor.X, neighbor.Y, item.label)
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

	// Convert labels into the label map for the final superpixel result
	labelMap := superpixel_map.NewLabelMap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			label := labels.ValueAt(x, y)
			if label < 0 {
				return superpixel.SpResult{}, errors.New("Rooted spanning growth did not cover every pixel.")
			}
			labelMap.SetValueAt(x, y, uint32(label))
		}
	}
	return superpixel.NewSpResult(labelMap), nil
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
