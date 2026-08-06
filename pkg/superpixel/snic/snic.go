package snic

/*

This is an implementation of SNIC superpixels, based on the original paper.

@inproceedings{achanta2017superpixels,
  title={Superpixels and polygons using simple non-iterative clustering},
  author={Achanta, Radhakrishna and Susstrunk, Sabine},
  booktitle={Proceedings of the IEEE conference on computer vision and pattern recognition},
  pages={4651--4660},
  year={2017}
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
			Description: "The compactness of the superpixels. Higher values result in more compact superpixels.",
			Kind:        superpixel.SpParameterKindFloat,
			Default:     10.0,
			Min:         0,
			Max:         250,
		},
		{
			Id:          "initial_perturbation_neighborhood_radius",
			Title:       "Initial Perturbation Neighborhood Radius",
			Description: "The radius of the neighborhood in which the initial cluster centers are perturbed to avoid placing them on edges.",
			Kind:        superpixel.SpParameterKindInt,
			Default:     1,
			Min:         0,
			Max:         16,
		},
	}
	runFromMap := func(img image.Image, params map[string]float64) (superpixel.SpResult, error) {
		return Run(img, int(params["count"]), params["compactness"], int(params["initial_perturbation_neighborhood_radius"]))
	}
	superpixel.RegisterSpAlgorithm("snic", "SNIC", "This is an implementation of SNIC superpixels, based on the original paper.", runFromMap, parameterInfo)
}

// Run executes the SNIC algorithm on the given image.
func Run(img image.Image, count int, compactness float64, initialPerturbationNeighborhoodRadius int) (superpixel.SpResult, error) {

	// Validate parameters
	if count < 2 {
		return superpixel.SpResult{}, errors.New("Superpixel count must be at least 2.")
	}

	if compactness < 0 {
		return superpixel.SpResult{}, errors.New("Compactness can not be negative.")
	}

	if initialPerturbationNeighborhoodRadius < 0 {
		return superpixel.SpResult{}, errors.New("Initial perturbation neighborhood radius can not be negative.")
	}

	if img == nil {
		return superpixel.SpResult{}, errors.New("Image is nil.")
	}

	// Get image dimensions
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pixelCount := width * height

	// Prepare the state
	state := snicState{
		img:                                   img,
		imgHeight:                             height,
		imgWidth:                              width,
		imgPixelCount:                         pixelCount,
		count:                                 count,
		compactness:                           compactness,
		initialPerturbationNeighborhoodRadius: initialPerturbationNeighborhoodRadius,
		labelMap:                              make([]int, pixelCount),
		labCache:                              make([][3]float64, pixelCount),
	}

	// Initialize labels to -1 (unassigned)
	for i := range state.labelMap {
		state.labelMap[i] = -1
	}

	// Main steps
	superpixel_utils.BuildLabCache(img, &state.labCache)
	state.initializeClusters()
	state.growSuperpixels()

	// Write the final result into a SpResult struct
	resultLabelMap := image.NewGray16(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			label := state.labelMap[y*width+x]
			resultLabelMap.SetGray16(x, y, color.Gray16{Y: uint16(label)})
		}
	}
	result := superpixel.MakeSpResult(resultLabelMap)
	return result, nil

}

type snicSpCluster struct {
	x, y    int
	l, a, b float64

	// Running sums for incremental centroid updates
	pixelCount       int
	sumX, sumY       int64
	sumL, sumA, sumB float64
}

type snicState struct {
	// Generic algorithm data
	img           image.Image
	imgWidth      int
	imgHeight     int
	imgPixelCount int

	// Specific parameters
	count                                 int
	compactness                           float64
	initialPerturbationNeighborhoodRadius int

	// State
	gridInterval int             // Interval in which the initial cluster centers are placed.
	clusters     []snicSpCluster // The current cluster centers, indexed by cluster label.

	// State maps, all indexed by y*width+x
	labelMap []int        // For each pixel, the index of the assigned cluster (-1 if unassigned).
	labCache [][3]float64 // Pre-computed LAB values for each pixel.
}

// Priority queue item used by the SNIC flood-fill expansion.
type snicQueueItem struct {
	x, y         int
	clusterIndex int
	distance     float64
}

func snicQueueItemComparator(a, b snicQueueItem) int {
	switch {
	case a.distance < b.distance:
		return -1
	case a.distance > b.distance:
		return 1
	default:
		return 0
	}
}

// The distance measure used to assign pixels to clusters, matching the SLIC formulation.
func (state *snicState) distance(cluster snicSpCluster, pixelX int, pixelY int) float64 {

	// If the pixel is out of bounds, return infinity so it is never assigned.
	if pixelX < 0 || pixelY < 0 || pixelX >= state.imgWidth || pixelY >= state.imgHeight {
		return math.Inf(1)
	}

	// Look up the pixel LAB values from the cache
	lab := state.labCache[pixelY*state.imgWidth+pixelX]
	pixelL, pixelA, pixelB := lab[0], lab[1], lab[2]

	// Calculate the distance in LAB and XY and combine it.
	distanceLab := math.Sqrt((cluster.l-pixelL)*(cluster.l-pixelL) + (cluster.a-pixelA)*(cluster.a-pixelA) + (cluster.b-pixelB)*(cluster.b-pixelB))
	distanceXY := math.Sqrt(float64((cluster.x-pixelX)*(cluster.x-pixelX) + (cluster.y-pixelY)*(cluster.y-pixelY)))
	distance := distanceLab + (state.compactness/float64(state.gridInterval))*distanceXY
	return distance

}

// Implements the gradient function used to avoid placing cluster centers on edges.
func (state *snicState) gradient(pixelX int, pixelY int) float64 {

	// Return infinity near the image borders to avoid choosing edge positions.
	if pixelX < 1 || pixelY < 1 || pixelX > state.imgWidth-2 || pixelY > state.imgHeight-2 {
		return math.Inf(1)
	}

	// Get the color components of the neighboring pixels from the cache
	rightIndex := pixelY*state.imgWidth + (pixelX + 1)
	rightL, rightA, rightB := state.labCache[rightIndex][0], state.labCache[rightIndex][1], state.labCache[rightIndex][2]
	leftIndex := pixelY*state.imgWidth + (pixelX - 1)
	leftL, leftA, leftB := state.labCache[leftIndex][0], state.labCache[leftIndex][1], state.labCache[leftIndex][2]
	downIndex := (pixelY+1)*state.imgWidth + pixelX
	downL, downA, downB := state.labCache[downIndex][0], state.labCache[downIndex][1], state.labCache[downIndex][2]
	upIndex := (pixelY-1)*state.imgWidth + pixelX
	upL, upA, upB := state.labCache[upIndex][0], state.labCache[upIndex][1], state.labCache[upIndex][2]

	// Calculate the gradient
	gradient := (rightL-leftL)*(rightL-leftL) + (rightA-leftA)*(rightA-leftA) + (rightB-leftB)*(rightB-leftB) + (downL-upL)*(downL-upL) + (downA-upA)*(downA-upA) + (downB-upB)*(downB-upB)

	return gradient
}

func (state *snicState) initializeClusters() {

	// Initialize clusters on a regular grid.
	points, gridInterval := superpixel_utils.RegularGridPoints(state.imgWidth, state.imgHeight, state.count)
	state.gridInterval = gridInterval
	state.clusters = make([]snicSpCluster, len(points))
	for index, point := range points {
		state.clusters[index].x = point.X
		state.clusters[index].y = point.Y
		state.clusters[index].l = state.labCache[point.Y*state.imgWidth+point.X][0]
		state.clusters[index].a = state.labCache[point.Y*state.imgWidth+point.X][1]
		state.clusters[index].b = state.labCache[point.Y*state.imgWidth+point.X][2]
	}

	// Move cluster centers to the lowest gradient position in their neighborhood.
	for index, cluster := range state.clusters {
		minGradient := math.Inf(1)
		newCenterX := cluster.x
		newCenterY := cluster.y

		for iX := -state.initialPerturbationNeighborhoodRadius; iX <= state.initialPerturbationNeighborhoodRadius; iX++ {
			for iY := -state.initialPerturbationNeighborhoodRadius; iY <= state.initialPerturbationNeighborhoodRadius; iY++ {
				gradient := state.gradient(cluster.x+iX, cluster.y+iY)
				if gradient < minGradient {
					minGradient = gradient
					newCenterX = cluster.x + iX
					newCenterY = cluster.y + iY
				}
			}
		}

		state.clusters[index].x = newCenterX
		state.clusters[index].y = newCenterY
		state.clusters[index].l = state.labCache[newCenterY*state.imgWidth+newCenterX][0]
		state.clusters[index].a = state.labCache[newCenterY*state.imgWidth+newCenterX][1]
		state.clusters[index].b = state.labCache[newCenterY*state.imgWidth+newCenterX][2]
	}
}

func (state *snicState) growSuperpixels() {

	pq := priorityqueue.NewWith(snicQueueItemComparator)

	// Seed the priority queue with the initial cluster centers.
	for i, cluster := range state.clusters {
		pq.Enqueue(snicQueueItem{
			x:            cluster.x,
			y:            cluster.y,
			clusterIndex: i,
			distance:     0,
		})
	}

	// 4-connected neighborhood
	dx4 := [4]int{-1, 0, 1, 0}
	dy4 := [4]int{0, -1, 0, 1}

	for !pq.Empty() {
		item, _ := pq.Dequeue()
		index := item.y*state.imgWidth + item.x

		// Skip pixels that have already been assigned to another cluster.
		if state.labelMap[index] >= 0 {
			continue
		}

		// Assign the pixel to the cluster that reached it.
		state.labelMap[index] = item.clusterIndex

		// Update the cluster centroid incrementally.
		cluster := &state.clusters[item.clusterIndex]
		cluster.pixelCount++
		cluster.sumX += int64(item.x)
		cluster.sumY += int64(item.y)
		lab := state.labCache[index]
		cluster.sumL += lab[0]
		cluster.sumA += lab[1]
		cluster.sumB += lab[2]

		cluster.x = int(cluster.sumX / int64(cluster.pixelCount))
		cluster.y = int(cluster.sumY / int64(cluster.pixelCount))
		cluster.l = cluster.sumL / float64(cluster.pixelCount)
		cluster.a = cluster.sumA / float64(cluster.pixelCount)
		cluster.b = cluster.sumB / float64(cluster.pixelCount)

		// Add unassigned neighbors to the queue using the updated centroid.
		for n := 0; n < 4; n++ {
			nx := item.x + dx4[n]
			ny := item.y + dy4[n]
			if nx < 0 || ny < 0 || nx >= state.imgWidth || ny >= state.imgHeight {
				continue
			}
			nindex := ny*state.imgWidth + nx
			if state.labelMap[nindex] >= 0 {
				continue
			}
			distance := state.distance(*cluster, nx, ny)
			pq.Enqueue(snicQueueItem{
				x:            nx,
				y:            ny,
				clusterIndex: item.clusterIndex,
				distance:     distance,
			})
		}
	}
}
