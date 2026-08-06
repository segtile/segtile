package superpixel_utils

import "image"

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
