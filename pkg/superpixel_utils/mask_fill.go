package superpixel_utils

import "image"

func FloodFillMaskFromPoint(mask *image.Gray16, x, y int) bool {
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
				setMaskWhiteAt(mask, px, py)
				changed = true
			}
		}
	}
	return changed
}

func FillEnclosedMaskRegions(mask *image.Gray16) bool {
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
			if !rowReachable[px-bounds.Min.X] && isMaskZeroAt(mask, px, py) {
				setMaskWhiteAt(mask, px, py)
				changed = true
			}
		}
	}
	return changed
}

func floodFillUnselected(mask *image.Gray16, seeds []image.Point) []bool {
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
		if visited[index] || !isMaskZeroAt(mask, x, y) {
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

// isMaskZeroAt reports whether the pixel at (x, y) is 0, without bounds checking.
func isMaskZeroAt(img *image.Gray16, x, y int) bool {
	offset := img.PixOffset(x, y)
	return img.Pix[offset] == 0 && img.Pix[offset+1] == 0
}

// setMaskWhiteAt sets the pixel at (x, y) to 65535, without bounds checking.
func setMaskWhiteAt(img *image.Gray16, x, y int) {
	offset := img.PixOffset(x, y)
	img.Pix[offset] = 0xFF
	img.Pix[offset+1] = 0xFF
}
