package superpixel_utils

import (
	"image"

	"github.com/segtile/segtile/pkg/superpixel_map"
)

func FloodFillMaskFromPoint(mask *superpixel_map.Mask, x, y int) bool {
	bounds := mask.Bounds()
	if !image.Pt(x, y).In(bounds) {
		return false
	}
	visited := floodFillUnselected(mask, []image.Point{{X: x, Y: y}})
	changed := false
	for py := bounds.Min.Y; py < bounds.Max.Y; py++ {
		rowVisited := visited[(py-bounds.Min.Y)*bounds.Dx() : (py-bounds.Min.Y+1)*bounds.Dx()]
		for px := bounds.Min.X; px < bounds.Max.X; px++ {
			if rowVisited[px-bounds.Min.X] {
				mask.SetValueAt(px, py, true)
				changed = true
			}
		}
	}
	return changed
}

func FillEnclosedMaskRegions(mask *superpixel_map.Mask) bool {
	bounds := mask.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	if width == 0 || height == 0 {
		return false
	}

	// Seed the BFS with all unselected pixels on the image border.
	seeds := make([]image.Point, 0, 2*(width+height))
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		seeds = append(seeds, image.Pt(x, bounds.Min.Y), image.Pt(x, bounds.Max.Y-1))
	}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		seeds = append(seeds, image.Pt(bounds.Min.X, y), image.Pt(bounds.Max.X-1, y))
	}

	// Mark all unselected pixels reachable from the border.
	reachable := floodFillUnselected(mask, seeds)

	// Select everything that is neither selected nor reachable from the border.
	changed := false
	for py := bounds.Min.Y; py < bounds.Max.Y; py++ {
		rowReachable := reachable[(py-bounds.Min.Y)*width : (py-bounds.Min.Y+1)*width]
		for px := bounds.Min.X; px < bounds.Max.X; px++ {
			if !rowReachable[px-bounds.Min.X] && !mask.PositiveAt(px, py) {
				mask.SetValueAt(px, py, true)
				changed = true
			}
		}
	}
	return changed
}

func floodFillUnselected(mask *superpixel_map.Mask, seeds []image.Point) []bool {
	bounds := mask.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	size := width * height

	visited := make([]bool, size)

	// Queue with head-pointer dequeue; each pixel is enqueued at most once.
	queue := make([]image.Point, 0, 4096)
	head := 0

	enqueue := func(x, y int) {
		index := (y-bounds.Min.Y)*width + (x - bounds.Min.X)
		if visited[index] || mask.PositiveAt(x, y) {
			return
		}
		visited[index] = true
		queue = append(queue, image.Point{X: x, Y: y})
	}

	for _, seed := range seeds {
		if seed.In(bounds) {
			enqueue(seed.X, seed.Y)
		}
	}

	for head < len(queue) {
		current := queue[head]
		head++
		// Compact the consumed prefix once it dominates the queue, to bound memory.
		if head > 4096 && head*2 >= len(queue) {
			copy(queue, queue[head:])
			queue = queue[:len(queue)-head]
			head = 0
		}
		if current.X > bounds.Min.X {
			enqueue(current.X-1, current.Y)
		}
		if current.X < bounds.Max.X-1 {
			enqueue(current.X+1, current.Y)
		}
		if current.Y > bounds.Min.Y {
			enqueue(current.X, current.Y-1)
		}
		if current.Y < bounds.Max.Y-1 {
			enqueue(current.X, current.Y+1)
		}
	}
	return visited
}
