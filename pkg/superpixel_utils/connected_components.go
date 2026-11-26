package superpixel_utils

import (
	"github.com/segtile/segtile/pkg/superpixel_map"
)

// FindConnectedComponents finds all 4-connected components of pixels with equal values.
// The returned row-major map stores component id + 1 per pixel (0 means unvisited while
// the search is running); after the call every pixel holds a valid id in 1..componentCount.
func FindConnectedComponents(labelMap superpixel_map.Map) (componentIdMap *superpixel_map.LabelMap, componentValue []uint32, componentSize []int) {

	// Zero-initialized: 0 means unvisited, stored ids are shifted by +1.
	bounds := labelMap.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	componentIdMap = superpixel_map.NewLabelMap(width, height)

	componentValue = []uint32{}
	componentSize = []int{}
	queue := []int{}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			startIndex := y*width + x

			// Skip pixels that already belong to a component
			if componentIdMap.ValueAt(startIndex%width, startIndex/width) != 0 {
				continue
			}

			// Start a new connected component from this pixel
			value := labelMap.Uint32At(x, y)
			currentComponentId := uint32(len(componentValue) + 1)
			componentIdMap.SetValueAt(startIndex%width, startIndex/width, currentComponentId)

			queue = queue[:0]
			queue = append(queue, startIndex)
			size := 0

			for head := 0; head < len(queue); head++ {
				currentIndex := queue[head]
				size++

				currentX := currentIndex % width
				currentY := currentIndex / width

				// Left neighbor
				if currentX > 0 {
					neighborIndex := currentIndex - 1
					if componentIdMap.ValueAt(neighborIndex%width, neighborIndex/width) == 0 && labelMap.Uint32At(currentX-1, currentY) == value {
						componentIdMap.SetValueAt(neighborIndex%width, neighborIndex/width, currentComponentId)
						queue = append(queue, neighborIndex)
					}
				}
				// Right neighbor
				if currentX+1 < width {
					neighborIndex := currentIndex + 1
					if componentIdMap.ValueAt(neighborIndex%width, neighborIndex/width) == 0 && labelMap.Uint32At(currentX+1, currentY) == value {
						componentIdMap.SetValueAt(neighborIndex%width, neighborIndex/width, currentComponentId)
						queue = append(queue, neighborIndex)
					}
				}
				// Top neighbor
				if currentY > 0 {
					neighborIndex := currentIndex - width
					if componentIdMap.ValueAt(neighborIndex%width, neighborIndex/width) == 0 && labelMap.Uint32At(currentX, currentY-1) == value {
						componentIdMap.SetValueAt(neighborIndex%width, neighborIndex/width, currentComponentId)
						queue = append(queue, neighborIndex)
					}
				}
				// Bottom neighbor
				if currentY+1 < height {
					neighborIndex := currentIndex + width
					if componentIdMap.ValueAt(neighborIndex%width, neighborIndex/width) == 0 && labelMap.Uint32At(currentX, currentY+1) == value {
						componentIdMap.SetValueAt(neighborIndex%width, neighborIndex/width, currentComponentId)
						queue = append(queue, neighborIndex)
					}
				}
			}

			componentValue = append(componentValue, value)
			componentSize = append(componentSize, size)
		}
	}

	return componentIdMap, componentValue, componentSize
}

// SelectConnectedComponent creates a new mask where only the pixels belonging to the specified island label are set.
func SelectConnectedComponent(islandMap *superpixel_map.LabelMap, islandLabel uint32) *superpixel_map.Mask {
	bounds := islandMap.Bounds()
	selectedIsland := superpixel_map.NewMask(islandMap.Width, islandMap.Height)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if islandMap.ValueAt(x, y) == islandLabel {
				selectedIsland.SetValueAt(x, y, true)
			}
		}
	}
	return selectedIsland
}
