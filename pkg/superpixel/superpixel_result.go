package superpixel

import (
	"sync"

	"github.com/struffel/annotation-tool/pkg/superpixel_map"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// SpResult encapsulates the result of a superpixel segmentation, including the label map and the count of superpixels.
// LabelMap is a grayscale image where each pixel's value corresponds to the superpixel label it belongs to. Labels should be contiguous and start from 0.
// SpCount is the total number of superpixels in the label map.
type SpResult struct {
	LabelMap      *superpixel_map.LabelMap
	SpCount       int // Number of superpixels. This should be the actual number of SP in the final map.
	BoundingBoxes map[uint32]*superpixel_utils.BoundingBox
}

func NewSpResult(labelMap *superpixel_map.LabelMap) SpResult {
	spCount := len(labelMap.LabelUsed())
	result := &SpResult{
		LabelMap:      labelMap,
		SpCount:       spCount,
		BoundingBoxes: map[uint32]*superpixel_utils.BoundingBox{},
	}
	result.Optimize()
	return *result
}

func (r *SpResult) Optimize() {
	// Fix issues with the label map
	superpixel_utils.EnforceConnectivity(r.LabelMap)
	superpixel_utils.EnforceContiguousIndexing(r.LabelMap)

	var wg sync.WaitGroup
	wg.Add(2)

	// Recalculate the superpixel count after enforcing connectivity and contiguous indexing
	go func() {
		defer wg.Done()
		labelUsed := r.LabelMap.LabelUsed()
		r.SpCount = len(labelUsed)
	}()

	// Recalculate the bounding boxes for each superpixel based on the updated label map.
	go func() {
		defer wg.Done()
		boundingBoxes := map[uint32]*superpixel_utils.BoundingBox{}
		for x := 0; x < r.LabelMap.Width; x++ {
			for y := 0; y < r.LabelMap.Height; y++ {
				label := r.LabelMap.ValueAt(x, y)
				if _, exists := boundingBoxes[label]; !exists {
					boundingBoxes[label] = &superpixel_utils.BoundingBox{}
				}
				boundingBoxes[label].AddPoint(x, y)
			}
		}
		r.BoundingBoxes = boundingBoxes
	}()

	wg.Wait()
}

// Clone creates a deep copy of the result, including a newly allocated copy of the label map.
func (r *SpResult) Clone() SpResult {
	newLabelMap := superpixel_map.NewLabelMap(r.LabelMap.Width, r.LabelMap.Height)
	copy(newLabelMap.Labels, r.LabelMap.Labels)
	return SpResult{
		LabelMap: newLabelMap,
		SpCount:  r.SpCount,
	}
}

// computeOverlapCounts counts, for each superpixel, the total number of pixels it contains
// and how many of those pixels are non-zero in mask.
func (r *SpResult) computeOverlapCounts(mask *superpixel_map.Mask) (spPixelCount []int, spOverlapCount []int) {

	spPixelCount = make([]int, r.SpCount)
	spOverlapCount = make([]int, r.SpCount)
	for x := 0; x < r.LabelMap.Width; x++ {
		for y := 0; y < r.LabelMap.Height; y++ {
			spLabel := r.LabelMap.ValueAt(x, y)
			spPixelCount[spLabel]++
			if mask.PositiveAt(x, y) {
				spOverlapCount[spLabel]++
			}
		}
	}
	return
}

// SelectionToMask converts a boolean selection slice into a binary mask.
// Pixels belonging to selected superpixels are set to true; all others remain false.
func (r *SpResult) SelectionToMask(selection []bool) *superpixel_map.Mask {

	mask := superpixel_map.NewMask(r.LabelMap.Width, r.LabelMap.Height)

	for x := 0; x < r.LabelMap.Width; x++ {
		for y := 0; y < r.LabelMap.Height; y++ {
			spLabel := r.LabelMap.ValueAt(x, y)
			if selection[spLabel] {
				mask.SetValueAt(x, y, true)
			}
		}
	}

	return mask
}

func (m *SpResult) IndexToMask(spId uint32) *superpixel_map.Mask {
	mask := superpixel_map.NewMask(m.LabelMap.Width, m.LabelMap.Height)

	for x := 0; x < m.LabelMap.Width; x++ {
		for y := 0; y < m.LabelMap.Height; y++ {
			if m.LabelMap.ValueAt(x, y) == spId {
				mask.SetValueAt(x, y, true)
			}
		}
	}
	return mask
}

// SelectFromMask returns a boolean selection indicating which superpixels best approximate
// the given binary mask. A superpixel is selected when the fraction of its pixels that
// overlap the mask meets or exceeds threshold (e.g. 0.5 = 50%).
func (m *SpResult) SelectFromMask(mask *superpixel_map.Mask, threshold float64) []bool {
	if m.LabelMap.Width != mask.Width || m.LabelMap.Height != mask.Height {
		panic("Mask and superpixel label map must have the same dimensions")
	}
	spPixelCount, spOverlapCount := m.computeOverlapCounts(mask)
	selection := make([]bool, m.SpCount)
	for spLabel := 0; spLabel < m.SpCount; spLabel++ {
		if spPixelCount[spLabel] > 0 && float64(spOverlapCount[spLabel])/float64(spPixelCount[spLabel]) > threshold {
			selection[spLabel] = true
		}
	}
	return selection
}
