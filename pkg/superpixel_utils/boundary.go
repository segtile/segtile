package superpixel_utils

import (
	"image"

	"github.com/segtile/segtile/pkg/superpixel_map"
)

// Finds all pixels in the image that have a neighbor with a different value and returns them as a slice of image.Points
// If singleSided is true, only the brighter pixel of each edge is returned, otherwise both pixels are returned.
func BoundaryPoints(labelMap superpixel_map.Map, singleSided bool) []image.Point {

	bounds := labelMap.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	points := []image.Point{}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			value := labelMap.Uint32At(x, y)
			hasBoundaryNeighbor := false

			for _, neighbor := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				neighborX := x + neighbor[0]
				neighborY := y + neighbor[1]

				// Skip out-of-bounds neighbors
				if neighborX < 0 || neighborX >= width || neighborY < 0 || neighborY >= height {
					continue
				}

				// Check if the neighbor has a different value
				neighborValue := labelMap.Uint32At(neighborX, neighborY)
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

func BoundaryPointsPerValue(labelMap superpixel_map.Map) (points map[uint32][]image.Point) {
	points = make(map[uint32][]image.Point)
	for y := 0; y < labelMap.Bounds().Dy(); y++ {
		for x := 0; x < labelMap.Bounds().Dx(); x++ {
			value := labelMap.Uint32At(x, y)
			hasBoundaryNeighbor := false

			for _, neighbor := range [][2]int{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				neighborX := x + neighbor[0]
				neighborY := y + neighbor[1]

				// Skip out-of-bounds neighbors
				if neighborX < 0 || neighborX >= labelMap.Bounds().Dx() || neighborY < 0 || neighborY >= labelMap.Bounds().Dy() {
					continue
				}

				// Check if the neighbor has a different value
				neighborValue := labelMap.Uint32At(neighborX, neighborY)
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
func DistanceToNextBoundaryPixel(labelMap superpixel_map.Map) *superpixel_map.LabelMap {
	bounds := labelMap.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	const inf = int(^uint(0) >> 1)
	dist := make([]int, w*h)
	for i := range dist {
		dist[i] = inf
	}

	queue := make([]image.Point, 0, w*h)

	// Seed the BFS with all boundary pixels (pixels with at least one neighbour with a different value).
	for _, p := range BoundaryPoints(labelMap, false) {
		idx := p.Y*w + p.X
		dist[idx] = 0
		queue = append(queue, p)
	}

	neighbors := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}

	// BFS propagation
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curIdx := cur.Y*w + cur.X
		curDist := dist[curIdx]
		for _, d := range neighbors {
			nx, ny := cur.X+d[0], cur.Y+d[1]
			if nx < 0 || nx >= w || ny < 0 || ny >= h {
				continue
			}
			nIdx := ny*w + nx
			if nd := curDist + 1; nd < dist[nIdx] {
				dist[nIdx] = nd
				queue = append(queue, image.Point{X: nx, Y: ny})
			}
		}
	}

	distanceMap := superpixel_map.NewLabelMap(w, h)
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			idx := y*w + x
			d := dist[idx]
			distanceMap.SetValueAt(x, y, uint32(d))
		}
	}

	return distanceMap
}
