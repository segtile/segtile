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
	"fmt"
	"image"
	"image/color"
	"math"

	vendor_slic "github.com/kurige/SLIC"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/util"
)

type slicSpCluster struct {
	x, y    int
	l, a, b float64
}

type SpSlic struct {
	// Generic algorithm data
	superpixel.SpAlgorithmData

	// Specific parameters
	numSuperpixels                 int
	compactness                    float64
	initialPertubationNeighborhood int
	iterations                     int

	// State
	gridInterval int             // Interval in which the initial cluster centers are placed.
	clusters     []slicSpCluster // The current cluster centers, indexed by cluster label.

	// State maps, all indexed by y*width+x
	labelMap         []uint16     // For each pixel, the index of the assigned cluster.
	bestDistancesMap []float64    // For each pixel, the distance to the closest cluster center.
	labCache         [][3]float64 // Pre-computed LAB values for each pixel.

}

func (this *SpSlic) SetDefaultParameters() {
	this.numSuperpixels = 800
	this.compactness = 20
	this.initialPertubationNeighborhood = 3
	this.iterations = 8
}

func (this *SpSlic) SetNumSuperpixels(num int) {
	this.numSuperpixels = num
}

func (this *SpSlic) SetCompactness(compactness float64) {
	this.compactness = compactness
}

func (this *SpSlic) SetIterations(iterations int) {
	if iterations < 1 {
		iterations = 1
	}
	this.iterations = iterations
}

func (this *SpSlic) SetImage(img image.Image) {
	this.Img = img
	this.bestDistancesMap = nil
	this.Result = superpixel.SpResult{}
	this.clusters = nil
	this.gridInterval = 0
	this.bestDistancesMap = nil

	this.buildLabCache()

}

// The distance measure SLIC uses to assign pixels to clusters, see Equation 1 in the original paper
// clL, clA, clB are the pre-computed LAB values of the cluster center, to avoid recomputing them for every pixel.
func (this *SpSlic) distance(cluster slicSpCluster, pixelX int, pixelY int) (gradient float64, err error) {

	// If the pixel is out of bounds, we return infinity and an error to make sure it is not assigned to any cluster.
	width := this.Img.Bounds().Dx()
	if pixelX < 0 || pixelY < 0 || pixelX >= width || pixelY >= this.Img.Bounds().Dy() {
		return math.Inf(1), errors.New("Out of bounds, assuming infinite distance")
	}

	// Look up the pixel LAB values from the cache
	lab := this.labCache[pixelY*width+pixelX]
	pixelL, pixelA, pixelB := lab[0], lab[1], lab[2]

	// Calculate the distance in LAB and XY and combine it.
	distanceLab := math.Sqrt(math.Pow(cluster.l-pixelL, 2) + math.Pow(cluster.a-pixelA, 2) + math.Pow(cluster.b-pixelB, 2))
	distanceXY := math.Sqrt(math.Pow(float64(cluster.x-pixelX), 2) + math.Pow(float64(cluster.y-pixelY), 2))
	distance := distanceLab + (this.compactness/float64(this.gridInterval))*distanceXY
	return distance, nil

}

// Implements the gradient function SLIC uses to avoid placing cluster centers on edges.
// See Equation 2 in the original paper.
func (this *SpSlic) gradient(pixelX int, pixelY int) (float64, error) {
	width := this.Img.Bounds().Dx()
	height := this.Img.Bounds().Dy()

	// The original equation does not specify what to do near the image borders,
	// but since having cluster centers right near the edge is generally undesirable,
	// we just return infinity to make this very unlikely to be chosen as a cluster center.
	if pixelX < 1 || pixelY < 1 || pixelX > width-2 || pixelY > height-2 {
		return math.Inf(1), errors.New("Out of bounds, infinite error")
	}

	// Get the color components of the neighboring pixels in LAB
	right, _ := colorful.MakeColor(this.Img.At(pixelX+1, pixelY))
	left, _ := colorful.MakeColor(this.Img.At(pixelX-1, pixelY))
	down, _ := colorful.MakeColor(this.Img.At(pixelX, pixelY+1))
	up, _ := colorful.MakeColor(this.Img.At(pixelX, pixelY-1))

	rightL, rightA, rightB := right.Lab()
	leftL, leftA, leftB := left.Lab()
	downL, downA, downB := down.Lab()
	upL, upA, upB := up.Lab()

	// Calculate the gradient
	gradient := math.Pow(rightL-leftL, 2) + math.Pow(rightA-leftA, 2) + math.Pow(rightB-leftB, 2) + math.Pow(downL-upL, 2) + math.Pow(downA-upA, 2) + math.Pow(downB-upB, 2)

	return gradient, nil
}

