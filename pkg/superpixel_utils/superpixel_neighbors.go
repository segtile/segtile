package superpixel_utils

import (
	"image"
)

// LabelMapNeighborhoodsWeighted computes, for each label in a label map, which other labels
// share a border with it and how long that shared border is (in pixels).
func LabelMapNeighborhoodsWeighted(labelMap *image.Gray16, labelCount int) [][]LabelBorder {

	bounds := labelMap.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Measure the shared border length between each pair of neighboring labels.
	// Only right and bottom neighbors are checked, so every undirected pair is counted exactly once.
	borderLengths := make([]map[uint16]int, labelCount)
	for label := range borderLengths {
		borderLengths[label] = make(map[uint16]int)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			currentLabel := UnsafeGray16At(labelMap, x, y)
			if x+1 < width {
				if neighborLabel := UnsafeGray16At(labelMap, x+1, y); neighborLabel != currentLabel {
					borderLengths[currentLabel][neighborLabel]++
					borderLengths[neighborLabel][currentLabel]++
				}
			}
			if y+1 < height {
				if neighborLabel := UnsafeGray16At(labelMap, x, y+1); neighborLabel != currentLabel {
					borderLengths[currentLabel][neighborLabel]++
					borderLengths[neighborLabel][currentLabel]++
				}
			}
		}
	}

	// Flatten the per-label maps into border slices.
	borders := make([][]LabelBorder, labelCount)
	for label, lengths := range borderLengths {
		borders[label] = make([]LabelBorder, 0, len(lengths))
		for neighborLabel, length := range lengths {
			borders[label] = append(borders[label], LabelBorder{Label: neighborLabel, SharedBorderLength: length})
		}
	}
	return borders
}

// LabelBorder describes one neighboring label and the length of the border shared with it.
type LabelBorder struct {
	Label              uint16
	SharedBorderLength int
}

// Computes for each superpixel which other superpixels are its neighbors (i.e. share a border with it)
func LabelMapNeighborhoods(labelMap *image.Gray16, spCount int) [][]uint16 {

	borders := LabelMapNeighborhoodsWeighted(labelMap, spCount)
	neighbors := make([][]uint16, spCount)
	for label, labelBorders := range borders {
		for _, border := range labelBorders {
			neighbors[label] = append(neighbors[label], border.Label)
		}
	}
	return neighbors
}

// Returns a new selection which includes all SP from the old selection, but only if they border at least one unselected SP (i.e. they are on the border of the selection)
func SelectBorderingSuperpixels(labelMap *image.Gray16, selection []bool) []bool {
	neighbors := LabelMapNeighborhoods(labelMap, len(selection))
	newSelection := make([]bool, len(selection))
	//copy(newSelection, selection)
	for spLabel, selected := range selection {
		if selected {
			for _, neighbor := range neighbors[spLabel] {
				if !selection[neighbor] {
					newSelection[spLabel] = true
					break
				}
			}
		}
	}
	return newSelection
}

func CountBorderingSuperpixels(labelMap *image.Gray16, selection []bool) (selectedSpCount int, selectedBorderingSpCount int) {

	neighbors := LabelMapNeighborhoods(labelMap, len(selection))

	for spLabel, selected := range selection {
		if selected {
			selectedSpCount++
			for _, neighbor := range neighbors[spLabel] {
				if !selection[neighbor] {
					selectedBorderingSpCount++
					break
				}
			}
		}

	}
	return selectedSpCount, selectedBorderingSpCount
}

// When clicking on an unselected superpixel, add all connected and still unselected superpixels to the selection
func SuperpixelFloodFillFromPoint(labelMap *image.Gray16, x, y int, selection []bool) (newSelection []bool) {
	targetLabel := labelMap.Gray16At(x, y).Y
	if targetLabel == 0 || selection[targetLabel] {
		return selection
	}
	neighbors := LabelMapNeighborhoods(labelMap, len(selection))
	queue := []uint16{targetLabel}
	newSelection = make([]bool, len(selection))
	copy(newSelection, selection)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if !newSelection[current] {
			newSelection[current] = true
			queue = append(queue, neighbors[current]...)
		}
	}
	return newSelection
}

// Selects all superpixels that are enclosed by selected superpixels (i.e. they are not connected to the edge of the image)
// This is done by performing a flood fill from all unselected superpixels touching the edge of the image and marking
// all reachable superpixels as unselected, and selecting all other superpixels.
func SuperpixelAutoFloodFillSelection(labelMap *image.Gray16, selection []bool) (newSelection []bool, changed bool) {

	bounds := labelMap.Bounds()
	changed = false

	// Collect every unselected label touching the edge of the image.
	seeds := make(map[uint16]struct{})
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		if label := labelMap.Gray16At(bounds.Min.X, y).Y; !selection[label] {
			seeds[label] = struct{}{}
		}
		if label := labelMap.Gray16At(bounds.Max.X-1, y).Y; !selection[label] {
			seeds[label] = struct{}{}
		}
	}
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		if label := labelMap.Gray16At(x, bounds.Min.Y).Y; !selection[label] {
			seeds[label] = struct{}{}
		}
		if label := labelMap.Gray16At(x, bounds.Max.Y-1).Y; !selection[label] {
			seeds[label] = struct{}{}
		}
	}
	if len(seeds) == 0 {
		return selection, false
	}

	// Flood fill from all border seeds
	neighbors := LabelMapNeighborhoods(labelMap, len(selection))
	newSelection = make([]bool, len(selection))
	copy(newSelection, selection)
	queue := make([]uint16, 0, len(seeds))
	for label := range seeds {
		queue = append(queue, label)
	}
	for len(queue) > 0 {

		// Dequeue
		current := queue[0]
		queue = queue[1:]

		// If this superpixel has not been marked as reachable yet, mark it and enqueue all its neighbors
		if !newSelection[current] {
			newSelection[current] = true
			queue = append(queue, neighbors[current]...)
		}
	}

	// Mark all reachable superpixels as unselected and all unreachable superpixels (plus the original selection) as selected
	for i := 0; i < len(newSelection); i++ {
		newSelection[i] = !newSelection[i] || selection[i]
	}

	// Compare old and new selection to see if anything changed
	for i := 0; i < len(selection); i++ {
		if selection[i] != newSelection[i] {
			changed = true
			break
		}
	}

	return newSelection, changed

}
