package superpixel_utils

import (
	"image"
	"image/color"
)

// Finds all pixels in the image that have a neighbor with a different value and returns them as a slice of image.Points
// If singleSided is true, only the brighter pixel of each edge is returned, otherwise both pixels are returned.
func BoundaryPoints(mask *image.Gray16, singleSided bool) []image.Point {
	bounds := mask.Bounds()
	points := []image.Point{}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			value := mask.Gray16At(x, y).Y
			hasBoundaryNeighbor := false

			for _, neighbor := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				neighborX := x + neighbor[0]
				neighborY := y + neighbor[1]

				// Skip out-of-bounds neighbors
				if neighborX < bounds.Min.X || neighborX >= bounds.Max.X || neighborY < bounds.Min.Y || neighborY >= bounds.Max.Y {
					continue
				}

				// Check if the neighbor has a different value
				neighborValue := mask.Gray16At(neighborX, neighborY).Y
				if singleSided {
					if neighborValue < value {
						hasBoundaryNeighbor = true
						break
					}
				} else if neighborValue != value {
					hasBoundaryNeighbor = true
					break
				}
			}
			if hasBoundaryNeighbor {
				points = append(points, image.Point{X: x, Y: y})
			}
		}
	}
	return points
}

func BoundaryPointsPerValue(labelMap *image.Gray16) (points map[uint16][]image.Point) {
	bounds := labelMap.Bounds()
	points = make(map[uint16][]image.Point)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			value := labelMap.Gray16At(x, y).Y
			hasBoundaryNeighbor := false

			for _, neighbor := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				neighborX := x + neighbor[0]
				neighborY := y + neighbor[1]

				// Skip out-of-bounds neighbors
				if neighborX < bounds.Min.X || neighborX >= bounds.Max.X || neighborY < bounds.Min.Y || neighborY >= bounds.Max.Y {
					continue
				}

				// Check if the neighbor has a different value
				neighborValue := labelMap.Gray16At(neighborX, neighborY).Y
				if neighborValue != value {
					hasBoundaryNeighbor = true
					break
				}
			}
			if hasBoundaryNeighbor {
				points[value] = append(points[value], image.Point{X: x, Y: y})
			}
		}
	}
	return points
}

// DistanceToNextBoundaryPixel computes the distance from each pixel to another pixel with a different value in the segmentation.
// If a given pixel is a boundary pixel, its distance will be 0.
func DistanceToNextBoundaryPixel(segmentation *image.Gray16) *image.Gray16 {
	bounds := segmentation.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	const inf = int(^uint(0) >> 1)
	dist := make([]int, w*h)
	for i := range dist {
		dist[i] = inf
	}

	queue := make([]image.Point, 0, w*h)

	// Seed the BFS with all boundary pixels (pixels with at least one neighbour with a different value).
	for _, p := range BoundaryPoints(segmentation, false) {
		idx := (p.Y-bounds.Min.Y)*w + (p.X - bounds.Min.X)
		dist[idx] = 0
		queue = append(queue, p)
	}

	neighbors := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	// BFS propagation — each step increments distance by 1 (Chebyshev-like; use squared for Euclidean approximation).
	// For exact squared-Euclidean use a two-pass approach; BFS gives L∞ (Chebyshev) distance which is a fast, tight approximation.
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curIdx := (cur.Y-bounds.Min.Y)*w + (cur.X - bounds.Min.X)
		curDist := dist[curIdx]
		for _, d := range neighbors {
			nx, ny := cur.X+d[0], cur.Y+d[1]
			if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
				continue
			}
			if segmentation.Gray16At(nx, ny).Y == 0 {
				continue
			}
			nIdx := (ny-bounds.Min.Y)*w + (nx - bounds.Min.X)
			if nd := curDist + 1; nd < dist[nIdx] {
				dist[nIdx] = nd
				queue = append(queue, image.Point{X: nx, Y: ny})
			}
		}
	}

	// Write results to output image.
	distanceMap := image.NewGray16(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if segmentation.Gray16At(x, y).Y > 0 {
				idx := (y-bounds.Min.Y)*w + (x - bounds.Min.X)
				d := dist[idx]
				if d > 65535 {
					d = 65535
				}
				distanceMap.SetGray16(x, y, color.Gray16{Y: uint16(d)})
			}
		}
	}

	return distanceMap
}
