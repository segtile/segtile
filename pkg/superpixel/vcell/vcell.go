package vcell

/*

This is an implementation of VCells superpixels, based on the original paper:

@article{wang2012vcells,
  title={VCells: Simple and Efficient Superpixels Using Edge-Weighted Centroidal Voronoi Tessellations},
  author={Wang, Jie and Wang, Xiaoqiang},
  journal={IEEE Transactions on Pattern Analysis and Machine Intelligence},
  volume={34},
  number={6},
  pages={1241--1247},
  year={2012},
  publisher={IEEE}
}

*/

import (
	"errors"
	"image"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameterInfo := []superpixel.SpParameterInfo{
		{
			Id:          "count",
			Title:       "Superpixel Count",
			Description: "The number of superpixels to generate.",
			Kind:        superpixel.SpParameterKindInt,
			Default:     500,
			Min:         2,
			Max:         32768,
		},
		{
			Id:          "compactness",
			Title:       "Compactness",
			Description: "The compactness weight (lambda) of the superpixels. If 0, it is automatically computed.",
			Kind:        superpixel.SpParameterKindFloat,
			Default:     10.0,
			Min:         0.1,
			Max:         1000,
		},
	}
	runFromMap := func(img image.Image, params map[string]float64) (superpixel.SpResult, error) {
		return Run(img, int(params["count"]), params["compactness"])
	}
	superpixel.RegisterSpAlgorithm("vcell", "VCells", "This is an implementation of VCells superpixels, based on the original paper.", runFromMap, parameterInfo)
}

// Run executes the VCells superpixel algorithm on the given image.
func Run(img image.Image, count int, compactness float64) (superpixel.SpResult, error) {

	const iterations = 10
	const omega = 3

	// Validate Parameters
	if count < 2 {
		count = 2
	}

	if compactness <= 0 {
		compactness = 0.1
	}

	if img == nil {
		return superpixel.SpResult{}, errors.New("Image is nil.")
	}

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pixelCount := width * height

	if pixelCount == 0 {
		return superpixel.SpResult{}, errors.New("Image has zero dimensions.")
	}

	// Prepare the state
	state := vcellState{
		img:           img,
		imgHeight:     height,
		imgWidth:      width,
		imgPixelCount: pixelCount,
		count:         count,
		compactness:   compactness,
		iterations:    iterations,
		omega:         omega,
		labelMap:      make([]int, pixelCount),
		labCache:      make([][3]float64, pixelCount),
		diskOffsets:   buildDiskOffsets(omega),
	}

	// Build Lab cache
	superpixel_utils.BuildLabCache(img, &state.labCache)

	// Stage 1 uniform hexagonal initial segmentation
	state.initializeClusters()

	// Stage 2 EWCVT-LNN iterations
	state.runEwcvtLnn()

	// Detecting-Segment-Breaking (DSB / Algorithm 1)
	state.detectSegmentBreaking()

	// Write the final result into a SpResult struct
	resultLabelMap := superpixel_map.NewLabelMap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			label := state.labelMap[y*width+x]
			resultLabelMap.SetLabelAt(x, y, uint32(label))
		}
	}

	result := superpixel.NewSpResult(resultLabelMap)
	return result, nil
}

type vcellSpCluster struct {
	pixelCount       int
	sumL, sumA, sumB float64
	l, a, b          float64 // Color centroid
}

type vcellState struct {
	img           image.Image
	imgWidth      int
	imgHeight     int
	imgPixelCount int

	count       int
	compactness float64
	iterations  int
	omega       int

	clusters    []vcellSpCluster
	labelMap    []int
	labCache    [][3]float64
	diskOffsets [][2]int
}

func buildDiskOffsets(omega int) [][2]int {
	var offsets [][2]int
	omega2 := omega * omega
	for dy := -omega; dy <= omega; dy++ {
		for dx := -omega; dx <= omega; dx++ {
			if dx == 0 && dy == 0 {
				continue
			}
			if dx*dx+dy*dy <= omega2 {
				offsets = append(offsets, [2]int{dx, dy})
			}
		}
	}
	return offsets
}

