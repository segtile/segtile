package scale_adaptive

/*
Scale-adaptive superpixel algorithm for medical images, based on:

  Sun, L., Ma, D. and Zhou, Y. (2022) Scale-adaptive superpixels for medical images.
  Quantitative Biology, 10(3): 264-275. https://doi.org/10.15302/J-QB-021-0327

Superpixels are grown sequentially from seed pixels using breadth-first search and a
greedy shortest-path strategy. A path-based distance measure (Eq. 4 in the paper)
accumulates colour differences along the path from the seed, weighted by a decay
factor β. Growth terminates when the path distance exceeds threshold T; boundary
pixels' unassigned neighbours seed the next superpixel. A new superpixel can claim
a pixel from a completed superpixel when its path distance is shorter, which
encourages tight boundary adherence.

Parameters
  T (Threshold): path-distance threshold – larger values produce fewer, bigger superpixels
  β (Beta):      colour-distance decay factor in [0,1] – recommended range [0.8, 1.0]
*/

import (
	"container/heap"
	"errors"
	"image"
	"image/color"
	"math"

	"github.com/lucasb-eyer/go-colorful"

	"github.com/struffel/annotation-tool/superpixel"
)

// ---------------------------------------------------------------------------
// Priority queue (min-heap on path distance)
// ---------------------------------------------------------------------------

type pqItem struct {
	pixelIdx int
	distance float64
	index    int // maintained by heap.Interface
}

type priorityQueue []*pqItem

func (pq priorityQueue) Len() int           { return len(pq) }
func (pq priorityQueue) Less(i, j int) bool { return pq[i].distance < pq[j].distance }
func (pq priorityQueue) Swap(i, j int) {
	pq[i], pq[j] = pq[j], pq[i]
	pq[i].index = i
	pq[j].index = j
}
func (pq *priorityQueue) Push(x any) {
	item := x.(*pqItem)
	item.index = len(*pq)
	*pq = append(*pq, item)
}
func (pq *priorityQueue) Pop() any {
	old := *pq
	n := len(old)
	item := old[n-1]
	old[n-1] = nil
	*pq = old[:n-1]
	return item
}

// ---------------------------------------------------------------------------
// Per-superpixel running colour average
// ---------------------------------------------------------------------------

type spState struct {
	sumL, sumA, sumB float64
	count            int
}

func (s *spState) avg() (l, a, b float64) {
	c := float64(s.count)
	return s.sumL / c, s.sumA / c, s.sumB / c
}

func (s *spState) add(lab [3]float64) {
	s.sumL += lab[0]
	s.sumA += lab[1]
	s.sumB += lab[2]
	s.count++
}

func (s *spState) remove(lab [3]float64) {
	s.sumL -= lab[0]
	s.sumA -= lab[1]
	s.sumB -= lab[2]
	s.count--
}

// ---------------------------------------------------------------------------
// SpScaleAdaptive
// ---------------------------------------------------------------------------

type SpScaleAdaptive struct {
	superpixel.SpAlgorithmData

	// Parameters
	threshold float64 // T – path-distance threshold
	beta      float64 // β – colour-distance decay factor

	// Pre-computed LAB cache indexed by y*width+x
	labCache [][3]float64
}

func (sp *SpScaleAdaptive) SetRecommendedParameters() {
	sp.threshold = 200
	sp.beta = 0.85
}

func (sp *SpScaleAdaptive) GetSupportedParameters() []superpixel.SpParameter {
	return []superpixel.SpParameter{
		superpixel.Threshold,
		superpixel.Beta,
	}
}

func (sp *SpScaleAdaptive) GetParameter(parameter superpixel.SpParameter) (float64, error) {
	switch parameter {
	case superpixel.Threshold:
		return sp.threshold, nil
	case superpixel.Beta:
		return sp.beta, nil
	default:
		return 0, errors.New("ScaleAdaptive does not have this parameter.")
	}
}

func (sp *SpScaleAdaptive) SetParameter(parameter superpixel.SpParameter, value float64) error {
	switch parameter {
	case superpixel.Threshold:
		if value <= 0 {
			return errors.New("Threshold must be positive.")
		}
		sp.threshold = value
		return nil
	case superpixel.Beta:
		if value < 0 || value > 1 {
			return errors.New("Beta must be in [0, 1].")
		}
		sp.beta = value
		return nil
	default:
		return errors.New("ScaleAdaptive does not support this parameter.")
	}
}

func (sp *SpScaleAdaptive) SetImage(img image.Image) {
	sp.Img = img
	sp.Result = superpixel.SpResult{}
	sp.labCache = nil
	sp.buildLabCache()
}

func (sp *SpScaleAdaptive) buildLabCache() {
	width := sp.Img.Bounds().Dx()
	height := sp.Img.Bounds().Dy()
	sp.labCache = make([][3]float64, width*height)
	for y := range height {
		for x := range width {
			c, _ := colorful.MakeColor(sp.Img.At(x, y))
			l, a, b := c.Lab()
			// go-colorful returns L in [0,1]; scale to match SLIC convention.
			l *= 100
			a *= 100
			b *= 100
			sp.labCache[y*width+x] = [3]float64{l, a, b}
		}
	}
}

