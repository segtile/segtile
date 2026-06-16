package superpixel_interaction

import (
	"image"

	"github.com/struffel/annotation-tool/internal/evaluation"
)

// When clicking on an unselected superpixel, add all connected and still unselected superpixels to the selection
func SuperpixelFloodFillSelection(labelMap *image.Gray16, x, y int, selection []bool) (newSelection []bool) {
	targetLabel := labelMap.Gray16At(x, y).Y
	if targetLabel == 0 || selection[targetLabel] {
		return selection
	}
	neighbors := evaluation.LabelMapNeighborhoods(labelMap, len(selection))
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
