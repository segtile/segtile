package superpixel

import (
	"image"
	"image/color"
)

type SpAlgorithm interface {
	SetImage(img image.Image)
	GetImage() image.Image

	GetResult() *SpResult

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

func (this *SpAlgorithmData) GetResult() *SpResult {
	return &this.Result
}

type SpResult struct {
	LabelMap *image.Gray16 // Image where each pixel's value is the superpixel label
	SpCount  int           // Number of superpixels
}

func (this *SpResult) GetSuperPixelBorders(borderWidth int, borderColor color.Color) *image.NRGBA {
	bounds := this.LabelMap.Bounds()
	borderImage := image.NewNRGBA(bounds)
	for x := 0; x < bounds.Dx(); x++ {
		for y := 0; y < bounds.Dy(); y++ {
			currentLabel := this.LabelMap.Gray16At(x, y).Y
			// Check neighboring pixels for different labels
			isBorder := false
			for dx := -borderWidth; dx <= borderWidth; dx++ {
				for dy := -borderWidth; dy <= borderWidth; dy++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < bounds.Dx() && ny >= 0 && ny < bounds.Dy() {
						if this.LabelMap.Gray16At(nx, ny).Y != currentLabel {
							isBorder = true
							break
						}
					}
				}
				if isBorder {
					break
				}
			}
			if isBorder {
				borderImage.Set(x, y, borderColor)
			} else {
				borderImage.Set(x, y, color.Transparent)
			}
		}
	}
	return borderImage
}
