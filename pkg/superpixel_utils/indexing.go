package superpixel_utils

import (
	"sort"

	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

// EnforceContiguousIndexing ensures that the superpixel labels are contiguous and start from 0.
// If the labels are not contiguous, they are remapped to be contiguous, and SpCount is updated accordingly.
// If the labels are already contiguous, no changes are made.
// Returns true if any changes were made to the label map.
func EnforceContiguousIndexing(labelMap *superpixel_map.LabelMap) (changed bool) {

	// Get the actual number of superpixels from the map
	highestLabel := labelMap.HighestLabel()
	labelUsed := labelMap.LabelUsed()
	labelCount := len(labelUsed)

	// If the labels are not contiguous, they need to be remapped.
	if labelCount != int(highestLabel)+1 {

		sortedLabels := make([]uint32, 0, len(labelUsed))
		for label := range labelUsed {
			sortedLabels = append(sortedLabels, label)
		}
		sort.Slice(sortedLabels, func(i, j int) bool { return sortedLabels[i] < sortedLabels[j] })

		labelRemap := make(map[uint32]uint32, len(sortedLabels))
		for newLabel, oldLabel := range sortedLabels {
			labelRemap[oldLabel] = uint32(newLabel)
		}
		newLabel := len(sortedLabels)

		// Remap the labels in the label map
		for x := 0; x < labelMap.Width; x++ {
			for y := 0; y < labelMap.Height; y++ {
				oldLabel := labelMap.Labels[y*labelMap.Width+x]
				labelMap.SetLabelAt(x, y, uint32(labelRemap[oldLabel]))
			}
		}
		labelCount = newLabel
	}

	return labelCount != int(highestLabel)+1
}
