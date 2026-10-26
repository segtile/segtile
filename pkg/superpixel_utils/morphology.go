package superpixel_utils

import (
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

// distanceToBoundary computes, for every pixel in mask, the 4-connected BFS distance to the
// nearest pixel on the opposite side of the binary mask boundary (0 for boundary pixels
// themselves on either side). The image edge is not treated as a boundary.
func distanceToBoundary(mask *superpixel_map.Mask) *superpixel_map.LabelMap {

	const inf = ^uint32(0) >> 1
	dist := make([]uint32, mask.Width*mask.Height)
	for i := range dist {
		dist[i] = inf
	}

	queue := make([]image.Point, 0, mask.Width*mask.Height)
	for _, p := range BoundaryPoints(mask, false) {
		idx := p.Y*mask.Width + p.X
		dist[idx] = 0
		queue = append(queue, p)
	}

	neighbors := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curIdx := cur.Y*mask.Width + cur.X
		curDist := dist[curIdx]
		for _, d := range neighbors {
			nx, ny := cur.X+d[0], cur.Y+d[1]
			if nx < 0 || nx >= mask.Width || ny < 0 || ny >= mask.Height {
				continue
			}
			nIdx := ny*mask.Width + nx
			if nd := curDist + 1; nd < dist[nIdx] {
				dist[nIdx] = nd
				queue = append(queue, image.Point{X: nx, Y: ny})
			}
		}
	}
	return &superpixel_map.LabelMap{
		Width:  mask.Width,
		Height: mask.Height,
		Labels: dist,
	}
}

// GrowShrink dilates (radius > 0) or erodes (radius < 0) a binary mask by the given number of pixels.
// The operation is performed in-place on the input mask.
// Radius == 0 leaves the mask unchanged.
func GrowShrink(mask *superpixel_map.Mask, radius int) {

	if radius == 0 {
		return
	}

	dist := distanceToBoundary(mask)
	w, h := mask.Width, mask.Height
	for y := 0; y < h; y++ {
		for x := 0; x < w; x++ {
			isForeground := mask.PositiveAt(x, y)
			d := uint64(dist.Labels[y*w+x])

			var keep bool
			if radius > 0 {
				// Dilate: keep existing foreground, plus background pixels within radius of the boundary.
				keep = isForeground || d < uint64(radius)
			} else {
				// Erode: keep only foreground pixels at least -radius away from the boundary.
				erosionRadius := uint64(-(radius + 1)) + 1
				keep = isForeground && d >= erosionRadius
			}

			mask.SetValueAt(x, y, keep)
		}
	}
}

// SmoothMask blurs a binary mask with a box filter of the given pixel radius and re-binarizes
// at the 50% coverage threshold, so the result stays binary but has rounded/smoothed edges.
// The operation is performed in-place. Radius <= 0 leaves the mask unchanged.
func SmoothMask(mask *superpixel_map.Mask, radius int) {
	if radius <= 0 {
		return
	}

	// Summed-area table over the binary mask (values 0/1) so any window sum is O(1).
	satStride := mask.Width + 1
	sat := make([]int64, satStride*(mask.Height+1))
	for y := 0; y < mask.Height; y++ {
		for x := 0; x < mask.Width; x++ {
			var v int64
			if mask.PositiveAt(x, y) {
				v = 1
			}
			sat[(y+1)*satStride+(x+1)] = v + sat[y*satStride+(x+1)] + sat[(y+1)*satStride+x] - sat[y*satStride+x]
		}
	}

	for y := 0; y < mask.Height; y++ {
		y0 := 0
		if radius < y {
			y0 = y - radius
		}
		y1 := mask.Height - 1
		if radius < mask.Height-1-y {
			y1 = y + radius
		}
		for x := 0; x < mask.Width; x++ {
			x0 := 0
			if radius < x {
				x0 = x - radius
			}
			x1 := mask.Width - 1
			if radius < mask.Width-1-x {
				x1 = x + radius
			}

			area := int64(x1-x0+1) * int64(y1-y0+1)
			sum := sat[(y1+1)*satStride+(x1+1)] - sat[y0*satStride+(x1+1)] - sat[(y1+1)*satStride+x0] + sat[y0*satStride+x0]

			mask.SetValueAt(x, y, sum*2 >= area)
		}
	}
}
