package superpixel_map

import (
	"image"
)

type ThreeChannelUint8Map struct {

	// Dimensions
	Width  int
	Height int

	// Row-major order label map for superpixels. Each element represents the label of a pixel.
	Values [][3]uint8
}

func NewThreeChannelUint8Map(width, height int) *ThreeChannelUint8Map {
	return &ThreeChannelUint8Map{
		Width:  width,
		Height: height,
		Values: make([][3]uint8, width*height),
	}
}

func toGray(value [3]uint8) uint32 {
	return uint32(0.299*float64(value[0]) + 0.587*float64(value[1]) + 0.114*float64(value[2]))
}

// Returns the label at the specified coordinates without any bounds-checking.
// Use LabelAtSafely if you need bounds-checking.
func (m *ThreeChannelUint8Map) ValueAt(x, y int) [3]uint8 {
	return m.Values[y*m.Width+x]
}

func (m *ThreeChannelUint8Map) Uint32At(x, y int) uint32 {
	value := m.ValueAt(x, y)
	// Convert RGB to grayscale using the luminance formula
	return toGray(value)
}

func (m *ThreeChannelUint8Map) PositiveAt(x, y int) bool {
	v := m.ValueAt(x, y)
	return v[0] > 0 || v[1] > 0 || v[2] > 0
}

// Sets the label at the specified coordinates without any bounds-checking.
func (m *ThreeChannelUint8Map) SetValueAt(x, y int, value [3]uint8) {
	// Set the new value.
	m.Values[y*m.Width+x] = value
}

// Converts the label map to a 2D array of labels.
func (m *ThreeChannelUint8Map) To2DArray() [][][3]uint8 {
	result := make([][][3]uint8, m.Height)
	for y := 0; y < m.Height; y++ {
		row := make([][3]uint8, m.Width)
		for x := 0; x < m.Width; x++ {
			row[x] = m.Values[y*m.Width+x]
		}
		result[y] = row
	}
	return result
}

// Converts the label map to a 16 bit grayscale image.
// Values that exceed the 16-bit range will be clamped.
func (m *ThreeChannelUint8Map) ToImageGray16() *image.Gray16 {
	img := &image.Gray16{
		Pix:    make([]uint8, m.Width*m.Height*2), // Each pixel takes 2 bytes for Gray16
		Stride: m.Width * 2,
		Rect:   image.Rect(0, 0, m.Width, m.Height),
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			value := m.Values[y*m.Width+x]
			// Convert RGB to grayscale using the luminance formula
			gray := toGray(value)

			// clamping
			if gray > 0xFFFF {
				gray = 0xFFFF
			}
			offset := (y*m.Width + x) * 2
			img.Pix[offset] = uint8(gray >> 8)
			img.Pix[offset+1] = uint8(gray & 0xFF)
		}
	}
	return img
}

func (m *ThreeChannelUint8Map) Bounds() image.Rectangle {
	return image.Rect(0, 0, m.Width, m.Height)
}
