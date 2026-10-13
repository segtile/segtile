package superpixel_map

import (
	"image"
)

type IntMap struct {

	// Dimensions
	Width  int
	Height int

	// Row-major order label map for superpixels. Each element represents the label of a pixel.
	Values []int
}

func NewIntMap(width, height int) *IntMap {
	return &IntMap{
		Width:  width,
		Height: height,
		Values: make([]int, width*height),
	}
}

// Returns the label at the specified coordinates without any bounds-checking.
// Use LabelAtSafely if you need bounds-checking.
func (m *IntMap) ValueAt(x, y int) int {
	return m.Values[y*m.Width+x]
}

func (m *IntMap) Uint32At(x, y int) uint32 {
	return uint32(m.ValueAt(x, y))
}

func (m *IntMap) PositiveAt(x, y int) bool {
	return m.ValueAt(x, y) > 0
}

// Sets the label at the specified coordinates without any bounds-checking.
func (m *IntMap) SetValueAt(x, y int, label int) {
	// Set the new label.
	m.Values[y*m.Width+x] = label
}

// Converts the label map to a 2D array of labels.
func (m *IntMap) To2DArray() [][]int {
	result := make([][]int, m.Height)
	for y := 0; y < m.Height; y++ {
		row := make([]int, m.Width)
		for x := 0; x < m.Width; x++ {
			row[x] = m.Values[y*m.Width+x]
		}
		result[y] = row
	}
	return result
}

// Converts the label map to a 16 bit grayscale image.
// Values that exceed the 16-bit range will be clamped.
func (m *IntMap) ToImageGray16() *image.Gray16 {
	img := &image.Gray16{
		Pix:    make([]uint8, m.Width*m.Height*2),
		Stride: m.Width * 2,
		Rect:   image.Rect(0, 0, m.Width, m.Height),
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			label := m.Values[y*m.Width+x]

			// clamping
			if label > 0xFFFF {
				label = 0xFFFF
			}
			offset := (y*m.Width + x) * 2
			img.Pix[offset] = uint8(label >> 8)
			img.Pix[offset+1] = uint8(label & 0xFF)
		}
	}
	return img
}

func (m *IntMap) ToLabelMap() *LabelMap {
	labelMap := NewLabelMap(m.Width, m.Height)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			labelMap.SetValueAt(x, y, uint32(m.ValueAt(x, y)))
		}
	}
	return labelMap
}

func (m *IntMap) Bounds() image.Rectangle {
	return image.Rect(0, 0, m.Width, m.Height)
}
