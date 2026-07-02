package superpixel_utils

import (
	"image"
)

// Returns the number of pixels in the mask that are covered
func Area(mask *image.Gray16) int {
	bounds := mask.Bounds()

	covered := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.Gray16At(x, y).Y > 0 {
				covered++
			}
		}
	}

	return covered
}