// initializeClusters performs the initial segmentation.
func (state *vcellState) initializeClusters() {

	points, gridInterval := superpixel_utils.RegularGridPoints(state.imgWidth, state.imgHeight, state.count)

	// Build spatial buckets for fast nearest-center lookup
	bucketSize := gridInterval
	if bucketSize < 1 {
		bucketSize = 1
	}
	bucketCols := (state.imgWidth + bucketSize - 1) / bucketSize
	bucketRows := (state.imgHeight + bucketSize - 1) / bucketSize
	buckets := make([][]int, bucketCols*bucketRows)
	for index, point := range points {
		bx := point.X / bucketSize
		by := point.Y / bucketSize
		if bx >= bucketCols {
			bx = bucketCols - 1
		}
		if by >= bucketRows {
			by = bucketRows - 1
		}
		bIdx := by*bucketCols + bx
		buckets[bIdx] = append(buckets[bIdx], index)
	}

	// Assign each pixel to the nearest initial center
	for y := 0; y < state.imgHeight; y++ {
		by := y / bucketSize
		for x := 0; x < state.imgWidth; x++ {
			bx := x / bucketSize
			bestDist := math.MaxInt64
			bestPointIdx := 0

			for dy := -2; dy <= 2; dy++ {
				nby := by + dy
				if nby < 0 || nby >= bucketRows {
					continue
				}
				for dx := -2; dx <= 2; dx++ {
					nbx := bx + dx
					if nbx < 0 || nbx >= bucketCols {
						continue
					}
					for _, ptIdx := range buckets[nby*bucketCols+nbx] {
						pt := points[ptIdx]
						d := (x-pt.X)*(x-pt.X) + (y-pt.Y)*(y-pt.Y)
						if d < bestDist {
							bestDist = d
							bestPointIdx = ptIdx
						}
					}
				}
			}
			state.labelMap[y*state.imgWidth+x] = bestPointIdx
		}
	}

	// Compact labels to 0..numClusters-1
	usedLabels := make(map[int]int)
	for _, l := range state.labelMap {
		if _, exists := usedLabels[l]; !exists {
			usedLabels[l] = len(usedLabels)
		}
	}

	numClusters := len(usedLabels)
	state.clusters = make([]vcellSpCluster, numClusters)

	for y := 0; y < state.imgHeight; y++ {
		for x := 0; x < state.imgWidth; x++ {
			idx := y*state.imgWidth + x
			remapped := usedLabels[state.labelMap[idx]]
			state.labelMap[idx] = remapped

			lab := state.labCache[idx]
			state.clusters[remapped].pixelCount++
			state.clusters[remapped].sumL += lab[0]
			state.clusters[remapped].sumA += lab[1]
			state.clusters[remapped].sumB += lab[2]
		}
	}

	for i := range state.clusters {
		c := &state.clusters[i]
		if c.pixelCount > 0 {
			cnt := float64(c.pixelCount)
			c.l = c.sumL / cnt
			c.a = c.sumA / cnt
			c.b = c.sumB / cnt
		}
	}
}

