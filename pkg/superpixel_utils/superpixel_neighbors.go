package superpixel_utils

import (
	"github.com/segtile/segtile/pkg/superpixel_map"
)

// labelMapNeighborhoodsWeighted computes, for each label in a label map, which other labels
// share a border with it and how long that shared border is (in pixels).
// The result is a map: from -> to -> shared border length (in pixels)
func LabelMapNeighborhoodsWeighted(labelMap *superpixel_map.LabelMap) map[uint32]map[uint32]int {
	width, height := labelMap.Width, labelMap.Height
	labels := labelMap.Labels
	labelCount := 0
	for _, l := range labels {
		if int(l)+1 > labelCount {
			labelCount = int(l) + 1
		}
	}

	// Measure the shared border length between each pair of neighboring labels.
	// Only right and bottom neighbors are checked, so every undirected pair is counted exactly once.
	borderLengths := make([]map[int]int, labelCount)
	for label := range borderLengths {
		borderLengths[label] = make(map[int]int)
	}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			currentLabel := int(labels[y*width+x])
			if x+1 < width {
				if neighborLabel := int(labels[y*width+x+1]); neighborLabel != currentLabel {
					borderLengths[currentLabel][neighborLabel]++
					borderLengths[neighborLabel][currentLabel]++
				}
			}
			if y+1 < height {
				if neighborLabel := int(labels[(y+1)*width+x]); neighborLabel != currentLabel {
					borderLengths[currentLabel][neighborLabel]++
					borderLengths[neighborLabel][currentLabel]++
				}
			}
		}
	}

	// Convert the per-label maps into the final map format.
	borders := make(map[uint32]map[uint32]int, labelCount)
	for label, lengths := range borderLengths {
		borders[uint32(label)] = make(map[uint32]int, len(lengths))
		for neighborLabel, length := range lengths {
			borders[uint32(label)][uint32(neighborLabel)] = length
		}
	}
	return borders
}

// Computes for each superpixel which other superpixels are its neighbors (i.e. share a border with it)
// The result is a slice of slices: [from label] -> [neighbor labels]
func LabelMapNeighborhoods(labelMap *superpixel_map.LabelMap, spCount int) [][]uint32 {

	borders := LabelMapNeighborhoodsWeighted(labelMap)
	neighbors := make([][]uint32, spCount)
	for label, labelBorders := range borders {
		for neighborLabel := range labelBorders {
			neighbors[label] = append(neighbors[label], neighborLabel)
		}
	}
	return neighbors
}
