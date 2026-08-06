package superpixel_utils

import "image"

// Computes for each superpixel which other superpixels are its neighbors (i.e. share a border with it)
func LabelMapNeighborhoods(labelMap *image.Gray16, spCount int) [][]uint16 {

	bounds := labelMap.Bounds()
	neighbors := make([][]uint16, spCount)
	seen := make([]map[uint16]struct{}, spCount)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			currentLabel := labelMap.Gray16At(x, y).Y
			if currentLabel == 0 {
				continue
			}
			if seen[currentLabel] == nil {
				seen[currentLabel] = make(map[uint16]struct{})
			}

			// Check 4-neighborhoods
			for _, nb := range Neighbors(image.Point{X: x, Y: y}, bounds, false) {
				neighborLabel := labelMap.Gray16At(nb.X, nb.Y).Y
				if neighborLabel != 0 && neighborLabel != currentLabel {
					if _, exists := seen[currentLabel][neighborLabel]; !exists {
						seen[currentLabel][neighborLabel] = struct{}{}
						neighbors[currentLabel] = append(neighbors[currentLabel], neighborLabel)
					}
				}
			}
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
func SuperpixelAutoFloodFillSelection(labelMap *image.Gray16, selection []bool) (newSelection []bool) {

	bounds := labelMap.Bounds()

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
		return selection
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

	return newSelection

}
