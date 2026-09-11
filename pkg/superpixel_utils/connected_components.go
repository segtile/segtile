package superpixel_utils

import (
	"image"
	"image/color"
)

// FindConnectedComponents finds all 4-connected components of pixels with equal values.
func FindConnectedComponents(img *image.Gray16) (componentId []int32, componentValue []uint16, componentSize []int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// componentId maps each pixel to its component id. -1 means unvisited.
	componentId = make([]int32, width*height)
	for i := range componentId {
		componentId[i] = -1
	}

	// Initialize with base size of 512
	componentValue = make([]uint16, 0, 512)
	componentSize = make([]int, 0, 512)
	queue := make([]int, 0, 512)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {

			startIndex := y*width + x
			if componentId[startIndex] >= 0 {
				continue
			}

			value := UnsafeGray16At(img, bounds.Min.X+x, bounds.Min.Y+y)
			currentComponentId := int32(len(componentValue))
			componentId[startIndex] = currentComponentId

			queue = queue[:0]
			queue = append(queue, startIndex)
			size := 0

			for head := 0; head < len(queue); head++ {
				currentIndex := queue[head]
				size++

				currentX := currentIndex % width
				currentY := currentIndex / width

				if currentX > 0 {
					neighborIndex := currentIndex - 1
					if componentId[neighborIndex] < 0 && UnsafeGray16At(img, bounds.Min.X+currentX-1, bounds.Min.Y+currentY) == value {
						componentId[neighborIndex] = currentComponentId
						queue = append(queue, neighborIndex)
					}
				}
				if currentX+1 < width {
					neighborIndex := currentIndex + 1
					if componentId[neighborIndex] < 0 && UnsafeGray16At(img, bounds.Min.X+currentX+1, bounds.Min.Y+currentY) == value {
						componentId[neighborIndex] = currentComponentId
						queue = append(queue, neighborIndex)
					}
				}
				if currentY > 0 {
					neighborIndex := currentIndex - width
					if componentId[neighborIndex] < 0 && UnsafeGray16At(img, bounds.Min.X+currentX, bounds.Min.Y+currentY-1) == value {
						componentId[neighborIndex] = currentComponentId
						queue = append(queue, neighborIndex)
					}
				}
				if currentY+1 < height {
					neighborIndex := currentIndex + width
					if componentId[neighborIndex] < 0 && UnsafeGray16At(img, bounds.Min.X+currentX, bounds.Min.Y+currentY+1) == value {
						componentId[neighborIndex] = currentComponentId
						queue = append(queue, neighborIndex)
					}
				}
			}

			componentValue = append(componentValue, value)
			componentSize = append(componentSize, size)
		}
	}

	return componentId, componentValue, componentSize
}

// Identifies connected components of non-zero pixels in the mask and assigns each component a unique label in the output image.
// The value 0 is used as the background, 1 for the first island, 2 for the second, and so on.
// It also returns the total number of connected components (islands) found.
func FindIslands(mask *image.Gray16) (islandMap *image.Gray16, islandCount uint16) {

	islandMap = image.NewGray16(mask.Bounds())
	bounds := mask.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	componentId, componentValue, _ := FindConnectedComponents(mask)

	// Assign each foreground component a label
	islandLabels := make([]uint16, len(componentValue))
	nextLabel := uint16(1)
	for id, value := range componentValue {
		if value > 0 {
			islandLabels[id] = nextLabel
			nextLabel++
		}
	}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if islandLabel := islandLabels[componentId[y*width+x]]; islandLabel > 0 {
				islandMap.SetGray16(bounds.Min.X+x, bounds.Min.Y+y, color.Gray16{Y: islandLabel})
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
