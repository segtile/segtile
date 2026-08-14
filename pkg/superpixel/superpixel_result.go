package superpixel

import (
	"image"
	"image/color"
	"sort"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// SpResult encapsulates the result of a superpixel segmentation, including the label map and the count of superpixels.
// LabelMap is a grayscale image where each pixel's value corresponds to the superpixel label it belongs to. Labels should be contiguous and start from 0.
// SpCount is the total number of superpixels in the label map.
type SpResult struct {
	LabelMap                   *image.Gray16
	SpCount                    int // Number of superpixels. This should be the actual number of SP in the final map.
	connectivityEnforced       bool
	contiguousIndexingEnforced bool
}

// NewSpResult creates a new SpResult with the given label map and superpixel count.
// If optimize is true, it will enforce connectivity and contiguous indexing on the label map and recalculate SpCount.
func NewSpResult(labelMap *image.Gray16, spCount int, optimize bool) SpResult {
	result := SpResult{
		LabelMap:                   labelMap,
		SpCount:                    spCount,
		connectivityEnforced:       false,
		contiguousIndexingEnforced: false,
	}
	if optimize {
		result.EnforceConnectivity()
		result.EnforceContiguousIndexing()
		result.SetActualSpCount() // Recalculate SpCount after enforcing connectivity and contiguous indexing
	}
	return result
}

// Clone creates a deep copy of the result, including a newly allocated copy of the label map.
func (this *SpResult) Clone() SpResult {
	newLabelMap := io_utils.ToGray16(this.LabelMap)
	return SpResult{
		LabelMap:                   newLabelMap,
		SpCount:                    this.SpCount,
		connectivityEnforced:       this.connectivityEnforced,
		contiguousIndexingEnforced: this.contiguousIndexingEnforced,
	}
}

func (this *SpResult) SuperpixelCount() int {
	return this.SpCount
}

func (this *SpResult) ConnectivityEnforced() bool {
	return this.connectivityEnforced
}

func (this *SpResult) ContiguousIndexingEnforced() bool {
	return this.contiguousIndexingEnforced
}

func (this *SpResult) analyzeLabelMap() (lowestLabel uint16, highestLabel uint16, labelUsed map[uint16]bool) {

	labelUsed = make(map[uint16]bool)
	lowestLabel = ^uint16(0) // max uint16
	highestLabel = uint16(0)

	bounds := this.LabelMap.Bounds()
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			label := this.LabelMap.Gray16At(x, y).Y

			labelUsed[label] = true
			if label > highestLabel {
				highestLabel = label
			}
			if label < lowestLabel {
				lowestLabel = label
			}
		}
	}

	return lowestLabel, highestLabel, labelUsed
}

func (this *SpResult) SetActualSpCount() {
	_, _, labelUsed := this.analyzeLabelMap()
	this.SpCount = len(labelUsed)
}

// EnforceContiguousIndexing ensures that the superpixel labels are contiguous and start from 0.
// If the labels are not contiguous, they are remapped to be contiguous, and SpCount is updated accordingly.
// If the labels are already contiguous, no changes are made.
// Returns true if any changes were made to the label map.
func (this *SpResult) EnforceContiguousIndexing() (changed bool) {

	if this.contiguousIndexingEnforced {
		return false
	}

	bounds := this.LabelMap.Bounds()

	// Get the actual number of superpixels from the map
	_, highestLabel, labelUsed := this.analyzeLabelMap()
	spCount := len(labelUsed)

	if spCount != int(highestLabel)+1 {
		sortedLabels := make([]uint16, 0, len(labelUsed))
		for label := range labelUsed {
			sortedLabels = append(sortedLabels, label)
		}
		sort.Slice(sortedLabels, func(i, j int) bool { return sortedLabels[i] < sortedLabels[j] })

		labelRemap := make(map[uint16]uint16, len(sortedLabels))
		for newLabel, oldLabel := range sortedLabels {
			labelRemap[oldLabel] = uint16(newLabel)
		}
		newLabel := len(sortedLabels)
		// Remap the labels in the label map
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
				oldLabel := this.LabelMap.Gray16At(x, y).Y
				this.LabelMap.SetGray16(x, y, color.Gray16{Y: labelRemap[oldLabel]})
			}
		}
		spCount = newLabel
	}

	this.contiguousIndexingEnforced = true

	return spCount != int(highestLabel)+1
}

