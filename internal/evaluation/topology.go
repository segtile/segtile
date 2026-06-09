package evaluation

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
			for _, offset := range []struct{ dx, dy int }{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+offset.dx, y+offset.dy
				if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
					neighborLabel := labelMap.Gray16At(nx, ny).Y
					if neighborLabel != 0 && neighborLabel != currentLabel {
						if _, exists := seen[currentLabel][neighborLabel]; !exists {
							seen[currentLabel][neighborLabel] = struct{}{}
							neighbors[currentLabel] = append(neighbors[currentLabel], neighborLabel)
						}
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
