package rendering

import (
	"image"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func CalculateSelectionOutlineFromIndex(labelMap *image.Gray16, selectionIndex uint16) (pathSpec clip.PathSpec) {
	selection := make([]bool, 65536)
	selection[selectionIndex] = true

	// TODO: This is the only reason the CalculateSelectionOutlineFromSlice function really exists, there is probably a better architecture for this.
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
	// pix := labelMap.Pix
	// stride := labelMap.Stride

	var vertical, horizontal [][2]f32.Point
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()

		// Check horizontally adjacent pairs: collect vertical border segments, merging consecutive runs
		borders := make([][2]f32.Point, 0, width)
		runStarts := make([]int, width-1)
		for x := range runStarts {
			runStarts[x] = -1
		}
		for y := 0; y < height; y++ {

			for x := 0; x < width-1; x++ {

				labelA := superpixel_utils.UnsafeGray16At(labelMap, x, y)
				labelB := superpixel_utils.UnsafeGray16At(labelMap, x+1, y)

				if selection[labelA] != selection[labelB] {
					if runStarts[x] < 0 {
						runStarts[x] = y
					}
				} else if runStarts[x] >= 0 {
					borders = append(borders, [2]f32.Point{
						{X: float32(x + 1), Y: float32(runStarts[x])},
						{X: float32(x + 1), Y: float32(y)},
					})
					runStarts[x] = -1
				}
			}
		}
		for x := 0; x < width-1; x++ {
			if runStarts[x] >= 0 {
				borders = append(borders, [2]f32.Point{
					{X: float32(x + 1), Y: float32(runStarts[x])},
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
			for x := 0; x < width; x++ {
				labelA := superpixel_utils.UnsafeGray16At(labelMap, x, y)
				labelB := superpixel_utils.UnsafeGray16At(labelMap, x, y+1)
				if selection[labelA] != selection[labelB] {
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

	var vertical, horizontal [][2]f32.Point
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// Vertical border segments between horizontally adjacent pixels.
		// Row-major scan (cache-friendly); run starts are tracked per column gap.
		borders := make([][2]f32.Point, 0, width)
		runStarts := make([]int, width-1)
		for x := range runStarts {
			runStarts[x] = -1
		}
		for y := 0; y < height; y++ {
			for x := 0; x < width-1; x++ {
				labelA := superpixel_utils.UnsafeGray16At(mask, x, y)
				labelB := superpixel_utils.UnsafeGray16At(mask, x+1, y)
				if labelA != labelB {
					if runStarts[x] < 0 {
						runStarts[x] = y
					}
				} else if runStarts[x] >= 0 {
					borders = append(borders, [2]f32.Point{
						{X: float32(x + 1), Y: float32(runStarts[x])},
						{X: float32(x + 1), Y: float32(y)},
					})
					runStarts[x] = -1
				}
			}
		}
		for x := 0; x < width-1; x++ {
			if runStarts[x] >= 0 {
				borders = append(borders, [2]f32.Point{
					{X: float32(x + 1), Y: float32(runStarts[x])},
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
			for x := 0; x < width; x++ {
				labelA := superpixel_utils.UnsafeGray16At(mask, x, y)
				labelB := superpixel_utils.UnsafeGray16At(mask, x, y+1)
				if labelA != labelB {
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