// EnforceConnectivity ensures that every label forms a single 4-connected component.
// If a label appears in multiple disjoint components, the largest component keeps the
// label, and every smaller component is merged into the label most common along its border.
// Labels are renumbered to remain contiguous afterwards.
func (this *SpResult) EnforceConnectivity() (changed bool) {

	// Skip if connectivity has already been enforced
	if this.connectivityEnforced {
		return false
	}

	// Prepare variables
	bounds := this.LabelMap.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Step 1: find every connected component in the label map.

	// ComponentId maps each pixel to its component id. -1 means unvisited.
	componentId := make([]int, width*height)
	for i := range componentId {
		componentId[i] = -1
	}

	// Each component has a label, size, and list of pixel indices.
	var componentLabel []uint16
	var componentSize []int
	var componentPixels [][]int

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			startIdx := y*width + x
			if componentId[startIdx] != -1 {
				continue
			}
			label := this.LabelMap.Gray16At(bounds.Min.X+x, bounds.Min.Y+y).Y
			id := len(componentLabel)
			componentId[startIdx] = id
			pixels := []int{startIdx}
			queue := []int{startIdx}
			for len(queue) > 0 {
				cur := queue[0]
				queue = queue[1:]
				cx, cy := cur%width, cur/width
				for _, nb := range superpixel_utils.Neighbors(image.Point{X: bounds.Min.X + cx, Y: bounds.Min.Y + cy}, bounds, false) {
					nIdx := (nb.Y-bounds.Min.Y)*width + (nb.X - bounds.Min.X)
					if this.LabelMap.Gray16At(nb.X, nb.Y).Y != label || componentId[nIdx] != -1 {
						continue
					}
					componentId[nIdx] = id
					pixels = append(pixels, nIdx)
					queue = append(queue, nIdx)
				}
			}
			componentLabel = append(componentLabel, label)
			componentSize = append(componentSize, len(pixels))
			componentPixels = append(componentPixels, pixels)
		}
	}
	numComponents := len(componentLabel)

	// Step 2: build a component-level adjacency graph, weighted by shared border length.
	neighborBorder := make([]map[int]int, numComponents)
	for i := range neighborBorder {
		neighborBorder[i] = make(map[int]int)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			id := componentId[y*width+x]
			for _, nb := range superpixel_utils.Neighbors(image.Point{X: bounds.Min.X + x, Y: bounds.Min.Y + y}, bounds, false) {
				nIdx := (nb.Y-bounds.Min.Y)*width + (nb.X - bounds.Min.X)
				if nid := componentId[nIdx]; nid != id {
					neighborBorder[id][nid]++
				}
			}
		}
	}

	// Step 3: the largest component of each label survives with that label; every other
	// component sharing the label is a loser that must be merged elsewhere.
	winnerOfLabel := make(map[uint16]int, numComponents)
	for id := 0; id < numComponents; id++ {
		lbl := componentLabel[id]
		if cur, ok := winnerOfLabel[lbl]; !ok || componentSize[id] > componentSize[cur] {
			winnerOfLabel[lbl] = id
		}
	}
	finalLabel := make([]uint16, numComponents)
	finalized := make([]bool, numComponents)
	pending := make([]int, 0, numComponents)
	for id := 0; id < numComponents; id++ {
		if winnerOfLabel[componentLabel[id]] == id {
			finalLabel[id] = componentLabel[id]
			finalized[id] = true
		} else {
			pending = append(pending, id)
		}
	}

	// Step 4: repeatedly merge every pending component that borders an already-finalized
	// one into whichever finalized neighbor shares the longest border (ties broken by the
	// lowest label, for determinism). Once a component is finalized it is never revisited.
	for len(pending) > 0 {
		stillPending := pending[:0:0]
		progressed := false
		for _, id := range pending {
			bestWeight := -1
			bestTarget := -1
			for nid, weight := range neighborBorder[id] {
				if !finalized[nid] {
					continue
				}
				if weight > bestWeight || (weight == bestWeight && bestTarget != -1 && finalLabel[nid] < finalLabel[bestTarget]) {
					bestWeight = weight
					bestTarget = nid
				}
			}
			if bestTarget == -1 {
				stillPending = append(stillPending, id)
				continue
			}
			finalLabel[id] = finalLabel[bestTarget]
			finalized[id] = true
			progressed = true
		}

		// A group of minority components that only border each other has no finalized
		// neighbor to attach to. Break the deadlock deterministically by letting the
		// largest of them survive with its own label, then keep resolving the rest.
		if !progressed {
			largest := stillPending[0]
			for _, id := range stillPending[1:] {
				if componentSize[id] > componentSize[largest] {
					largest = id
				}
			}
			finalLabel[largest] = componentLabel[largest]
			finalized[largest] = true
			remaining := stillPending[:0:0]
			for _, id := range stillPending {
				if id != largest {
					remaining = append(remaining, id)
				}
			}
			stillPending = remaining
		}

		pending = stillPending
	}

	// Step 5: write the resolved labels back to the label map.
	changed = false
	for id := 0; id < numComponents; id++ {
		if finalLabel[id] == componentLabel[id] {
			continue
		}
		changed = true
		for _, idx := range componentPixels[id] {
			px, py := idx%width, idx/width
			this.LabelMap.SetGray16(bounds.Min.X+px, bounds.Min.Y+py, color.Gray16{Y: finalLabel[id]})
		}
	}

	this.connectivityEnforced = true
	this.contiguousIndexingEnforced = false

	return changed
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
