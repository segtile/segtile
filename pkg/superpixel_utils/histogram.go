package superpixel_utils

import "image"

func Histogram(img *image.Gray16) []int {
	hist := make([]int, 65536)
	bounds := img.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			val := img.Gray16At(x, y).Y
			hist[val]++
		}
	}
	return hist
}
