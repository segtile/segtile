package superpixel_utils

import (
	"image"
)

// cloneMask returns a new Gray16 image with the same pixel values as mask.
func cloneMask(mask *image.Gray16) *image.Gray16 {
	bounds := mask.Bounds()
	clone := image.NewGray16(bounds)
	// Fast path: same stride means Pix is a contiguous block we can copy directly.
	if mask.Stride == clone.Stride && len(mask.Pix) == len(clone.Pix) {
		copy(clone.Pix, mask.Pix)
		return clone
	}
	// General path: copy row by row to respect differing strides / bounds origins.
	w := bounds.Dx() * 2
	for y := 0; y < bounds.Dy(); y++ {
		src := mask.Pix[y*mask.Stride : y*mask.Stride+w]
		dst := clone.Pix[y*clone.Stride : y*clone.Stride+w]
		copy(dst, src)
	}
	return clone
}

// distanceToBoundary computes, for every pixel in mask, the 4-connected BFS distance to the
// nearest pixel on the opposite side of the binary mask boundary (0 for boundary pixels
// themselves on either side). The image edge is not treated as a boundary.
func distanceToBoundary(mask *image.Gray16) []int {
	bounds := mask.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	const inf = int(^uint(0) >> 1)
	dist := make([]int, w*h)
	for i := range dist {
		dist[i] = inf
	}

	queue := make([]image.Point, 0, w*h)
	for _, p := range BoundaryPoints(mask, false) {
		idx := (p.Y-bounds.Min.Y)*w + (p.X - bounds.Min.X)
		dist[idx] = 0
		queue = append(queue, p)
	}

	neighbors := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
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
			nIdx := (ny-bounds.Min.Y)*w + (nx - bounds.Min.X)
			if nd := curDist + 1; nd < dist[nIdx] {
				dist[nIdx] = nd
				queue = append(queue, image.Point{X: nx, Y: ny})
			}
		}
	}
	return dist
}

// GrowShrink dilates (radius > 0) or erodes (radius < 0) a binary mask by the given number of
// pixels. radius == 0 returns an unchanged copy. mask is not modified.
func GrowShrink(mask *image.Gray16, radius int) *image.Gray16 {
	if radius == 0 {
		return cloneMask(mask)
	}

	bounds := mask.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	dist := distanceToBoundary(mask)
	result := image.NewGray16(bounds)

	srcStride := mask.Stride
	dstStride := result.Stride
	for y := 0; y < h; y++ {
		srcRow := y * srcStride
		dstRow := y * dstStride
		for x := 0; x < w; x++ {
			isForeground := mask.Pix[srcRow+x*2] != 0 || mask.Pix[srcRow+x*2+1] != 0
			d := dist[y*w+x]

			var keep bool
			if radius > 0 {
				// Dilate: keep existing foreground, plus background pixels within radius of the boundary.
				keep = isForeground || d < radius
			} else {
				// Erode: keep only foreground pixels at least -radius away from the boundary.
				keep = isForeground && d >= -radius
			}

			if keep {
				result.Pix[dstRow+x*2] = 0xFF
				result.Pix[dstRow+x*2+1] = 0xFF
			}
		}
	}
	return result
}

// SmoothMask blurs a binary mask with a box filter of the given pixel radius and re-binarizes
// at the 50% coverage threshold, so the result stays binary but has rounded/smoothed edges.
// radius <= 0 returns an unchanged copy. mask is not modified.
func SmoothMask(mask *image.Gray16, radius int) *image.Gray16 {
	if radius <= 0 {
		return cloneMask(mask)
	}

	bounds := mask.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()
	satStride := w + 1

	// Summed-area table over the binary mask (values 0/1) so any window sum is O(1).
	srcStride := mask.Stride
	sat := make([]int64, satStride*(h+1))
	for y := 0; y < h; y++ {
		srcRow := y * srcStride
		for x := 0; x < w; x++ {
			var v int64
			if mask.Pix[srcRow+x*2] != 0 || mask.Pix[srcRow+x*2+1] != 0 {
				v = 1
			}
			sat[(y+1)*satStride+(x+1)] = v + sat[y*satStride+(x+1)] + sat[(y+1)*satStride+x] - sat[y*satStride+x]
		}
	}

	result := image.NewGray16(bounds)
	dstStride := result.Stride
	for y := 0; y < h; y++ {
		y0, y1 := y-radius, y+radius
		if y0 < 0 {
			y0 = 0
		}
		if y1 >= h {
			y1 = h - 1
		}
		dstRow := y * dstStride
		for x := 0; x < w; x++ {
			x0, x1 := x-radius, x+radius
			if x0 < 0 {
				x0 = 0
			}
			if x1 >= w {
				x1 = w - 1
			}

			area := int64(x1-x0+1) * int64(y1-y0+1)
			sum := sat[(y1+1)*satStride+(x1+1)] - sat[y0*satStride+(x1+1)] - sat[(y1+1)*satStride+x0] + sat[y0*satStride+x0]

			if sum*2 >= area {
				result.Pix[dstRow+x*2] = 0xFF
				result.Pix[dstRow+x*2+1] = 0xFF
			}
		}
	}
	return result
}
