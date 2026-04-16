package superpixel

import "image"

type Superpixel interface {
	Run()
	GetLabelAt(x, y int) int
	GetLabelMap() [][]int
	DrawEdgesToImage(img image.Image) image.Image
}
