package slic

/*

This is an implementation of SLIC superpixels, based on the original paper.

@techreport{achanta2010slic,
  title={Slic superpixels},
  author={Achanta, Radhakrishna and Shaji, Appu and Smith, Kevin and Lucchi, Aurelien and Fua, Pascal and S{\"u}sstrunk, Sabine and others},
  year={2010},
  institution={Technical report EPFL}
}

Code architecture is inspired by github.com/kurige/SLIC/, licensed under MIT.

*/

import (
	"errors"
	"image"
	"image/color"
	"math"

	vendor_slic "github.com/kurige/SLIC"
	"github.com/lucasb-eyer/go-colorful"
	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/util"
)

type slicSpCluster struct {
	loc image.Point
	col colorful.Color
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
	gridInterval  int
	clusters      []slicSpCluster
	bestDistances []float64 // For each pixel, the distance to the closest cluster center. Used in the main loop.

}

func (this *SpSlic) SetDefaultParameters() {
	// Set default parameters for the SLIC algorithm
	this.numSuperpixels = 800
	this.compactness = 10
	this.initialPertubationNeighborhood = 3
	this.iterations = 12
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
	this.bestDistances = nil // Reset the best distances
}

// The distance measure SLIC uses to assign pixels to clusters, as defined in the original paper
// See Equation 1 in the original paper
func (this *SpSlic) distance(cluster slicSpCluster, pixelX int, pixelY int) (gradient float64, err error) {

	// If the pixel is out of bounds, we return infinity and an error to make sure it is not assigned to any cluster.
	if pixelX < 0 || pixelY < 0 || pixelX >= this.Img.Bounds().Dx() || pixelY >= this.Img.Bounds().Dy() {
		return math.Inf(1), errors.New("Out of bounds, assuming infinite distance")
	}

	// Get the color of the cluster pixel and reference pixel in LAB
	clusterL, clusterA, clusterB := cluster.col.Lab()
	pixelColor, _ := colorful.MakeColor(this.Img.At(pixelX, pixelY))
	pixelL, pixelA, pixelB := pixelColor.Lab()

	// Calculate the distance in LAB and XY and combine it
	distanceLab := math.Sqrt(math.Pow(clusterL-pixelL, 2) + math.Pow(clusterA-pixelA, 2) + math.Pow(clusterB-pixelB, 2))
	distanceXY := math.Sqrt(math.Pow(float64(cluster.loc.X-pixelX), 2) + math.Pow(float64(cluster.loc.Y-pixelY), 2))
	distance := distanceLab + (this.compactness/float64(this.gridInterval))*distanceXY
	return distance, nil

}

// Implements the gradient function SLIC uses to avoid placing cluster centers on edges
// See Equation 2 in the original paper
func (this *SpSlic) gradient(pixelX int, pixelY int) (float64, error) {
	width := this.Img.Bounds().Dx()
	height := this.Img.Bounds().Dy()

	// The original equation does not specify what to do near the image borders,
	// but since having cluster centers right near the edge is generally undesirable,
	// we just return infinity to make this very unlikely to be chosen as a cluster center.
	if pixelX < 1 || pixelY < 1 || pixelX > width-2 || pixelY > height-2 {
		return math.Inf(1), errors.New("Out of bounds, assuming infinite error")
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

func (this *SpSlic) NewRun() error {

	if this.Img == nil {
		return errors.New("Image not set.")
	}

	// Get image dimensions
	width := this.GetImage().Bounds().Dx()
	height := this.GetImage().Bounds().Dy()

	// Calculate the grid of initial cluster centers
	var points []image.Point
	points, this.gridInterval = util.RegularGridPoints(width, height, this.numSuperpixels)
	this.clusters = make([]slicSpCluster, len(points))
	for index, point := range points {
		this.clusters[index].loc = point
		this.clusters[index].col, _ = colorful.MakeColor(this.Img.At(point.X, point.Y))
	}

	// Move cluster centers to lowest gradient position in neighborhood.
	// In the original paper, this is explicitly defined as a 3x3 neighborhood, but we make it adjustable.
	for index, cluster := range this.clusters {
		minGradient := math.Inf(1)
		newCenter := cluster.loc

		for iX := -this.initialPertubationNeighborhood; iX <= this.initialPertubationNeighborhood; iX++ {
			for iY := -this.initialPertubationNeighborhood; iY <= this.initialPertubationNeighborhood; iY++ {
				gradient, _ := this.gradient(cluster.loc.X+iX, cluster.loc.Y+iY)
				if gradient < minGradient {
					minGradient = gradient
					newCenter = image.Point{X: cluster.loc.X + iX, Y: cluster.loc.Y + iY}
				}
			}
		}
		this.clusters[index].loc = newCenter
	}

	// The main loop of the SLIC algorithm

	for i := 0; i < this.iterations; i++ {
		this.resetDistances()

	}

	return nil
}

func (this *SpSlic) labelPixels() {

	width := this.Img.Bounds().Dx()

	// For every cluster, test the distance metric of pixels in a 2*gridInterval x 2*gridInterval neighborhood around the cluster center.
	// Assign them to the cluster if they are closer than their previous best cluster.
	for clusterIndex, cluster := range this.clusters {
		for iX := -this.gridInterval; iX <= this.gridInterval; iX++ {
			for iY := -this.gridInterval; iY <= this.gridInterval; iY++ {
				pixelX := cluster.loc.X + iX
				pixelY := cluster.loc.Y + iY
				distance, err := this.distance(cluster, pixelX, pixelY)
				if err != nil {
					continue // Skip out of bounds pixels
				}
				if distance < this.bestDistances[pixelY*width+pixelX] {
					this.bestDistances[pixelY*width+pixelX] = distance
				}
			}
		}
	}
}

func (this *SpSlic) resetDistances() {
	if this.bestDistances == nil {
		this.bestDistances = make([]float64, this.Img.Bounds().Dx()*this.Img.Bounds().Dy())
	}
	for index := range this.bestDistances {
		this.bestDistances[index] = math.Inf(1)
	}
}

// Kept around for now as a reference
func (this *SpSlic) Run() error {

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
