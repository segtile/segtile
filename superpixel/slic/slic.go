package slic

import (
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
	this.numSuperpixels = 1500
	this.compactness = 25.0
	this.iterations = 12

}

func (this *SpSlic) Run() error {

	// Get image dimensions
	width := this.GetImage().Bounds().Dx()
	height := this.GetImage().Bounds().Dy()

	// Use the library to perform SLIC segmentation
	vendorSlic := vendor_slic.MakeSlic(this.GetImage(), this.compactness, width*height/this.numSuperpixels)
	vendorSlic.Run(this.iterations)

	// Convert the result into a 2D label map
	labelmap := make([][]uint16, width)
	for x := 0; x < width; x++ {
		labelmap[x] = make([]uint16, height)
		for y := 0; y < height; y++ {
			labelmap[x][y] = uint16(vendorSlic.Labels[y*width+x])
		}
	}

	this.Result.LabelMap = labelmap
	this.Result.SpCount = len(vendorSlic.Superpixels)

	return nil
}
