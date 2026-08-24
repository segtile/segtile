package scale_adaptive

/*
Scale-adaptive superpixel algorithm for medical images, based on:

@article{sun2022scale,
  title={Scale-adaptive superpixels for medical images},
  author={Sun, Limin and Ma, Dongyang and Zhou, Yuanfeng},
  journal={Quantitative Biology},
  volume={10},
  number={3},
  pages={264--275},
  year={2022},
  publisher={Wiley Online Library}
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
	parameterInfo := []superpixel.SpParameterInfo{
		{
			Id:          "threshold",
			Title:       "Threshold",
			Description: "The path-distance threshold. Larger values produce fewer, bigger superpixels.",
			Kind:        superpixel.SpParameterKindFloat,
			Default:     200,
			Min:         0.0001,
			Max:         10000,
		},
		{
			Id:          "beta",
			Title:       "Beta",
			Description: "The colour-distance decay factor. Recommended range is [0.8, 1.0].",
			Kind:        superpixel.SpParameterKindFloat,
			Default:     0.85,
			Min:         0,
			Max:         1,
		},
	}
	runFromMap := func(img image.Image, params map[string]float64) (superpixel.SpResult, error) {
		return Run(img, params["threshold"], params["beta"])
	}
	superpixel.RegisterSpAlgorithm("scale_adaptive", "Scale-Adaptive", "Scale-adaptive superpixel algorithm for medical images, based on Sun, Ma and Zhou (2022).", runFromMap, parameterInfo)
}

// Priority queue

type scaleAdaptiveQueueItem struct {
	pixelIndex int
	distance   float64
}

func scaleAdaptiveQueueItemComparator(a, b scaleAdaptiveQueueItem) int {
	switch {
	case a.distance < b.distance:
		return -1
	case a.distance > b.distance:
		return 1
	default:
		return 0
	}
}

// Per-superpixel running colour average

type scaleAdaptiveSpState struct {
	sumL, sumA, sumB float64
	count            int
}

func (s *scaleAdaptiveSpState) avg() (l, a, b float64) {
	c := float64(s.count)
	return s.sumL / c, s.sumA / c, s.sumB / c
}

func (s *scaleAdaptiveSpState) add(lab [3]float64) {
	s.sumL += lab[0]
	s.sumA += lab[1]
	s.sumB += lab[2]
	s.count++
}

func (s *scaleAdaptiveSpState) remove(lab [3]float64) {
	s.sumL -= lab[0]
	s.sumA -= lab[1]
	s.sumB -= lab[2]
	s.count--
}

func colourDist(lab [3]float64, avgL, avgA, avgB float64) float64 {
	dl := lab[0] - avgL
	da := lab[1] - avgA
	db := lab[2] - avgB
	return math.Sqrt(dl*dl + da*da + db*db)
}

func Run(img image.Image, threshold float64, beta float64) (superpixel.SpResult, error) {

	// Test inputs

	if img == nil {
		return superpixel.SpResult{}, errors.New("Image is nil.")
	}

	if threshold <= 0 {
		return superpixel.SpResult{}, errors.New("Threshold must be positive.")
	}

	if beta < 0 || beta > 1 {
		return superpixel.SpResult{}, errors.New("Beta must be in [0, 1].")
	}

	// Create helper variables

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pixelCount := width * height
	bounds := img.Bounds()

	var labCache [][3]float64
	superpixel_utils.BuildLabCache(img, &labCache)

	// Working state
	labelMap := make([]int, pixelCount)
	for i := range labelMap {
		labelMap[i] = -1
	}
	distMap := make([]float64, pixelCount)
	for i := range distMap {
		distMap[i] = math.Inf(1)
	}

	// Seed queue with a single center seed.
	centerSeed := (height/2)*width + (width / 2)
	seeds := []int{centerSeed}
	inSeedSet := make([]bool, pixelCount)
	inSeedSet[centerSeed] = true

	// Running colour averages, one entry per superpixel.
	var spStates []scaleAdaptiveSpState

	label := 0

	// Main loop: grow one superpixel per iteration
	for len(seeds) > 0 {
		// Dequeue seed
		seedIdx := seeds[0]
		seeds = seeds[1:]

		// Skip if claimed by an earlier superpixel.
		if labelMap[seedIdx] != -1 {
			continue
		}

		// Initialise new superpixel.
		seedLAB := labCache[seedIdx]
		spStates = append(spStates, scaleAdaptiveSpState{
			sumL:  seedLAB[0],
			sumA:  seedLAB[1],
			sumB:  seedLAB[2],
			count: 1,
		})
		currentLabel := label
		label++

		labelMap[seedIdx] = currentLabel
		distMap[seedIdx] = 0

		// Initialise priority queue with the seed pixel.
		pq := priorityqueue.NewWith(scaleAdaptiveQueueItemComparator)
		pq.Enqueue(scaleAdaptiveQueueItem{pixelIndex: seedIdx, distance: 0})

		// Grow the superpixel until the queue is empty.
		for !pq.Empty() {
			currentItem, _ := pq.Dequeue()
			currentIndex := currentItem.pixelIndex
			currentDistance := currentItem.distance

			// Stale check: the pixel may have been stolen after it was pushed.
			// (Only possible from a previous superpixel stealing it; within the
			// current superpixel's growth a pixel is never re-queued.)
			if labelMap[currentIndex] != currentLabel {
				continue
			}

			// Convert linear index to 2D coordinates.
			currentStartingPoint := image.Point{X: currentIndex % width, Y: currentIndex / width}

			// Compute current superpixel average colour.
			avgL, avgA, avgB := spStates[currentLabel].avg()

			if currentDistance < threshold {
				// Interior pixel: try to expand into all 4-connected neighbours
				for _, nb := range superpixel_utils.PixelNeighbours(currentStartingPoint, bounds, false) {
					nbIdx := nb.Y*width + nb.X
					nbLAB := labCache[nbIdx]
					newDist := currentDistance*beta + colourDist(nbLAB, avgL, avgA, avgB)

					switch {
					case labelMap[nbIdx] == -1:
						// Condition 1: unassigned -> claim immediately.
						labelMap[nbIdx] = currentLabel
						distMap[nbIdx] = newDist
						spStates[currentLabel].add(nbLAB)
						pq.Enqueue(scaleAdaptiveQueueItem{pixelIndex: nbIdx, distance: newDist})

					case labelMap[nbIdx] != currentLabel && newDist < distMap[nbIdx]:
						// Condition 2: belongs to a completed superpixel but our
						// path is shorter -> steal the pixel.
						oldLabel := labelMap[nbIdx]
						spStates[oldLabel].remove(nbLAB)

						labelMap[nbIdx] = currentLabel
						distMap[nbIdx] = newDist
						spStates[currentLabel].add(nbLAB)
						pq.Enqueue(scaleAdaptiveQueueItem{pixelIndex: nbIdx, distance: newDist})
					}
				}
			} else {
				// Boundary pixel: add unassigned neighbours as future seeds
				for _, nb := range superpixel_utils.PixelNeighbours(currentStartingPoint, bounds, false) {
					nbIdx := nb.Y*width + nb.X
					if labelMap[nbIdx] == -1 && !inSeedSet[nbIdx] {
						seeds = append(seeds, nbIdx)
						inSeedSet[nbIdx] = true
					}
				}
			}
		}
	}

	// Build result
	resultLabelMap := image.NewGray16(img.Bounds())
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			lbl := labelMap[y*width+x]
			if lbl < 0 {
				lbl = 0
			}
			resultLabelMap.SetGray16(x, y, color.Gray16{Y: uint16(lbl)})
		}
	}

	return superpixel.NewSpResult(resultLabelMap, 0, true), nil
}
