package superpixel

import (
	"image"
	"image/color"
	//"github.com/hashicorp/go-set/v3"
)

// SpResult encapsulates the result of a superpixel segmentation, including the label map and the count of superpixels.
// LabelMap is a grayscale image where each pixel's value corresponds to the superpixel label it belongs to. Labels should be contiguous and start from 0.
// SpCount is the total number of superpixels in the label map.
type SpResult struct {
	LabelMap *image.Gray16
	SpCount  int // Number of superpixels. This should be the actual number of SP in the final map.
}

// Constructs a SpResult from a given label map.
// Note that the label map will get updated if it is not consistent with the expected format (contiguousness).
func MakeSpResult(labelMap *image.Gray16) SpResult {

	labelPix := labelMap.Pix

	// Determine which label indices are actually used in the map
	labelUsed := make(map[uint16]bool)
	highestLabel := uint16(0)

	bounds := labelMap.Bounds()
	for x := 0; x < bounds.Dx(); x++ {
		for y := 0; y < bounds.Dy(); y++ {
			label := uint16(labelPix[y*labelMap.Stride+x*2]) | uint16(labelPix[y*labelMap.Stride+x*2+1])<<8

			labelUsed[label] = true
			if label > highestLabel {
				highestLabel = label
			}
		}
	}

	// Get the actual number of superpixels from the map
	spCount := len(labelUsed)

	// If the labels are not contiguous, we need to remap them to ensure they are.
	if spCount != int(highestLabel)+1 {
		labelRemap := make(map[uint16]uint16)
		newLabel := uint16(0)
		for label := uint16(0); label <= highestLabel; label++ {
			if labelUsed[label] {
				labelRemap[label] = newLabel
				newLabel++
			}
		}
		// Remap the labels in the label map
		for x := 0; x < bounds.Dx(); x++ {
			for y := 0; y < bounds.Dy(); y++ {
				oldLabel := uint16(labelPix[y*labelMap.Stride+x*2]) | uint16(labelPix[y*labelMap.Stride+x*2+1])<<8
				//oldLabel := labelMap.Gray16At(x, y).Y
				labelMap.SetGray16(x, y, color.Gray16{Y: labelRemap[oldLabel]})
			}
		}
		spCount = int(newLabel)
	}

	return SpResult{
		LabelMap: labelMap,
		SpCount:  spCount,
	}
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
// Returns:
// -
func (this *SpResult) ApproximateMask(groundTruth *image.Gray16, threshold float64) (approximatedMask *image.Gray16, overlapHistogram []int, totalSelectedSpCount int, borderingSpCount int) {
	if this.LabelMap.Bounds() != groundTruth.Bounds() {
		panic("Ground truth and superpixel label map must have the same dimensions")
	}

	// Build the overlap histogram

	spPixelCount, spOverlapCount := this.computeOverlapCounts(groundTruth)
	overlapHistogram = make([]int, 101)
	selection := make([]bool, this.SpCount)

	for spLabel := 0; spLabel < this.SpCount; spLabel++ {
		if spPixelCount[spLabel] == 0 {
			continue
		}
		overlapRatio := float64(spOverlapCount[spLabel]) / float64(spPixelCount[spLabel])
		overlapHistogram[int(overlapRatio*100)]++
		if overlapRatio >= threshold {
			selection[spLabel] = true
			totalSelectedSpCount++
		}
	}

	// Use the final approximated mask to count how many superpixels are border SPs and how many are contained SPs
	approximatedMask = this.SelectionToMask(selection)

	bounds := approximatedMask.Bounds()
	bX := bounds.Max.X
	bY := bounds.Max.Y

	borderingSpSelection := make([]bool, this.SpCount)

	for x := 0; x < approximatedMask.Bounds().Dx(); x++ {
		for y := 0; y < approximatedMask.Bounds().Dy(); y++ {
			isBorderPixel := false
			if approximatedMask.Gray16At(x, y).Y > 0 {
				// Check 4-neighborhood for background pixels
				if x > 0 && x < bX-1 && y > 0 && y < bY-1 {
					if approximatedMask.Gray16At(x-1, y).Y == 0 || approximatedMask.Gray16At(x+1, y).Y == 0 || approximatedMask.Gray16At(x, y-1).Y == 0 || approximatedMask.Gray16At(x, y+1).Y == 0 {
						isBorderPixel = true
					}
				}
			}
			if isBorderPixel {
				borderingSpSelection[this.LabelMap.Gray16At(x, y).Y] = true
			}
		}
	}

	for i := 0; i < this.SpCount; i++ {
		if borderingSpSelection[i] {
			borderingSpCount++
		}
	}

	return approximatedMask, overlapHistogram, totalSelectedSpCount, borderingSpCount
}
