package superpixel

import (
	"image"
	"image/color"
	"image/draw"
	"sort"

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
	b := this.LabelMap.Bounds()
	newLabelMap := image.NewGray16(b)
	draw.Draw(newLabelMap, b, this.LabelMap, b.Min, draw.Src)
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
	for x := 0; x < bounds.Max.X; x++ {
		for y := 0; y < bounds.Max.Y; y++ {
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

	// log.Println("Enforcing contiguous indexing ...")

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
		for x := 0; x < bounds.Max.X; x++ {
			for y := 0; y < bounds.Max.Y; y++ {
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

	//log.Println("Enforcing connectivity ...")

	// Prepare variables
	bounds := this.LabelMap.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Step 1: find every connected component in the label map.
	// componentId maps each pixel to its component id; componentLabel and componentSize
	// hold the label and pixel count of each component.
	componentId, componentLabel, componentSize := superpixel_utils.FindConnectedComponents(this.LabelMap)
	numComponents := len(componentLabel)

	// Step 2: build a component-level relationship graph, weighted by shared border length.
	// Aggregate undirected edges once, then store them in compact adjacency arrays.
	edgeWeights := make(map[uint64]int, numComponents*2)
	for y := 0; y < height; y++ {
		rowStart := y * width
		for x := 0; x < width; x++ {
			id := componentId[rowStart+x]
			if x+1 < width {
				neighborId := componentId[rowStart+x+1]
				if id != neighborId {
					a, b := id, neighborId
					if a > b {
						a, b = b, a
					}
					edgeWeights[uint64(a)<<32|uint64(uint32(b))]++
				}
			}
			if y+1 < height {
				neighborId := componentId[rowStart+x+width]
				if id != neighborId {
					a, b := id, neighborId
					if a > b {
						a, b = b, a
					}
					edgeWeights[uint64(a)<<32|uint64(uint32(b))]++
				}
			}
		}
	}

	adjacencyDegree := make([]int, numComponents)
	for edge := range edgeWeights {
		adjacencyDegree[int(edge>>32)]++
		adjacencyDegree[int(uint32(edge))]++
	}
	adjacencyOffset := make([]int, numComponents+1)
	for id := 0; id < numComponents; id++ {
		adjacencyOffset[id+1] = adjacencyOffset[id] + adjacencyDegree[id]
	}
	adjacencyNeighbor := make([]int32, adjacencyOffset[numComponents])
	adjacencyWeight := make([]int, adjacencyOffset[numComponents])
	adjacencyCursor := make([]int, numComponents)
	copy(adjacencyCursor, adjacencyOffset[:numComponents])
	for edge, weight := range edgeWeights {
		a := int32(edge >> 32)
		b := int32(uint32(edge))

		position := adjacencyCursor[a]
		adjacencyNeighbor[position] = b
		adjacencyWeight[position] = weight
		adjacencyCursor[a]++

		position = adjacencyCursor[b]
		adjacencyNeighbor[position] = a
		adjacencyWeight[position] = weight
		adjacencyCursor[b]++
	}

	// Step 3: the largest component of each label survives with that label; every other
	// component sharing the label is a loser that must be merged elsewhere.
	winnerOfLabel := make(map[uint16]int32, numComponents)
	for id := 0; id < numComponents; id++ {
		label := componentLabel[id]
		if currentWinner, ok := winnerOfLabel[label]; !ok || componentSize[id] > componentSize[currentWinner] {
			winnerOfLabel[label] = int32(id)
		}
	}
	finalLabel := make([]uint16, numComponents)
	finalized := make([]bool, numComponents)
	pending := make([]int32, 0, numComponents)
	for id := 0; id < numComponents; id++ {
		if winnerOfLabel[componentLabel[id]] == int32(id) {
			finalLabel[id] = componentLabel[id]
			finalized[id] = true
		} else {
			pending = append(pending, int32(id))
		}
	}

	// Step 4: repeatedly merge every pending component that borders an already-finalized
	// one into whichever finalized neighbor shares the longest border (ties broken by the
	// lowest label, for determinism). Once a component is finalized it is never revisited.
	for len(pending) > 0 {
		stillPending := pending[:0]
		progressed := false
		for _, id := range pending {
			bestWeight := -1
			bestTarget := int32(-1)
			for position := adjacencyOffset[id]; position < adjacencyOffset[id+1]; position++ {
				neighborId := adjacencyNeighbor[position]
				if !finalized[neighborId] {
					continue
				}
				weight := adjacencyWeight[position]
				if weight > bestWeight || (weight == bestWeight && bestTarget != -1 && finalLabel[neighborId] < finalLabel[bestTarget]) {
					bestWeight = weight
					bestTarget = neighborId
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

		// A group of minority components that only border each other has no finalized neighbor to attach to.
		// Break the deadlock deterministically by letting the largest of them survive with its own label, then keep resolving the rest.
		if !progressed {
			largest := stillPending[0]
			for _, id := range stillPending[1:] {
				if componentSize[id] > componentSize[largest] {
					largest = id
				}
			}
			finalLabel[largest] = componentLabel[largest]
			finalized[largest] = true
			remaining := stillPending[:0]
			for _, id := range stillPending {
				if id != largest {
					remaining = append(remaining, id)
				}
			}
			stillPending = remaining
		}

		pending = stillPending
	}

	// Step 5: put the resolved labels back to the label map.
	changed = false
	for y := 0; y < height; y++ {
		rowStart := y * width
		for x := 0; x < width; x++ {
			id := componentId[rowStart+x]
			newLabel := finalLabel[id]
			if newLabel == componentLabel[id] {
				continue
			}
			changed = true
			offset := y*this.LabelMap.Stride + x*2
			this.LabelMap.Pix[offset] = byte(newLabel >> 8)
			this.LabelMap.Pix[offset+1] = byte(newLabel)
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
	for x := spBounds.Min.X; x < spBounds.Max.X; x++ {
		for y := spBounds.Min.Y; y < spBounds.Max.Y; y++ {
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

	for x := spBounds.Min.X; x < spBounds.Max.X; x++ {
		for y := spBounds.Min.Y; y < spBounds.Max.Y; y++ {
			if selection[this.LabelMap.Gray16At(x, y).Y] {
				mask.SetGray16(x, y, color.Gray16{Y: 65535})
			}
		}
	}
	return mask
}

func (this *SpResult) IndexToMask(spId uint16) *image.Gray16 {
	spBounds := this.LabelMap.Bounds()
	mask := image.NewGray16(spBounds)

	for x := spBounds.Min.X; x < spBounds.Max.X; x++ {
		for y := spBounds.Min.Y; y < spBounds.Max.Y; y++ {
			if this.LabelMap.Gray16At(x, y).Y == spId {
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
