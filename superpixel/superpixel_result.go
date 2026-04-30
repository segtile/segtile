package superpixel

import (
	"image"
	"image/color"
)

type SpResult struct {
	LabelMap *image.Gray16 // Image where each pixel's value is the superpixel label
	SpCount  int           // Number of superpixels
}

// Creates a mask by approximating the ground truth with the superpixels.
// A superpixel is included in the mask if its overlap with the ground truth exceeds the given threshold.
// E.G. A threshold of 0.5 indicates that 50% of the SP's pixels must also be contained in the mask.
// Additionally returns the overlap ratios for each superpixel.
func (this *SpResult) ApproximateMask(groundTruth *image.Gray16, threshold float64) (*image.Gray16, []int) {
	spBounds := this.LabelMap.Bounds()
	spMask := image.NewGray16(spBounds)

	if spBounds != groundTruth.Bounds() {
		panic("Ground truth and superpixel label map must have the same dimensions")
	}

	// Count pixels in each superpixel and their overlap with the ground truth
	spPixelCount := make([]int, this.SpCount)
	spOverlapCount := make([]int, this.SpCount)
	overlapHistogram := make([]int, 101) // Store the SP label for each overlap ratio (0-100)

	for x := 0; x < spBounds.Dx(); x++ {
		for y := 0; y < spBounds.Dy(); y++ {
			spLabel := this.LabelMap.Gray16At(x, y).Y
			spPixelCount[spLabel]++
			if groundTruth.Gray16At(x, y).Y > 0 {
				spOverlapCount[spLabel]++
			}
		}
	}

	// Determine which superpixels are included in the mask based on the threshold
	for spLabel := 0; spLabel < this.SpCount; spLabel++ {
		if spPixelCount[spLabel] == 0 {
			continue
		}
		overlapRatio := float64(spOverlapCount[spLabel]) / float64(spPixelCount[spLabel])
		overlapHistogram[int(overlapRatio*100)]++
		if overlapRatio >= threshold {
			for x := 0; x < spBounds.Dx(); x++ {
				for y := 0; y < spBounds.Dy(); y++ {
					if this.LabelMap.Gray16At(x, y).Y == uint16(spLabel) {
						spMask.SetGray16(x, y, color.Gray16{Y: 65535})
					}
				}
			}
		}
	}

	return spMask, overlapHistogram
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
