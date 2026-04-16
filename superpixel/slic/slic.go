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
	this.numSuperpixels = 1000
	this.compactness = 10.0
	this.iterations = 10

}

func (this *SpSlic) Run() error {

	// Get image dimensions
	width := this.GetImage().Bounds().Dx()
	height := this.GetImage().Bounds().Dy()

	// Use the library to perform SLIC segmentation
	vendorSlic := vendor_slic.MakeSlic(this.GetImage(), this.compactness, width*height/this.numSuperpixels)
	vendorSlic.Run(this.iterations)

	// Convert the result into a grayscale image
	labelmap := image.NewGray16(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			label := vendorSlic.Labels[y*width+x]
			labelmap.SetGray16(x, y, color.Gray16{Y: uint16(label)})
		}
	}

	this.Result.LabelMap = labelmap
	this.Result.SpCount = len(vendorSlic.Superpixels)

	return nil
}