// This step is not discussed in detail in the original paper which is why this function leans more on the existing implementation mentioned above.
func (this *SpSlic) enforceConnectivity() {

	// Get image dimensions
	width := this.Img.Bounds().Dx()
	height := this.Img.Bounds().Dy()
	totalPixelCount := width * height

	// Minimum component size: components smaller than minimumSpPixelCount pixels are merged into a neighbor.
	minimumSpPixelCount := (this.gridInterval * this.gridInterval) / 4

	// Read current labels into a flat int slice for efficient access during flood fill.
	labels := make([]int, totalPixelCount)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			labels[y*width+x] = int(this.Result.LabelMap.Gray16At(x, y).Y)
		}
	}

	// Templates for creating the 4-connected neighborhood of a pixel
	dx4 := [4]int{-1, 0, 1, 0}
	dy4 := [4]int{0, -1, 0, 1}

	// Vectors for the flood fill.
	xVector := make([]int, totalPixelCount)
	yVector := make([]int, totalPixelCount)

	oindex := 0
	adjacentLabel := 0

	label := 0
	newLabelMap := make([]int, totalPixelCount)
	for i := range newLabelMap {
		newLabelMap[i] = -1
	}

	for j := 0; j < height; j++ {
		for k := 0; k < width; k++ {
			if newLabelMap[oindex] < 0 {
				newLabelMap[oindex] = label

				// Start a new segment
				xVector[0] = k
				yVector[0] = j

				// Quickly find an adjacent label for use later if needed
				for n := 0; n < 4; n++ {
					x := xVector[0] + dx4[n]
					y := yVector[0] + dy4[n]
					if x >= 0 && x < width && y >= 0 && y < height {
						nindex := y*width + x
						if newLabelMap[nindex] >= 0 {
							adjacentLabel = newLabelMap[nindex]
						}
					}
				}

				// Flood fill: expand to all connected pixels sharing the same original label
				originLabel := labels[oindex]
				count := 1
				for c := 0; c < count; c++ {
					for n := 0; n < 4; n++ {
						x := xVector[c] + dx4[n]
						y := yVector[c] + dy4[n]
						if x >= 0 && x < width && y >= 0 && y < height {
							nindex := y*width + x
							if newLabelMap[nindex] < 0 && labels[nindex] == originLabel {
								xVector[count] = x
								yVector[count] = y
								newLabelMap[nindex] = label
								count++
							}
						}
					}
				}

				// If the component is too small, absorb it into an adjacent segment
				if count <= minimumSpPixelCount>>2 {
					for c := 0; c < count; c++ {
						ind := yVector[c]*width + xVector[c]
						newLabelMap[ind] = adjacentLabel
					}
					label--
				}
				label++
			}
			oindex++
		}
	}

	// Write new labels back to the label map
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			this.Result.LabelMap.SetGray16(x, y, color.Gray16{Y: uint16(newLabelMap[y*width+x])})
		}
	}

	this.Result.SpCount = label
}

func (this *SpSlic) labelPixels() {

	width := this.Img.Bounds().Dx()

	// For every cluster, test the distance metric of pixels in a 2*gridInterval x 2*gridInterval neighborhood around the cluster center.
	// Assign them to the cluster if they are closer than their previous best cluster.
	for clusterIndex, cluster := range this.clusters {
		for iX := -this.gridInterval; iX <= this.gridInterval; iX++ {
			for iY := -this.gridInterval; iY <= this.gridInterval; iY++ {
				pixelX := int(cluster.x) + iX
				pixelY := int(cluster.y) + iY
				distance, err := this.distance(cluster, pixelX, pixelY)
				if err != nil {
					continue // Skip out of bounds pixels
				}
				if distance < this.bestDistancesMap[pixelY*width+pixelX] {
					this.bestDistancesMap[pixelY*width+pixelX] = distance
					this.Result.LabelMap.SetGray16(pixelX, pixelY, color.Gray16{Y: uint16(clusterIndex)})
				}
			}
		}
	}
}