// runEwcvtLnn performs Algorithm 2 (EWCVT-LNN).
func (state *vcellState) runEwcvtLnn() {
	width := state.imgWidth
	height := state.imgHeight

	for iter := 0; iter < state.iterations; iter++ {
		pixelMoved := false

		for y := 0; y < height; y++ {
			for x := 0; x < width; x++ {
				idx := y*width + x
				currentLabel := state.labelMap[idx]

				// Check if pixel is on a cluster boundary
				isOnBoundary := false
				for _, dir := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nx := x + dir[0]
					ny := y + dir[1]
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						if state.labelMap[ny*width+nx] != currentLabel {
							isOnBoundary = true
							break
						}
					}
				}
				if !isOnBoundary {
					continue
				}

				// Collect candidate clusters from disk neighborhood IN_omega(p)
				var candidates [32]int
				var candidateDiskCounts [32]int
				candidates[0] = currentLabel
				candidateCount := 1
				validNeighbors := 0

				for _, offset := range state.diskOffsets {
					nx := x + offset[0]
					ny := y + offset[1]
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						validNeighbors++
						qLabel := state.labelMap[ny*width+nx]
						found := false
						for c := 0; c < candidateCount; c++ {
							if candidates[c] == qLabel {
								candidateDiskCounts[c]++
								found = true
								break
							}
						}
						if !found && candidateCount < len(candidates) {
							candidates[candidateCount] = qLabel
							candidateDiskCounts[candidateCount] = 1
							candidateCount++
						}
					}
				}

				// Find m = argmin_k dist(p, g_k)
				lab := state.labCache[idx]
				bestDistSq := math.Inf(1)
				bestCluster := currentLabel

				for c := 0; c < candidateCount; c++ {
					k := candidates[c]
					nK := float64(validNeighbors - candidateDiskCounts[c])
					centroid := state.clusters[k]

					diffL := lab[0] - centroid.l
					diffA := lab[1] - centroid.a
					diffB := lab[2] - centroid.b
					colorDistSq := diffL*diffL + diffA*diffA + diffB*diffB

					distSq := colorDistSq + 2.0*state.compactness*nK
					if distSq < bestDistSq {
						bestDistSq = distSq
						bestCluster = k
					}
				}

				// If m != l, transfer p to cluster m and update centroids
				if bestCluster != currentLabel && state.clusters[currentLabel].pixelCount > 1 {
					pixelMoved = true
					state.labelMap[idx] = bestCluster

					// Update old cluster
					oldC := &state.clusters[currentLabel]
					oldC.pixelCount--
					oldC.sumL -= lab[0]
					oldC.sumA -= lab[1]
					oldC.sumB -= lab[2]
					cntOld := float64(oldC.pixelCount)
					oldC.l = oldC.sumL / cntOld
					oldC.a = oldC.sumA / cntOld
					oldC.b = oldC.sumB / cntOld

					// Update new cluster
					newC := &state.clusters[bestCluster]
					newC.pixelCount++
					newC.sumL += lab[0]
					newC.sumA += lab[1]
					newC.sumB += lab[2]
					cntNew := float64(newC.pixelCount)
					newC.l = newC.sumL / cntNew
					newC.a = newC.sumA / cntNew
					newC.b = newC.sumB / cntNew
				}
			}
		}

		if !pixelMoved {
			break
		}
	}
}

// detectSegmentBreaking performs Algorithm 1 (DSB) using connected components flood fill.
func (state *vcellState) detectSegmentBreaking() {
	width := state.imgWidth
	height := state.imgHeight
	pixelCount := state.imgPixelCount

	visited := make([]bool, pixelCount)
	clusterVisited := make(map[int]bool)
	newLabelMap := make([]int, pixelCount)

	nextClusterId := len(state.clusters)
	queue := make([]int, 0, 128)
	component := make([]int, 0, 128)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			idx := y*width + x
			if visited[idx] {
				continue
			}

			targetLabel := state.labelMap[idx]
			visited[idx] = true

			queue = queue[:0]
			queue = append(queue, idx)
			component = component[:0]
			component = append(component, idx)

			for len(queue) > 0 {
				curr := queue[0]
				queue = queue[1:]

				cx := curr % width
				cy := curr / width

				for _, dir := range [4][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
					nx := cx + dir[0]
					ny := cy + dir[1]
					if nx >= 0 && nx < width && ny >= 0 && ny < height {
						nIdx := ny*width + nx
						if !visited[nIdx] && state.labelMap[nIdx] == targetLabel {
							visited[nIdx] = true
							queue = append(queue, nIdx)
							component = append(component, nIdx)
						}
					}
				}
			}

			var assignedLabel int
			if !clusterVisited[targetLabel] {
				clusterVisited[targetLabel] = true
				assignedLabel = targetLabel
			} else {
				assignedLabel = nextClusterId
				nextClusterId++
			}

			for _, pIdx := range component {
				newLabelMap[pIdx] = assignedLabel
			}
		}
	}

	state.labelMap = newLabelMap
}
