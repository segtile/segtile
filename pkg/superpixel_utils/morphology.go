package superpixel_utils

import (
	"image"
	"image/color"
)

// cloneMask returns a new Gray16 image with the same pixel values as mask.
func cloneMask(mask *image.Gray16) *image.Gray16 {
	bounds := mask.Bounds()
	clone := image.NewGray16(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			clone.SetGray16(x, y, mask.Gray16At(x, y))
		}
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
	dist := distanceToBoundary(mask)
	result := image.NewGray16(bounds)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			idx := (y-bounds.Min.Y)*w + (x - bounds.Min.X)
			isForeground := mask.Gray16At(x, y).Y != 0
			d := dist[idx]

			var keep bool
			if radius > 0 {
				// Dilate: keep existing foreground, plus background pixels within radius of the boundary.
				keep = isForeground || d < radius
			} else {
				// Erode: keep only foreground pixels at least -radius away from the boundary.
				keep = isForeground && d >= -radius
			}

			if keep {
				result.SetGray16(x, y, color.Gray16{Y: 65535})
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
	stride := w + 1

	// Summed-area table over the binary mask (values 0/1) so any window sum is O(1).
	sat := make([]int64, stride*(h+1))
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			var v int64
			if mask.Gray16At(bounds.Min.X+x, bounds.Min.Y+y).Y != 0 {
				v = 1
			}
			sat[(y+1)*stride+(x+1)] = v + sat[y*stride+(x+1)] + sat[(y+1)*stride+x] - sat[y*stride+x]
		}
	}

	result := image.NewGray16(bounds)
	for y := 0; y < h; y++ {
		y0, y1 := y-radius, y+radius
		if y0 < 0 {
			y0 = 0
		}
		if y1 >= h {
			y1 = h - 1
		}
		for x := 0; x < w; x++ {
			x0, x1 := x-radius, x+radius
			if x0 < 0 {
				x0 = 0
			}
			if x1 >= w {
				x1 = w - 1
			}

			area := int64(x1-x0+1) * int64(y1-y0+1)
			sum := sat[(y1+1)*stride+(x1+1)] - sat[y0*stride+(x1+1)] - sat[(y1+1)*stride+x0] + sat[y0*stride+x0]

			if sum*2 >= area {
				result.SetGray16(bounds.Min.X+x, bounds.Min.Y+y, color.Gray16{Y: 65535})
			}
		}
	}
	return result
}
