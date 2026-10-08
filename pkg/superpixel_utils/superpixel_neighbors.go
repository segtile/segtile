package superpixel_utils

import (
	"image"
)

// LabelMapNeighborhoodsWeighted computes, for each label in a label map, which other labels
// share a border with it and how long that shared border is (in pixels).
func LabelMapNeighborhoodsWeighted(labelMap *image.Gray16, labelCount int) [][]LabelBorder {
	bounds := labelMap.Bounds()
	intBorders := labelNeighborhoodsWeighted(bounds, labelCount, func(x, y int) int {
		return int(UnsafeGray16At(labelMap, x, y))
	})
	borders := make([][]LabelBorder, labelCount)
	for label, neighbors := range intBorders {
		borders[label] = make([]LabelBorder, 0, len(neighbors))
		for _, neighbor := range neighbors {
			borders[label] = append(borders[label], LabelBorder{
				Label:              uint16(neighbor.Label),
				SharedBorderLength: neighbor.SharedBorderLength,
			})
		}
	}
	return borders
}

// LabelMapNeighborhoodsWeightedInt computes weighted neighborhoods for a row-major integer label map.
func LabelMapNeighborhoodsWeightedInt(labelMap []int, bounds image.Rectangle, labelCount int) [][]IntLabelBorder {
	return labelNeighborhoodsWeighted(bounds, labelCount, func(x, y int) int {
		return labelMap[(y-bounds.Min.Y)*bounds.Dx()+x-bounds.Min.X]
	})
}

type IntLabelBorder struct {
	Label              int
	SharedBorderLength int
}

func labelNeighborhoodsWeighted(bounds image.Rectangle, labelCount int, labelAt func(x, y int) int) [][]IntLabelBorder {
	// Measure the shared border length between each pair of neighboring labels.
	// Only right and bottom neighbors are checked, so every undirected pair is counted exactly once.
	borderLengths := make([]map[int]int, labelCount)
	for label := range borderLengths {
		borderLengths[label] = make(map[int]int)
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			currentLabel := labelAt(x, y)
			if x+1 < bounds.Max.X {
				if neighborLabel := labelAt(x+1, y); neighborLabel != currentLabel {
					borderLengths[currentLabel][neighborLabel]++
					borderLengths[neighborLabel][currentLabel]++
				}
			}
			if y+1 < bounds.Max.Y {
				if neighborLabel := labelAt(x, y+1); neighborLabel != currentLabel {
					borderLengths[currentLabel][neighborLabel]++
					borderLengths[neighborLabel][currentLabel]++
				}
			}
		}
	}

	// Flatten the per-label maps into border slices.
	borders := make([][]IntLabelBorder, labelCount)
	for label, lengths := range borderLengths {
		borders[label] = make([]IntLabelBorder, 0, len(lengths))
		for neighborLabel, length := range lengths {
			borders[label] = append(borders[label], IntLabelBorder{Label: neighborLabel, SharedBorderLength: length})
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
