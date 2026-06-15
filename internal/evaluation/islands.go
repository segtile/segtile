package evaluation

import (
	"image"
	"image/color"
)

// Identifies connected components of non-zero pixels in the mask and assigns each component a unique label in the output image.
// The value 0 is used as the background, 1 for the first island, 2 for the second, and so on.
// It also returns the total number of connected components (islands) found.
func FindIslands(mask *image.Gray16) (islandMap *image.Gray16, islandCount uint16) {

	islandMap = image.NewGray16(mask.Bounds())
	bounds := mask.Bounds()
	visited := make([][]bool, bounds.Dy())

	for y := range visited {
		visited[y] = make([]bool, bounds.Dx())
	}

	islandLabel := uint16(1)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.Gray16At(x, y).Y > 0 && !visited[y][x] {
				// Start a new island
				queue := []struct{ x, y int }{{x, y}}
				visited[y][x] = true
				islandMap.SetGray16(x, y, color.Gray16{Y: islandLabel})
				for len(queue) > 0 {
					current := queue[0]
					queue = queue[1:]
					// Check 4-neighborhood
					for _, offset := range []struct{ dx, dy int }{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
						newX, newY := current.x+offset.dx, current.y+offset.dy
						if newX >= bounds.Min.X && newX < bounds.Max.X && newY >= bounds.Min.Y && newY < bounds.Max.Y {
							if mask.Gray16At(newX, newY).Y > 0 && !visited[newY][newX] {
								visited[newY][newX] = true
								islandMap.SetGray16(newX, newY, color.Gray16{Y: islandLabel})
								queue = append(queue, struct{ x, y int }{newX, newY})
							}
						}
					}
				}
				islandLabel++
			}
		}
	}
	return islandMap, islandLabel - 1
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
