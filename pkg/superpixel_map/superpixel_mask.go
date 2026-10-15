package superpixel_map

import (
	"image"
	"image/color"
)

type Mask struct {
	Width  int
	Height int
	Mask   []bool
}

func NewMask(width, height int) *Mask {
	return &Mask{
		Width:  width,
		Height: height,
		Mask:   make([]bool, width*height),
	}
}

func NewMaskFromImage(img image.Image) *Mask {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	mask := NewMask(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grayColor := color.Gray16Model.Convert(img.At(bounds.Min.X+x, bounds.Min.Y+y)).(color.Gray16)
			mask.SetValueAt(x, y, grayColor.Y > 0)
		}
	}
	return mask
}

// Returns the value at the specified coordinates without any bounds-checking.
// Use ValueAtSafely if you need bounds-checking.
func (m *Mask) PositiveAt(x, y int) bool {
	return m.Mask[y*m.Width+x]
}

// For interface compatibility
func (m *Mask) Uint32At(x, y int) uint32 {
	if m.Mask[y*m.Width+x] {
		return 1
	}
	return 0
}

func (m *Mask) SetValueAt(x, y int, value bool) {
	m.Mask[y*m.Width+x] = value
}

func (m *Mask) SetValueAtSafely(x, y int, value bool) bool {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return false
	}
	m.Mask[y*m.Width+x] = value
	return true
}

// Equals reports whether both masks have identical dimensions and pixel values.
func (m *Mask) Equals(other *Mask) bool {
	if m == nil || other == nil {
		return m == other
	}
	if m.Width != other.Width || m.Height != other.Height {
		return false
	}
	for i := range m.Mask {
		if m.Mask[i] != other.Mask[i] {
			return false
		}
	}
	return true
}

func (m *Mask) IsEmpty() bool {
	for i := range m.Mask {
		if m.Mask[i] {
			return false
		}
	}
	return true
}

func (m *Mask) Clear() {
	for i := range m.Mask {
		m.Mask[i] = false
	}
}

// Converts the mask to a 2D array of boolean values.
func (m *Mask) To2DArray() [][]bool {
	result := make([][]bool, m.Height)
	for y := 0; y < m.Height; y++ {
		row := make([]bool, m.Width)
		for x := 0; x < m.Width; x++ {
			row[x] = m.Mask[y*m.Width+x]
		}
		result[y] = row
	}
	return result
}

// Converts the mask to a 16 bit grayscale image.
// Values that exceed the 16-bit range will be clamped.
func (m *Mask) ToImageGray16() *image.Gray16 {
	img := &image.Gray16{
		Pix:    make([]uint8, m.Width*m.Height*2),
		Stride: m.Width * 2,
		Rect:   image.Rect(0, 0, m.Width, m.Height),
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			mask := m.Mask[y*m.Width+x]

			var value uint16
			if mask {
				value = 0xFFFF
			} else {
				value = 0
			}
			offset := (y*m.Width + x) * 2
			img.Pix[offset] = uint8(value >> 8)
			img.Pix[offset+1] = uint8(value & 0xFF)
		}
	}
	return img
}

func (m *Mask) ToLabelMap() *LabelMap {
	labelMap := NewLabelMap(m.Width, m.Height)
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			if m.PositiveAt(x, y) {
				labelMap.SetValueAt(x, y, 1)
			} else {
				labelMap.SetValueAt(x, y, 0)
			}
		}
	}
	return labelMap
}

func (m *Mask) Bounds() image.Rectangle {
	return image.Rect(0, 0, m.Width, m.Height)
}

func (m *Mask) Clone() *Mask {
	clone := NewMask(m.Width, m.Height)
	copy(clone.Mask, m.Mask)
	return clone
}
