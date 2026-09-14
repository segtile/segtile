package superpixel_utils

import (
	"image"
	"image/color"
)

// FindConnectedComponents finds all 4-connected components of pixels with equal values.
// The returned map stores component id + 1 per pixel (0 means unvisited while the search is
// running); after the call every pixel holds a valid id in 1..componentCount.
func FindConnectedComponents(img *image.Gray16) (componentIdMap *image.Gray16, componentValue []uint16, componentSize []int) {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	// Zero-initialized: 0 means unvisited, stored ids are shifted by +1.
	componentIdMap = image.NewGray16(bounds)

	componentValue = []uint16{}
	componentSize = []int{}
	queue := []int{}

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {

			// Skip pixels that already belong to a component
			if UnsafeGray16At(componentIdMap, x, y) != 0 {
				continue
			}

			// Start a new connected component from this pixel
			value := UnsafeGray16At(img, x, y)
			currentComponentId := uint16(len(componentValue)) + 1
			UnsafeSetGray16At(componentIdMap, x, y, currentComponentId)

			startIndex := y*width + x
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
					if UnsafeGray16At(componentIdMap, currentX-1, currentY) == 0 && UnsafeGray16At(img, currentX-1, currentY) == value {
						UnsafeSetGray16At(componentIdMap, currentX-1, currentY, currentComponentId)
						queue = append(queue, neighborIndex)
					}
				}
				// Right neighbor
				if currentX+1 < width {
					neighborIndex := currentIndex + 1
					if UnsafeGray16At(componentIdMap, currentX+1, currentY) == 0 && UnsafeGray16At(img, currentX+1, currentY) == value {
						UnsafeSetGray16At(componentIdMap, currentX+1, currentY, currentComponentId)
						queue = append(queue, neighborIndex)
					}
				}
				// Top neighbor
				if currentY > 0 {
					neighborIndex := currentIndex - width
					if UnsafeGray16At(componentIdMap, currentX, currentY-1) == 0 && UnsafeGray16At(img, currentX, currentY-1) == value {
						UnsafeSetGray16At(componentIdMap, currentX, currentY-1, currentComponentId)
						queue = append(queue, neighborIndex)
					}
				}
				// Bottom neighbor
				if currentY+1 < height {
					neighborIndex := currentIndex + width
					if UnsafeGray16At(componentIdMap, currentX, currentY+1) == 0 && UnsafeGray16At(img, currentX, currentY+1) == value {
						UnsafeSetGray16At(componentIdMap, currentX, currentY+1, currentComponentId)
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
			componentId := UnsafeGray16At(componentIdMap, x, y) - 1
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