func colourDist(lab [3]float64, avgL, avgA, avgB float64) float64 {
	dl := lab[0] - avgL
	da := lab[1] - avgA
	db := lab[2] - avgB
	return math.Sqrt(dl*dl + da*da + db*db)
}

// Run executes the scale-adaptive superpixel algorithm (Algorithms 1 & 2 in the paper).
func (sp *SpScaleAdaptive) Run() error {
	if sp.Img == nil {
		return errors.New("Image not set.")
	}

	width := sp.Img.Bounds().Dx()
	height := sp.Img.Bounds().Dy()
	total := width * height

	// Working state --------------------------------------------------------
	labelMap := make([]int, total)
	for i := range labelMap {
		labelMap[i] = -1
	}
	distMap := make([]float64, total)
	for i := range distMap {
		distMap[i] = math.Inf(1)
	}

	// 4-connected offsets
	dx4 := [4]int{-1, 0, 1, 0}
	dy4 := [4]int{0, -1, 0, 1}

	// Seed queue (FIFO). First seed: centre pixel (as stated in the paper).
	centerSeed := (height/2)*width + (width / 2)
	seeds := []int{centerSeed}
	inSeedSet := make([]bool, total)
	inSeedSet[centerSeed] = true

	// Running colour averages, one entry per superpixel.
	var spStates []spState

	label := 0

	// Main loop: grow one superpixel per iteration --------------------------
	for len(seeds) > 0 {
		// Dequeue seed
		seedIdx := seeds[0]
		seeds = seeds[1:]

		// Skip if claimed by an earlier superpixel.
		if labelMap[seedIdx] != -1 {
			continue
		}

		// Initialise new superpixel.
		seedLAB := sp.labCache[seedIdx]
		spStates = append(spStates, spState{
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
		pq := &priorityQueue{}
		heap.Init(pq)
		heap.Push(pq, &pqItem{pixelIdx: seedIdx, distance: 0})

		// Grow the superpixel until the queue is empty.
		for pq.Len() > 0 {
			popped := heap.Pop(pq).(*pqItem)
			p := popped.pixelIdx
			dp := popped.distance

			// Stale check: the pixel may have been stolen after it was pushed.
			// (Only possible from a previous superpixel stealing it; within the
			// current superpixel's growth a pixel is never re-queued.)
			if labelMap[p] != currentLabel {
				continue
			}

			px := p % width
			py := p / width

			// Compute current superpixel average colour.
			avgL, avgA, avgB := spStates[currentLabel].avg()

			if dp < sp.threshold {
				// --- Interior pixel: try to expand into all 4-connected neighbours ---
				for n := range 4 {
					nx := px + dx4[n]
					ny := py + dy4[n]
					if nx < 0 || ny < 0 || nx >= width || ny >= height {
						continue
					}
					nbIdx := ny*width + nx
					nbLAB := sp.labCache[nbIdx]
					newDist := dp*sp.beta + colourDist(nbLAB, avgL, avgA, avgB)

					switch {
					case labelMap[nbIdx] == -1:
						// Condition 1: unassigned – claim immediately.
						labelMap[nbIdx] = currentLabel
						distMap[nbIdx] = newDist
						spStates[currentLabel].add(nbLAB)
						heap.Push(pq, &pqItem{pixelIdx: nbIdx, distance: newDist})

					case labelMap[nbIdx] != currentLabel && newDist < distMap[nbIdx]:
						// Condition 2: belongs to a completed superpixel but our
						// path is shorter – steal the pixel.
						oldLabel := labelMap[nbIdx]
						spStates[oldLabel].remove(nbLAB)

						labelMap[nbIdx] = currentLabel
						distMap[nbIdx] = newDist
						spStates[currentLabel].add(nbLAB)
						heap.Push(pq, &pqItem{pixelIdx: nbIdx, distance: newDist})
					}
				}
			} else {
				// --- Boundary pixel: add unassigned neighbours as future seeds ---
				for n := range 4 {
					nx := px + dx4[n]
					ny := py + dy4[n]
					if nx < 0 || ny < 0 || nx >= width || ny >= height {
						continue
					}
					nbIdx := ny*width + nx
					if labelMap[nbIdx] == -1 && !inSeedSet[nbIdx] {
						seeds = append(seeds, nbIdx)
						inSeedSet[nbIdx] = true
					}
				}
			}
		}
	}

	// Build result ---------------------------------------------------------
	resultLabelMap := image.NewGray16(image.Rect(0, 0, width, height))
	for y := range height {
		for x := range width {
			lbl := labelMap[y*width+x]
			if lbl < 0 {
				lbl = 0
			}
			resultLabelMap.SetGray16(x, y, color.Gray16{Y: uint16(lbl)})
		}
	}
	sp.Result = superpixel.SpResult{
		LabelMap: resultLabelMap,
		SpCount:  label,
	}

	return nil
}
