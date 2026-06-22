package f1_score

import (
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// Returns a new selection which includes all SP from the old selection, but only if they border at least one unselected SP (i.e. they are on the border of the selection)
func SelectBorderingSuperpixels(labelMap *image.Gray16, selection []bool) []bool {
	neighbors := superpixel_utils.LabelMapNeighborhoods(labelMap, len(selection))
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

	neighbors := superpixel_utils.LabelMapNeighborhoods(labelMap, len(selection))

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
