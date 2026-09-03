package superpixel_utils

import "math"

type BoundingBox struct {
	minX, minY, maxX, maxY int
	initialized            bool
}

func (bb *BoundingBox) AddPoint(x, y int) {
	if !bb.initialized {
		bb.minX, bb.minY, bb.maxX, bb.maxY = x, y, x, y
		bb.initialized = true
		return
	}
	if x < bb.minX {
		bb.minX = x
	}
	if x > bb.maxX {
		bb.maxX = x
	}
	if y < bb.minY {
		bb.minY = y
	}
	if y > bb.maxY {
		bb.maxY = y
	}
}

func (bb BoundingBox) Area() int {
	if !bb.initialized {
		return 0
	}
	return (bb.maxX - bb.minX + 1) * (bb.maxY - bb.minY + 1)
}

func (bb BoundingBox) Width() int {
	if !bb.initialized {
		return 0
	}
	return bb.maxX - bb.minX + 1
}

func (bb BoundingBox) Height() int {
	if !bb.initialized {
		return 0
	}
	return bb.maxY - bb.minY + 1
}

func (bb BoundingBox) Diagonal() float64 {
	if !bb.initialized {
		return 0
	}
	width := bb.Width()
	height := bb.Height()
	return math.Sqrt(float64(width*width + height*height))
}
