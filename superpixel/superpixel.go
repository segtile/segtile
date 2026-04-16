package superpixel

import (
	"image"
)

type SpAlgorithm interface {
	SetImage(img image.Image)
	GetImage() image.Image

	GetResult() SpResult

	SetDefaultParameters()

	Run() error
}

type SpAlgorithmData struct {
	Img    image.Image
	Result SpResult
}

func (this *SpAlgorithmData) SetImage(img image.Image) {
	this.Img = img
}

func (this *SpAlgorithmData) GetImage() image.Image {
	return this.Img
}

func (this *SpAlgorithmData) GetResult() SpResult {
	return this.Result
}

type SpResult struct {
	LabelMap [][]uint16
	SpCount  int
}
