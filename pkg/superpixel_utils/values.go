package superpixel_utils

import (
	"image"
)

func HasNonZeroPixels(img *image.Gray16) bool {
	_, maxVal := MinMax(img)
	return maxVal > 0
}

func IsBinaryMask(img *image.Gray16) bool {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			val := img.Gray16At(x, y).Y
			if val != 0 && val != 65535 {
				return false
			}
		}
	}
	return true
}

func MinMax(img *image.Gray16) (minVal, maxVal uint16) {
	minVal = 65535
	maxVal = 0
	bounds := img.Bounds()
	if bounds.Empty() {
		return 0, 0
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			val := img.Gray16At(x, y).Y
			if val > maxVal {
				maxVal = val
			}
			if val < minVal {
				minVal = val
			}
		}
	}
	return minVal, maxVal
}
