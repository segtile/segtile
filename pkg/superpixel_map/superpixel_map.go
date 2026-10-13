package superpixel_map

import "image"

type Map interface {
	Bounds() image.Rectangle
	ValueAt(x, y int) uint32
	PositiveAt(x, y int) bool
}
