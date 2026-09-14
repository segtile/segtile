package superpixel_utils

import (
	"image"
)

func HasNonZeroPixels(img *image.Gray16) bool {
	for _, p := range img.Pix {
		if p != 0 {
			return true
		}
	}
	return false
}

func ResetImage(img *image.Gray16) {
	for i := range img.Pix {
		img.Pix[i] = 0
	}
}

func HasPixelWithValue(img *image.Gray16, value uint16) bool {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if UnsafeGray16At(img, x, y) == value {
				return true
			}
		}
	}
	return false
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

// UnsafeGray16At returns the 16-bit grayscale value at the specified coordinates without bounds checking.
func UnsafeGray16At(img *image.Gray16, x, y int) uint16 {
	offset := img.PixOffset(x, y)
	return uint16(img.Pix[offset])<<8 | uint16(img.Pix[offset+1])
}

func UnsafeSetGray16At(img *image.Gray16, x, y int, value uint16) {
	offset := img.PixOffset(x, y)
	img.Pix[offset] = uint8(value >> 8)
	img.Pix[offset+1] = uint8(value & 0xFF)
}
