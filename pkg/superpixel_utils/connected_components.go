package superpixel_utils

import (
	"image"
	"image/color"
)

// FindConnectedComponents finds all 4-connected components of pixels with equal values.
// The returned row-major map stores component id + 1 per pixel (0 means unvisited while
// the search is running); after the call every pixel holds a valid id in 1..componentCount.
func FindConnectedComponents(img *image.Gray16) (componentIdMap []int, componentValue []uint16, componentSize []int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Zero-initialized: 0 means unvisited, stored ids are shifted by +1.
	componentIdMap = make([]int, width*height)

	componentValue = []uint16{}
	componentSize = []int{}
	queue := []int{}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			startIndex := (y-bounds.Min.Y)*width + (x - bounds.Min.X)

			// Skip pixels that already belong to a component
			if componentIdMap[startIndex] != 0 {
				continue
			}

			// Start a new connected component from this pixel
			value := UnsafeGray16At(img, x, y)
			currentComponentId := len(componentValue) + 1
			componentIdMap[startIndex] = currentComponentId

			queue = queue[:0]
			queue = append(queue, startIndex)
			size := 0

			for head := 0; head < len(queue); head++ {
				currentIndex := queue[head]
				size++

				currentX := (currentIndex % width) + bounds.Min.X
				currentY := (currentIndex / width) + bounds.Min.Y

				// Left neighbor
				if currentX > bounds.Min.X {
					neighborIndex := currentIndex - 1
					if componentIdMap[neighborIndex] == 0 && UnsafeGray16At(img, currentX-1, currentY) == value {
						componentIdMap[neighborIndex] = currentComponentId
						queue = append(queue, neighborIndex)
					}
				}
				// Right neighbor
				if currentX+1 < bounds.Max.X {
					neighborIndex := currentIndex + 1
					if componentIdMap[neighborIndex] == 0 && UnsafeGray16At(img, currentX+1, currentY) == value {
						componentIdMap[neighborIndex] = currentComponentId
						queue = append(queue, neighborIndex)
					}
				}
				// Top neighbor
				if currentY > bounds.Min.Y {
					neighborIndex := currentIndex - width
					if componentIdMap[neighborIndex] == 0 && UnsafeGray16At(img, currentX, currentY-1) == value {
						componentIdMap[neighborIndex] = currentComponentId
						queue = append(queue, neighborIndex)
					}
				}
				// Bottom neighbor
				if currentY+1 < bounds.Max.Y {
					neighborIndex := currentIndex + width
					if componentIdMap[neighborIndex] == 0 && UnsafeGray16At(img, currentX, currentY+1) == value {
						componentIdMap[neighborIndex] = currentComponentId
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

// Identifies connected components of non-zero pixels in the mask and assigns each component a unique label in the output image.
// The value 0 is used as the background, 1 for the first island, 2 for the second, and so on.
// It also returns the total number of connected components (islands) found.
func FindIslands(mask *image.Gray16) (islandMap *image.Gray16, islandCount uint16) {

	islandMap = image.NewGray16(mask.Bounds())
	bounds := mask.Bounds()

	componentIdMap, componentValue, _ := FindConnectedComponents(mask)

	// Assign each foreground component a label
	islandLabels := make([]uint16, len(componentValue))
	nextLabel := uint16(1)
	for id, value := range componentValue {
		if value > 0 {
			islandLabels[id] = nextLabel
			nextLabel++
		}
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			componentIndex := (y-bounds.Min.Y)*bounds.Dx() + (x - bounds.Min.X)
			componentId := componentIdMap[componentIndex] - 1
			if islandLabel := islandLabels[componentId]; islandLabel > 0 {
				islandMap.SetGray16(x, y, color.Gray16{Y: islandLabel})
			}
		}
	}

	return islandMap, nextLabel - 1
}

// SelectIsland creates a new mask where only the pixels belonging to the specified island label are set.
func SelectIsland(islandMap *image.Gray16, islandLabel uint16) *image.Gray16 {
	bounds := islandMap.Bounds()
	selectedIsland := image.NewGray16(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if islandMap.Gray16At(x, y).Y == islandLabel {
				selectedIsland.SetGray16(x, y, color.Gray16{Y: 65535})
			}
		}
	}
	return selectedIsland
}