func (this *SpSlic) recalculateClusterCenters() {

	// Get image dimensions
	width := this.Img.Bounds().Dx()
	height := this.Img.Bounds().Dy()

	// For every cluster, calculate the mean position and color of pixels assigned to it and update the cluster center.
	clusterPixelCount := make([]uint64, len(this.clusters))
	clusterXPos := make([]uint64, len(this.clusters))
	clusterYPos := make([]uint64, len(this.clusters))
	clusterL := make([]float64, len(this.clusters))
	clusterA := make([]float64, len(this.clusters))
	clusterB := make([]float64, len(this.clusters))

	// Sum up all pixel counts, positions, and LAB color values
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			label := this.Result.LabelMap.Gray16At(x, y).Y
			clusterPixelCount[label] += 1
			clusterXPos[label] += uint64(x)
			clusterYPos[label] += uint64(y)
			lab := this.labCache[y*width+x]
			clusterL[label] += lab[0]
			clusterA[label] += lab[1]
			clusterB[label] += lab[2]
		}
	}

	// Divide sums by pixel count to get the averages and save the result
	for i := 0; i < len(this.clusters); i++ {
		if clusterPixelCount[i] < 1 {
			clusterPixelCount[i] = 1
		}
		count := clusterPixelCount[i]
		this.clusters[i].x = int(clusterXPos[i] / count)
		this.clusters[i].y = int(clusterYPos[i] / count)
		this.clusters[i].l = clusterL[i] / float64(count)
		this.clusters[i].a = clusterA[i] / float64(count)
		this.clusters[i].b = clusterB[i] / float64(count)
	}

}
func (this *SpSlic) buildLabCache() {
	width := this.Img.Bounds().Dx()
	height := this.Img.Bounds().Dy()
	this.labCache = make([][3]float64, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			c, _ := colorful.MakeColor(this.Img.At(x, y))
			l, a, b := c.Lab()
			// go-colorful returns LAB values in a different range than standard CIELAB (L in [0,1] instead of the official [0,100]),
			// so we scale the lab values by 100 to match the scale the compactness parameter is calibrated for.
			l *= 100
			a *= 100
			b *= 100

			this.labCache[y*width+x] = [3]float64{l, a, b}
		}
	}
}

func (this *SpSlic) resetDistances() {
	if this.bestDistancesMap == nil {
		this.bestDistancesMap = make([]float64, this.Img.Bounds().Dx()*this.Img.Bounds().Dy())
	}
	for index := range this.bestDistancesMap {
		this.bestDistancesMap[index] = math.Inf(1)
	}
}

func (this *SpSlic) Run() error {

	fmt.Println("Running SLIC superpixel algorithm...")

	if this.Img == nil {
		return errors.New("Image not set.")
	}

	// Get image dimensions
	width := this.GetImage().Bounds().Dx()
	height := this.GetImage().Bounds().Dy()

	// Initialize the result struct
	this.Result = superpixel.SpResult{
		LabelMap: image.NewGray16(image.Rect(0, 0, width, height)),
		SpCount:  0,
	}

	// Initialize clusters
	var points []image.Point
	points, this.gridInterval = util.RegularGridPoints(width, height, this.numSuperpixels)
	this.clusters = make([]slicSpCluster, len(points))
	for index, point := range points {
		this.clusters[index].x = point.X
		this.clusters[index].y = point.Y
		this.clusters[index].l = this.labCache[point.Y*width+point.X][0]
		this.clusters[index].a = this.labCache[point.Y*width+point.X][1]
		this.clusters[index].b = this.labCache[point.Y*width+point.X][2]
	}

	// Move cluster centers to lowest gradient position in neighborhood.
	// In the original paper, this is explicitly defined as a 3x3 neighborhood, but we make it adjustable.
	for index, cluster := range this.clusters {
		minGradient := math.Inf(1)
		newCenterX := cluster.x
		newCenterY := cluster.y

		for iX := -this.initialPertubationNeighborhood; iX <= this.initialPertubationNeighborhood; iX++ {
			for iY := -this.initialPertubationNeighborhood; iY <= this.initialPertubationNeighborhood; iY++ {
				gradient, _ := this.gradient(int(cluster.x)+iX, int(cluster.y)+iY)
				if gradient < minGradient {
					minGradient = gradient
					newCenterX = cluster.x + iX
					newCenterY = cluster.y + iY
				}
			}
		}
		this.clusters[index].x = newCenterX
		this.clusters[index].y = newCenterY
	}

	// The main loop of the SLIC algorithm

	for i := 0; i < this.iterations; i++ {
		fmt.Printf("SLIC iteration %d/%d\n", i+1, this.iterations)
		this.resetDistances()
		this.labelPixels()
		this.recalculateClusterCenters()
	}

	this.enforceConnectivity()

	this.Result.SpCount = len(this.clusters)

	fmt.Printf("SLIC finished with %d superpixels.\n", this.Result.SpCount)

	return nil
}

// Kept around for now as a reference
func (this *SpSlic) OldRun() error {

	// Get image dimensions
	width := this.GetImage().Bounds().Dx()
	height := this.GetImage().Bounds().Dy()

	// Use the library to perform SLIC segmentation
	vendorSlic := vendor_slic.MakeSlic(this.GetImage(), this.compactness, width*height/this.numSuperpixels)
	vendorSlic.Run(this.iterations)

	// Convert the result into a Gray16 label map
	labelmap := image.NewGray16(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			label := vendorSlic.Labels[y*width+x]
			labelmap.SetGray16(x, y, color.Gray16{Y: uint16(label)})
		}
	}

	this.Result.LabelMap = labelmap

	// The connectivity algorithm can produce more SP, so we need to
	// count manually.
	maxLabel := 0
	for _, l := range vendorSlic.Labels {
		if l > maxLabel {
			maxLabel = l
		}
	}
	this.Result.SpCount = maxLabel + 1

	return nil
}
