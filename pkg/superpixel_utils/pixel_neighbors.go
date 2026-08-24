package superpixel_utils

import (
	"image"
)

// PixelNeighbours returns the neighboring pixels of a given pixel within the specified bounds.
// If includeDiagonals is true, it includes diagonal neighbors as well.
// If a pixel is on the edge of the image, it will only return the neighbors that are within the bounds.
func PixelNeighbours(point image.Point, bounds image.Rectangle, includeDiagonals bool) []image.Point {
	neighbors := []image.Point{}
	directions := []image.Point{
		{X: -1, Y: 0},
		{X: 1, Y: 0},
		{X: 0, Y: -1},
		{X: 0, Y: 1},
	}
	if includeDiagonals {
		directions = append(directions,
			image.Point{X: -1, Y: -1},
			image.Point{X: -1, Y: 1},
			image.Point{X: 1, Y: -1},
			image.Point{X: 1, Y: 1},
		)
	}
	for _, dir := range directions {
		neighbor := image.Point{X: point.X + dir.X, Y: point.Y + dir.Y}
		if neighbor.In(bounds) {
			neighbors = append(neighbors, neighbor)
		}
	}
	return neighbors
}

// FloodFill performs a flood fill starting from the given point in the provided image.
// It returns a slice of all points that are connected to the starting point and have the same pixel value.
func FloodFill(start image.Point, img *image.Gray16) (points []image.Point) {

	// Allocate a new empty visited matrix
	bounds := img.Bounds()
	visited := make([][]bool, bounds.Dy())
	for i := range visited {
		visited[i] = make([]bool, bounds.Dx())
	}

	return MaskedFloodFill(start, img, &visited)

}

// MaskedFloodFill is a variant of FloodFill which works with an existing "visited" matrix.
// This allows it to be used in scenarios where you want to keep track of visited pixels across multiple flood fill operations.
func MaskedFloodFill(start image.Point, img *image.Gray16, visited *[][]bool) (points []image.Point) {
	bounds := img.Bounds()
	if !start.In(bounds) {
		return nil
	}
	targetValue := img.Gray16At(start.X, start.Y).Y

	points = []image.Point{start}
	queue := []image.Point{start}

	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nb := range PixelNeighbours(cur, bounds, false) {

			if img.Gray16At(nb.X, nb.Y).Y != targetValue || (*visited)[nb.Y][nb.X] {
				continue
			}

			points = append(points, nb)
			queue = append(queue, nb)
			(*visited)[nb.Y][nb.X] = true
		}
	}

	return points
}
