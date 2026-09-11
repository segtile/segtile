package slic

/*

This is an implementation of SLIC superpixels, based on the original paper.

@techreport{achanta2010slic,
  title={Slic superpixels},
  author={Achanta, Radhakrishna and Shaji, Appu and Smith, Kevin and Lucchi, Aurelien and Fua, Pascal and S{\"u}sstrunk, Sabine and others},
  year={2010},
  institution={Technical report EPFL}
}

Some parts of the implementation are based on the implementation in github.com/kurige/SLIC/, licensed under MIT.

*/

import (
	"errors"
	"image"
	"image/color"
	"math"
	"runtime"
	"sync"

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
	}
	runFromMap := func(img image.Image, params map[string]float64) (superpixel.SpResult, error) {
		return Run(img, int(params["count"]), params["compactness"])
	}
	superpixel.RegisterSpAlgorithm("slic", "SLIC", "This is an implementation of SLIC superpixels, based on the original paper.", runFromMap, parameterInfo)
}

// This is the new run function.
// It works like the old one, but no longer uses a stateful struct, instead it is completely functional.
func Run(img image.Image, count int, compactness float64) (superpixel.SpResult, error) {

	const initialPerturbationNeighborhoodRadius = 1
	const iterations = 8

	// Validate Parameters

	if count < 2 {
		return superpixel.SpResult{}, errors.New("Superpixel count must be at least 2.")
	}

	if compactness < 0 {
		return superpixel.SpResult{}, errors.New("Compactness can not be negative.")
	}

	if initialPerturbationNeighborhoodRadius < 0 {
		return superpixel.SpResult{}, errors.New("Initial perturbation neighborhood radius can not be negative.")
	}

	if iterations < 1 {
		return superpixel.SpResult{}, errors.New("Iterations must be at least 1.")
	}

	if img == nil {
		return superpixel.SpResult{}, errors.New("Image is nil.")
	}

	// Get image dimensions
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pixelCount := width * height

	// Prepare the state
	state := slicState{
		img:                                   img,
		imgHeight:                             height,
		imgWidth:                              width,
		imgPixelCount:                         pixelCount,
		count:                                 count,
		compactness:                           compactness,
		initialPerturbationNeighborhoodRadius: initialPerturbationNeighborhoodRadius,
		iterations:                            iterations,
		labelMap:                              make([]int, pixelCount),
		labCache:                              make([][3]float64, pixelCount),
	}

	// Perform algorithm steps
	superpixel_utils.BuildLabCache(img, &state.labCache)
	state.initializeClusters()

	for i := 0; i < state.iterations; i++ {
		state.resetDistances()
		state.labelPixels()
		state.recalculateClusterCenters()
	}

	// Write the final result into a SpResult struct
	resultLabelMap := image.NewGray16(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			label := state.labelMap[y*width+x]
			resultLabelMap.SetGray16(x, y, color.Gray16{Y: uint16(label)})
		}
	}
	result := superpixel.NewSpResult(resultLabelMap, 0)
	return result, nil

}

type slicSpCluster struct {
	x, y    int
	l, a, b float64
}

type slicState struct {
	// Generic algorithm data
	img           image.Image
	imgWidth      int
	imgHeight     int
	imgPixelCount int

	// Specific parameters (no longer needed, we use them directly from the Run method)
	count                                 int
	compactness                           float64
	initialPerturbationNeighborhoodRadius int
	iterations                            int

	// State
	gridInterval int             // Interval in which the initial cluster centers are placed.
	clusters     []slicSpCluster // The current cluster centers, indexed by cluster label.

	// State maps, all indexed by y*width+x
	labelMap         []int        // For each pixel, the index of the assigned cluster.
	bestDistancesMap []float64    // For each pixel, the distance to the closest cluster center.
	labCache         [][3]float64 // Pre-computed LAB values for each pixel.

}

// The distance measure SLIC uses to assign pixels to clusters, see Equation 1 in the original paper
// clL, clA, clB are the pre-computed LAB values of the cluster center, to avoid recomputing them for every pixel.
func (state *slicState) distance(cluster slicSpCluster, pixelX int, pixelY int) (gradient float64, err error) {

	// If the pixel is out of bounds, we return infinity and an error.

	if pixelX < 0 || pixelY < 0 || pixelX >= state.imgWidth || pixelY >= state.imgHeight {
		return math.Inf(1), errors.New("Out of bounds, assuming infinite distance")
	}

	// Look up the pixel LAB values from the cache
	lab := state.labCache[pixelY*state.imgWidth+pixelX]
	pixelL, pixelA, pixelB := lab[0], lab[1], lab[2]

	// Calculate the distance in LAB and XY and combine it.
	distanceLab := math.Sqrt((cluster.l-pixelL)*(cluster.l-pixelL) + (cluster.a-pixelA)*(cluster.a-pixelA) + (cluster.b-pixelB)*(cluster.b-pixelB))
	distanceXY := math.Sqrt(float64((cluster.x-pixelX)*(cluster.x-pixelX) + (cluster.y-pixelY)*(cluster.y-pixelY)))
	distance := distanceLab + (state.compactness/float64(state.gridInterval))*distanceXY
	return distance, nil

}

