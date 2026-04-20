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
	LabelMap [][]uint16 // 2D array where each element is the superpixel label for that pixel
	SpCount  int        // Number of superpixels
}

func (this *SpResult) GetSuperPixelBorders(borderWidth int, borderColor color.Color) *image.NRGBA {
	bounds := image.Rect(0, 0, len(this.LabelMap), len(this.LabelMap[0]))
	borderImage := image.NewNRGBA(bounds)
	for x := 0; x < bounds.Dx(); x++ {
		for y := 0; y < bounds.Dy(); y++ {
			currentLabel := this.LabelMap[x][y]
			// Check neighboring pixels for different labels
			isBorder := false
			for dx := -borderWidth; dx <= borderWidth; dx++ {
				for dy := -borderWidth; dy <= borderWidth; dy++ {
					nx, ny := x+dx, y+dy
					if nx >= 0 && nx < bounds.Dx() && ny >= 0 && ny < bounds.Dy() {
						if this.LabelMap[nx][ny] != currentLabel {
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
