package rendering

import (
	"image"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// Standard outlines

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

// Custom outlines for masks

type outlineSegment struct {
	startPoint f32.Point
	endPoint   f32.Point
}

// Generates the outline marking the line between pixels in the image that has the target value and those that do not.
func CalculateOutline(mask *image.Gray16, targetValue uint16) clip.PathSpec {

	// GATHER BORDERS

	width := mask.Bounds().Dx()
	height := mask.Bounds().Dy()

	if width <= 0 || height <= 0 {
		return GetEmptySelectionOutline()
	}

	var vertical, horizontal []outlineSegment
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		defer wg.Done()
		// Vertical border segments between horizontally adjacent pixels.
		// Row-major scan (cache-friendly); run starts are tracked per column gap.
		borders := make([]outlineSegment, 0, width)
		runStarts := make([]int, width-1)
		for x := range runStarts {
			runStarts[x] = -1
		}
		for y := 0; y < height; y++ {
			for x := 0; x < width-1; x++ {
				labelA := (superpixel_utils.UnsafeGray16At(mask, x, y) == targetValue)
				labelB := (superpixel_utils.UnsafeGray16At(mask, x+1, y) == targetValue)
				if labelA != labelB {
					if runStarts[x] < 0 {
						runStarts[x] = y
					}
				} else if runStarts[x] >= 0 {
					borders = append(borders, outlineSegment{
						startPoint: f32.Point{X: float32(x + 1), Y: float32(runStarts[x])},
						endPoint:   f32.Point{X: float32(x + 1), Y: float32(y)},
					})
					runStarts[x] = -1
				}
			}
		}
		for x := 0; x < width-1; x++ {
			if runStarts[x] >= 0 {
				borders = append(borders, outlineSegment{
					startPoint: f32.Point{X: float32(x + 1), Y: float32(runStarts[x])},
					endPoint:   f32.Point{X: float32(x + 1), Y: float32(height)},
				})
			}
		}
		vertical = borders
	}()

	go func() {
		defer wg.Done()
		// Horizontal border segments between vertically adjacent pixels.
		borders := make([]outlineSegment, 0, height)
		for y := 0; y < height-1; y++ {
			runStart := -1
			for x := 0; x < width; x++ {
				labelA := (superpixel_utils.UnsafeGray16At(mask, x, y) == targetValue)
				labelB := (superpixel_utils.UnsafeGray16At(mask, x, y+1) == targetValue)
				if labelA != labelB {
					if runStart < 0 {
						runStart = x
					}
				} else if runStart >= 0 {
					borders = append(borders, outlineSegment{
						startPoint: f32.Point{X: float32(runStart), Y: float32(y + 1)},
						endPoint:   f32.Point{X: float32(x), Y: float32(y + 1)},
					})
					runStart = -1
				}
			}
			if runStart >= 0 {
				borders = append(borders, outlineSegment{
					startPoint: f32.Point{X: float32(runStart), Y: float32(y + 1)},
					endPoint:   f32.Point{X: float32(width), Y: float32(y + 1)},
				})
			}
		}
		horizontal = borders
	}()

	wg.Wait()

	segments := append(vertical, horizontal...)

	// OPTIMIZE BORDERS

	// vertexIndex maps a grid point to its index in the (width+1)x(height+1)
	// vertex grid. Segment endpoints always lie on integer grid coordinates.
	vertexIndex := func(pt f32.Point) int32 {
		return int32(pt.Y)*int32(width+1) + int32(pt.X)
	}

	// startIndex[i] and b[i] are the vertex indices of segment i's endpoints.
	startIndex := make([]int32, len(segments))
	endIndex := make([]int32, len(segments))

	// adjHead[v] is the most recently registered segment incident to v,
	// stored as index+1 (0 = none). adjNext holds two link slots per segment:
	// slot 2i for endpoint a, slot 2i+1 for endpoint b (0 = list end).
	adjHead := make([]int32, (width+1)*(height+1))
	adjNext := make([]int32, 0, 2*len(segments))
	for i, s := range segments {
		startIndex[i] = vertexIndex(s.startPoint)
		endIndex[i] = vertexIndex(s.endPoint)
		adjNext = append(adjNext, adjHead[startIndex[i]], adjHead[endIndex[i]])
		adjHead[startIndex[i]] = int32(i + 1)
		adjHead[endIndex[i]] = int32(i + 1)
	}

	used := make([]bool, len(segments))
	// takeUnused returns and marks an unused segment incident to vertex v.
	takeUnused := func(v int32) (int, bool) {
		for e := adjHead[v]; e != 0; {
			i := int(e - 1)
			if startIndex[i] == v {
				e = adjNext[2*i]
			} else {
				e = adjNext[2*i+1]
			}
			if !used[i] {
				used[i] = true
				return i, true
			}
		}
		return -1, false
	}

	optimized := make([]outlineSegment, 0, len(segments))

	// Scratch buffers for extending a contour in both directions.
	var forward, backward []outlineSegment
	for i := range segments {
		if used[i] {
			continue
		}
		used[i] = true
		forward = forward[:0]
		backward = backward[:0]

		// Extend forward from segments[i].endPoint.
		v := endIndex[i]
		for {
			j, ok := takeUnused(v)
			if !ok {
				break
			}
			if startIndex[j] == v {
				forward = append(forward, outlineSegment{startPoint: segments[j].startPoint, endPoint: segments[j].endPoint})
				v = endIndex[j]
			} else {
				forward = append(forward, outlineSegment{startPoint: segments[j].endPoint, endPoint: segments[j].startPoint})
				v = startIndex[j]
			}
		}
		// Extend backward from segments[i].startPoint.
		v = startIndex[i]
		for {
			j, ok := takeUnused(v)
			if !ok {
				break
			}
			if startIndex[j] == v {
				backward = append(backward, outlineSegment{startPoint: segments[j].endPoint, endPoint: segments[j].startPoint})
				v = endIndex[j]
			} else {
				backward = append(backward, outlineSegment{startPoint: segments[j].startPoint, endPoint: segments[j].endPoint})
				v = startIndex[j]
			}
		}

		// Emit: reversed backward chain, then segment i, then forward chain.
		for k := len(backward) - 1; k >= 0; k-- {
			optimized = append(optimized, backward[k])
		}
		optimized = append(optimized, segments[i])
		optimized = append(optimized, forward...)
	}

	// BUILD OUTLINE
	ops := new(op.Ops)
	var p clip.Path
	p.Begin(ops)
	var prevEnd f32.Point
	for i, seg := range optimized {
		if i == 0 || seg.startPoint != prevEnd {
			p.MoveTo(seg.startPoint)
		}
		p.LineTo(seg.endPoint)
		prevEnd = seg.endPoint
	}
	return p.End()
}
