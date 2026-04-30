package superpixel

import (
	"image"
)

type SpImplementation interface {

	// SetImage sets the image to be processed.
	SetImage(img image.Image)

	// GetImage returns the image currently set for processing.
	GetImage() image.Image

	// SetRecommendedParameters sets the algorithm parameters to their recommended values.
	// This may be a fixed default, or it can be based on the properties of the currently set image.
	SetRecommendedParameters()

	// SetParameter sets a specific algorithm parameter to the given value.
	// Returns an error if the parameter is not supported or if the value is invalid.
	SetParameter(parameter SpParameter, value float64) error

	// GetSupportedParameters returns a list of parameters supported by this algorithm.
	GetSupportedParameters() []SpParameter

	// Run executes the superpixel algorithm on the set image.
	Run() error

	// GetResult returns the result of the algorithm execution.
	GetResult() *SpResult
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

func (this *SpAlgorithmData) GetResult() *SpResult {
	return &this.Result
}
