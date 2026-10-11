package superpixel_map

import "image"

type LabelMap struct {

	// Dimensions
	Width  int
	Height int

	// Row-major order label map for superpixels. Each element represents the label of a pixel.
	Labels []uint32
}

func NewLabelMap(width, height int) *LabelMap {
	return &LabelMap{
		Width:  width,
		Height: height,
		Labels: make([]uint32, width*height),
	}
}

func NewLabelMapFromImage(img image.Image) *LabelMap {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()
	labelMap := NewLabelMap(width, height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			grayColor := img.At(x, y)
			r, g, b, _ := grayColor.RGBA()
			// Convert to 16-bit grayscale value
			gray16 := uint32((r + g + b) / 3 >> 8)
			labelMap.SetLabelAt(x, y, gray16)
		}
	}
	return labelMap
}

// Returns the label at the specified coordinates without any bounds-checking.
// Use LabelAtSafely if you need bounds-checking.
func (m *LabelMap) ValueAt(x, y int) uint32 {
	return m.Labels[y*m.Width+x]
}

func (m *LabelMap) PositiveAt(x, y int) bool {
	return m.ValueAt(x, y) > 0
}

func (m *LabelMap) LowestLabel() uint32 {
	lowestLabel := ^uint32(0) // max uint32

	for i := range m.Labels {

		if m.Labels[i] < lowestLabel {
			lowestLabel = m.Labels[i]
		}
	}

	return lowestLabel
}

func (m *LabelMap) HighestLabel() uint32 {
	highestLabel := uint32(0)

	for i := range m.Labels {
		if m.Labels[i] > highestLabel {
			highestLabel = m.Labels[i]
		}
	}

	return highestLabel
}

func (m *LabelMap) LabelUsed() map[uint32]bool {
	labelUsed := make(map[uint32]bool)
	for i := range m.Labels {
		labelUsed[m.Labels[i]] = true
	}
	return labelUsed
}

func (m *LabelMap) LabelPixelIndex() map[uint32]int {
	labelCount := make(map[uint32]int)
	for i := range m.Labels {
		labelCount[m.Labels[i]]++
	}
	return labelCount
}

// Sets the label at the specified coordinates without any bounds-checking.
func (m *LabelMap) SetLabelAt(x, y int, label uint32) {
	// Set the new label.
	m.Labels[y*m.Width+x] = label
}

// Sets the label at the specified coordinates safely. Returns false if the coordinates are out of bounds.
func (m *LabelMap) SetLabelAtSafely(x, y int, label uint32) bool {
	if x < 0 || x >= m.Width || y < 0 || y >= m.Height {
		return false
	}

	m.SetLabelAt(x, y, label)

	return true
}

// Converts the label map to a 2D array of labels.
func (m *LabelMap) To2DArray() [][]uint32 {
	result := make([][]uint32, m.Height)
	for y := 0; y < m.Height; y++ {
		row := make([]uint32, m.Width)
		for x := 0; x < m.Width; x++ {
			row[x] = m.Labels[y*m.Width+x]
		}
		result[y] = row
	}
	return result
}

// Converts the label map to a 16 bit grayscale image.
// Values that exceed the 16-bit range will be clamped.
func (m *LabelMap) ToImageGray16() image.Gray16 {
	img := image.Gray16{
		Pix:    make([]uint8, m.Width*m.Height*2),
		Stride: m.Width * 2,
		Rect:   image.Rect(0, 0, m.Width, m.Height),
	}
	for y := 0; y < m.Height; y++ {
		for x := 0; x < m.Width; x++ {
			label := m.Labels[y*m.Width+x]

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

func (m *LabelMap) Bounds() image.Rectangle {
	return image.Rect(0, 0, m.Width, m.Height)
}
