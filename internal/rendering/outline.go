package rendering

import (
	"image"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
)

func CalculateSelectionOutlineFromIndex(labelMap *image.Gray16, selectionIndex uint16) (pathSpec clip.PathSpec) {
	selection := make([]bool, 65536)
	selection[selectionIndex] = true
	return CalculateSelectionOutlineFromSlice(labelMap, selection)
}

func GetEmptySelectionOutline() (pathSpec clip.PathSpec) {
	selectionPathOps := new(op.Ops)
	var p clip.Path
	p.Begin(selectionPathOps)
	return p.End()
}

func CalculateCircleOutline(centerX int, centerY int, radius int) (pathSpec clip.PathSpec) {
	if radius <= 0 {
		return GetEmptySelectionOutline()
	}

	ops := new(op.Ops)
	bounds := image.Rect(centerX-radius, centerY-radius, centerX+radius, centerY+radius)
	return clip.Ellipse(bounds).Path(ops)
}

// buildOutlinePath turns border segments into a clip.PathSpec.
func buildOutlinePath(vertical, horizontal [][2]f32.Point) clip.PathSpec {
	ops := new(op.Ops)
	var p clip.Path
	p.Begin(ops)
	for _, seg := range vertical {
		p.MoveTo(seg[0])
		p.LineTo(seg[1])
	}
	for _, seg := range horizontal {
		p.MoveTo(seg[0])
		p.LineTo(seg[1])
	}
	return p.End()
}

func CalculateSelectionOutlineFromSlice(labelMap *image.Gray16, selection []bool) (pathSpec clip.PathSpec) {

	//log.Println("Updating selection outline ...")
	//start := time.Now().UnixMicro()

	width := labelMap.Bounds().Dx()
	height := labelMap.Bounds().Dy()
	pix := labelMap.Pix
	stride := labelMap.Stride

	// Read labels directly from the raw Pix buffer (big-endian uint16) to avoid per-pixel
	// bounds checking and interface dispatch overhead of Gray16At.

	var vertical, horizontal [][2]f32.Point
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// Check horizontally adjacent pairs: collect vertical border segments, merging consecutive runs.
		borders := make([][2]f32.Point, 0, width)
		for x := 0; x < width-1; x++ {
			runStart := -1
			base := x * 2
			for y := 0; y < height; y++ {
				i := y*stride + base
				la := uint16(pix[i])<<8 | uint16(pix[i+1])
				lb := uint16(pix[i+2])<<8 | uint16(pix[i+3])
				if selection[la] != selection[lb] {
					if runStart < 0 {
						runStart = y
					}
				} else if runStart >= 0 {
					borders = append(borders, [2]f32.Point{
						{X: float32(x + 1), Y: float32(runStart)},
						{X: float32(x + 1), Y: float32(y)},
					})
					runStart = -1
				}
			}
			if runStart >= 0 {
				borders = append(borders, [2]f32.Point{
					{X: float32(x + 1), Y: float32(runStart)},
					{X: float32(x + 1), Y: float32(height)},
				})
			}
		}
		vertical = borders
	}()

	go func() {
		defer wg.Done()
		// Check vertically adjacent pairs: collect horizontal border segments, merging consecutive runs.
		borders := make([][2]f32.Point, 0, height)
		for y := 0; y < height-1; y++ {
			runStart := -1
			rowBase := y * stride
			nextRowBase := rowBase + stride
			for x := 0; x < width; x++ {
				i := rowBase + x*2
				j := nextRowBase + x*2
				la := uint16(pix[i])<<8 | uint16(pix[i+1])
				lb := uint16(pix[j])<<8 | uint16(pix[j+1])
				if selection[la] != selection[lb] {
					if runStart < 0 {
						runStart = x
					}
				} else if runStart >= 0 {
					borders = append(borders, [2]f32.Point{
						{X: float32(runStart), Y: float32(y + 1)},
						{X: float32(x), Y: float32(y + 1)},
					})
					runStart = -1
				}
			}
			if runStart >= 0 {
				borders = append(borders, [2]f32.Point{
					{X: float32(runStart), Y: float32(y + 1)},
					{X: float32(width), Y: float32(y + 1)},
				})
			}
		}
		horizontal = borders
	}()

	wg.Wait()

	//end := time.Now().UnixMicro()
	//log.Printf("Selection outline updated in %d ms\n", (end-start)/1000)
	return buildOutlinePath(vertical, horizontal)

}

// CalculateOutlineFromMask computes a clip.PathSpec that traces the border between
// non-zero (masked) and zero (unmasked) pixels in mask. The resulting outline can be
// drawn directly at image-pixel coordinates without any superpixel approximation.
func CalculateOutlineFromMask(mask *image.Gray16) clip.PathSpec {
	width := mask.Bounds().Dx()
	height := mask.Bounds().Dy()
	pix := mask.Pix
	stride := mask.Stride

	var vertical, horizontal [][2]f32.Point
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// Vertical border segments between horizontally adjacent pixels.
		borders := make([][2]f32.Point, 0, width)
		for x := 0; x < width-1; x++ {
			runStart := -1
			base := x * 2
			for y := 0; y < height; y++ {
				i := y*stride + base
				la := pix[i] != 0 || pix[i+1] != 0
				lb := pix[i+2] != 0 || pix[i+3] != 0
				if la != lb {
					if runStart < 0 {
						runStart = y
					}
				} else if runStart >= 0 {
					borders = append(borders, [2]f32.Point{
						{X: float32(x + 1), Y: float32(runStart)},
						{X: float32(x + 1), Y: float32(y)},
					})
					runStart = -1
				}
			}
			if runStart >= 0 {
				borders = append(borders, [2]f32.Point{
					{X: float32(x + 1), Y: float32(runStart)},
					{X: float32(x + 1), Y: float32(height)},
				})
			}
		}
		vertical = borders
	}()

	go func() {
		defer wg.Done()
		// Horizontal border segments between vertically adjacent pixels.
		borders := make([][2]f32.Point, 0, height)
		for y := 0; y < height-1; y++ {
			runStart := -1
			rowBase := y * stride
			nextRowBase := rowBase + stride
			for x := 0; x < width; x++ {
				i := rowBase + x*2
				j := nextRowBase + x*2
				la := pix[i] != 0 || pix[i+1] != 0
				lb := pix[j] != 0 || pix[j+1] != 0
				if la != lb {
					if runStart < 0 {
						runStart = x
					}
				} else if runStart >= 0 {
					borders = append(borders, [2]f32.Point{
						{X: float32(runStart), Y: float32(y + 1)},
						{X: float32(x), Y: float32(y + 1)},
					})
					runStart = -1
				}
			}
			if runStart >= 0 {
				borders = append(borders, [2]f32.Point{
					{X: float32(runStart), Y: float32(y + 1)},
					{X: float32(width), Y: float32(y + 1)},
				})
			}
		}
		horizontal = borders
	}()

	wg.Wait()

	return buildOutlinePath(vertical, horizontal)
}