// Implements the gradient function SLIC uses to avoid placing cluster centers on edges.
// See Equation 2 in the original paper.
func (state *slicState) gradient(pixelX int, pixelY int) float64 {

	// The original equation does not specify what to do near the image borders,
	// but since having cluster centers right near the edge is generally undesirable,
	// we just return infinity to make this very unlikely to be chosen as a cluster center.
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

// This step is not discussed in detail in the original paper which is why this function leans more on the existing implementation mentioned above.
func (state *slicState) labelPixels() {

	workers := runtime.NumCPU()

	var wg sync.WaitGroup
	wg.Add(workers)

	// Process the image in bands
	for w := range workers {
		bandStart := w * state.imgHeight / workers
		bandEnd := (w + 1) * state.imgHeight / workers

		go func(rowStart, rowEnd int) {
			defer wg.Done()

			// For every cluster, test the distance metric of pixels in a 2*gridInterval x 2*gridInterval neighborhood around the cluster center.
			// Assign them to the cluster if they are closer than their previous best cluster.
			for clusterIndex, cluster := range state.clusters {
				for iX := -state.gridInterval; iX <= state.gridInterval; iX++ {
					pixelX := cluster.x + iX
					// Clamp the iY range so this goroutine only touches rows in its band.
					iYMin := max(-state.gridInterval, rowStart-cluster.y)
					iYMax := min(state.gridInterval, rowEnd-1-cluster.y)
					for iY := iYMin; iY <= iYMax; iY++ {
						pixelY := cluster.y + iY
						distance, err := state.distance(cluster, pixelX, pixelY)
						if err != nil {
							continue // Skip out of bounds pixels
						}
						index := pixelY*state.imgWidth + pixelX
						if distance < state.bestDistancesMap[index] {
							state.bestDistancesMap[index] = distance
							state.labelMap[index] = clusterIndex
						}
					}
				}
			}
		}(bandStart, bandEnd)
	}

	wg.Wait()
}

func (state *slicState) recalculateClusterCenters() {

	// For every cluster, calculate the mean position and color of pixels assigned to it and update the cluster center.
	clusterPixelCount := make([]uint64, len(state.clusters))
	clusterXPos := make([]uint64, len(state.clusters))
	clusterYPos := make([]uint64, len(state.clusters))
	clusterL := make([]float64, len(state.clusters))
	clusterA := make([]float64, len(state.clusters))
	clusterB := make([]float64, len(state.clusters))

	// Sum up all pixel counts, positions, and LAB color values
	for y := 0; y < state.imgHeight; y++ {
		for x := 0; x < state.imgWidth; x++ {
			label := state.labelMap[y*state.imgWidth+x]
			clusterPixelCount[label]++
			clusterXPos[label] += uint64(x)
			clusterYPos[label] += uint64(y)
			lab := state.labCache[y*state.imgWidth+x]
			clusterL[label] += lab[0]
			clusterA[label] += lab[1]
			clusterB[label] += lab[2]
		}
	}

	// Divide sums by pixel count to get the averages and save the result
	for i := 0; i < len(state.clusters); i++ {
		if clusterPixelCount[i] < 1 {
			clusterPixelCount[i] = 1
		}
		count := clusterPixelCount[i]
		state.clusters[i].x = int(clusterXPos[i] / count)
		state.clusters[i].y = int(clusterYPos[i] / count)
		state.clusters[i].l = clusterL[i] / float64(count)
		state.clusters[i].a = clusterA[i] / float64(count)
		state.clusters[i].b = clusterB[i] / float64(count)
	}

}

func (state *slicState) resetDistances() {
	if state.bestDistancesMap == nil {
		state.bestDistancesMap = make([]float64, state.imgWidth*state.imgHeight)
	}
	for index := range state.bestDistancesMap {
		state.bestDistancesMap[index] = math.Inf(1)
	}
}

func (state *slicState) initializeClusters() {

	// Initialize clusters
	var points []image.Point
	points, state.gridInterval = superpixel_utils.RegularGridPoints(state.imgWidth, state.imgHeight, state.count)
	state.clusters = make([]slicSpCluster, len(points))
	for index, point := range points {
		state.clusters[index].x = point.X
		state.clusters[index].y = point.Y
		state.clusters[index].l = state.labCache[point.Y*state.imgWidth+point.X][0]
		state.clusters[index].a = state.labCache[point.Y*state.imgWidth+point.X][1]
		state.clusters[index].b = state.labCache[point.Y*state.imgWidth+point.X][2]
	}

	// Move cluster centers to lowest gradient position in neighborhood.
	// In the original paper, this is explicitly defined as a 3x3 neighborhood, but we make it adjustable.
	for index, cluster := range state.clusters {
		minGradient := math.Inf(1)
		newCenterX := cluster.x
		newCenterY := cluster.y

		for iX := -state.initialPerturbationNeighborhoodRadius; iX <= state.initialPerturbationNeighborhoodRadius; iX++ {
			for iY := -state.initialPerturbationNeighborhoodRadius; iY <= state.initialPerturbationNeighborhoodRadius; iY++ {
				gradient := state.gradient(int(cluster.x)+iX, int(cluster.y)+iY)
				if gradient < minGradient {
					minGradient = gradient
					newCenterX = cluster.x + iX
					newCenterY = cluster.y + iY
				}
			}
		}
		state.clusters[index].x = newCenterX
		state.clusters[index].y = newCenterY
	}
}
