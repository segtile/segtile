package superpixel_map

import (
	"image"
)

type FloatMap struct {

	// Dimensions
	Width  int
	Height int

	// Row-major order label map for superpixels. Each element represents the label of a pixel.
	Values []float64
}

func NewFloatMap(width, height int) *FloatMap {
	return &FloatMap{
		Width:  width,
		Height: height,
		Values: make([]float64, width*height),
	}
}

// Returns the label at the specified coordinates without any bounds-checking.
// Use LabelAtSafely if you need bounds-checking.
func (m *FloatMap) ValueAt(x, y int) float64 {
	return m.Values[y*m.Width+x]
}

func (m *FloatMap) PositiveAt(x, y int) bool {
	return m.ValueAt(x, y) > 0
}

// Sets the label at the specified coordinates without any bounds-checking.
func (m *FloatMap) SetValueAt(x, y int, label float64) {
	// Set the new label.
	m.Values[y*m.Width+x] = label
}

// Converts the label map to a 2D array of labels.
func (m *FloatMap) To2DArray() [][]float64 {
	result := make([][]float64, m.Height)
	for y := 0; y < m.Height; y++ {
		row := make([]float64, m.Width)
		for x := 0; x < m.Width; x++ {
			row[x] = m.Values[y*m.Width+x]
		}
		result[y] = row
	}
	return result
}

// Converts the label map to a 16 bit grayscale image.
// Values that exceed the 16-bit range will be clamped.
func (m *FloatMap) ToImageGray16() *image.Gray16 {
	img := &image.Gray16{
		Pix:    make([]uint8, m.Width*m.Height*2),
		Stride: m.Width * 2,
		Rect:   image.Rect(0, 0, m.Width, m.Height),
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			value := m.Values[y*m.Width+x]

			// clamping
			if value > 0xFFFF {
				value = 0xFFFF
			}
			offset := (y*m.Width + x) * 2
			img.Pix[offset] = uint8(int(value) >> 8)
			img.Pix[offset+1] = uint8(int(value) & 0xFF)
		}
	}
	return img
}

func (m *FloatMap) Bounds() image.Rectangle {
	return image.Rect(0, 0, m.Width, m.Height)
}
