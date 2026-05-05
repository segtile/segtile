package superpixel

import (
	"image"
	"image/color"
)

type SpResult struct {
	LabelMap *image.Gray16 // Image where each pixel's value is the superpixel label
	SpCount  int           // Number of superpixels
}

// computeOverlapCounts counts, for each superpixel, the total number of pixels it contains
// and how many of those pixels are non-zero in mask.
func (this *SpResult) computeOverlapCounts(mask *image.Gray16) (spPixelCount []int, spOverlapCount []int) {
	spBounds := this.LabelMap.Bounds()
	spPixelCount = make([]int, this.SpCount)
	spOverlapCount = make([]int, this.SpCount)
	for x := 0; x < spBounds.Dx(); x++ {
		for y := 0; y < spBounds.Dy(); y++ {
			spLabel := this.LabelMap.Gray16At(x, y).Y
			spPixelCount[spLabel]++
			if mask.Gray16At(x, y).Y > 0 {
				spOverlapCount[spLabel]++
			}
		}
	}
	return
}

// SelectionToMask converts a boolean selection slice into a binary Gray16 mask image.
// Pixels belonging to selected superpixels are set to white (65535); all others remain black (0).
func (this *SpResult) SelectionToMask(selection []bool) *image.Gray16 {
	spBounds := this.LabelMap.Bounds()
	mask := image.NewGray16(spBounds)
	for x := 0; x < spBounds.Dx(); x++ {
		for y := 0; y < spBounds.Dy(); y++ {
			if selection[this.LabelMap.Gray16At(x, y).Y] {
				mask.SetGray16(x, y, color.Gray16{Y: 65535})
			}
		}
	}
	return mask
}

// SelectFromMask returns a boolean selection indicating which superpixels best approximate
// the given binary mask. A superpixel is selected when the fraction of its pixels that
// overlap the mask meets or exceeds threshold (e.g. 0.5 = 50%).
func (this *SpResult) SelectFromMask(mask *image.Gray16, threshold float64) []bool {
	if this.LabelMap.Bounds() != mask.Bounds() {
		panic("Mask and superpixel label map must have the same dimensions")
	}
	spPixelCount, spOverlapCount := this.computeOverlapCounts(mask)
	selection := make([]bool, this.SpCount)
	for spLabel := 0; spLabel < this.SpCount; spLabel++ {
		if spPixelCount[spLabel] > 0 && float64(spOverlapCount[spLabel])/float64(spPixelCount[spLabel]) >= threshold {
			selection[spLabel] = true
		}
	}
	return selection
}

// ApproximateMask creates a mask by approximating the ground truth with the superpixels.
// A superpixel is included in the mask if its overlap with the ground truth exceeds the given threshold.
// E.G. A threshold of 0.5 indicates that 50% of the SP's pixels must also be contained in the mask.
// Additionally returns the overlap ratios for each superpixel.
func (this *SpResult) ApproximateMask(groundTruth *image.Gray16, threshold float64) (*image.Gray16, []int) {
	if this.LabelMap.Bounds() != groundTruth.Bounds() {
		panic("Ground truth and superpixel label map must have the same dimensions")
	}

	spPixelCount, spOverlapCount := this.computeOverlapCounts(groundTruth)
	overlapHistogram := make([]int, 101)
	selection := make([]bool, this.SpCount)

	for spLabel := 0; spLabel < this.SpCount; spLabel++ {
		if spPixelCount[spLabel] == 0 {
			continue
		}
		overlapRatio := float64(spOverlapCount[spLabel]) / float64(spPixelCount[spLabel])
		overlapHistogram[int(overlapRatio*100)]++
		if overlapRatio >= threshold {
			selection[spLabel] = true
		}
	}

	return this.SelectionToMask(selection), overlapHistogram
}
