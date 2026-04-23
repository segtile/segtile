package slic

import (
	"image"
	"image/color"

	vendor_slic "github.com/kurige/SLIC"
	"github.com/struffel/annotation-tool/superpixel"
)

type SpSlic struct {
	superpixel.SpAlgorithmData
	// TODO: Add parameters for the SLIC algorithm, e.g.:
	numSuperpixels int
	compactness    float64
	iterations     int
}

func (this *SpSlic) SetDefaultParameters() {
	// Set default parameters for the SLIC algorithm
	this.numSuperpixels = 800
	this.compactness = 10
	this.iterations = 12
}

func (this *SpSlic) SetNumSuperpixels(num int) {
	this.numSuperpixels = num
}

func (this *SpSlic) SetCompactness(compactness float64) {
	this.compactness = compactness
}

func (this *SpSlic) SetIterations(iterations int) {
	this.iterations = iterations
}

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
