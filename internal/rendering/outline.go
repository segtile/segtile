package rendering

import (
	"image"
	"log"
	"sync"

	"gioui.org/f32"
	"gioui.org/op"
	"gioui.org/op/clip"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
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
func CalculateOutline(mask superpixel_map.Map, targetValue uint32) clip.PathSpec {
	// Start measuring the time for outline calculation.
	//startTime := time.Now()
	defer func() {
		//elapsed := time.Since(startTime)
		//log.Printf("Rebuilt outline for %d in %s", targetValue, elapsed)
	}()

	// GATHER BORDERS

	bounds := mask.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= 0 || height <= 0 {
		return GetEmptySelectionOutline()
	}

	collectedSegments := make(chan outlineSegment)
	var wg sync.WaitGroup

	// First pass: Vertical border segments between horizontally adjacent pixels.
	for x := 0; x < width-1; x++ {
		wg.Add(1)
		go func(x int) {
			defer wg.Done()
			startPoint := f32.Point{X: -1, Y: -1}
			for y := 0; y < height; y++ {
				matchA := mask.ValueAt(x, y) == targetValue
				matchB := mask.ValueAt(x+1, y) == targetValue
				if matchA != matchB {
					// The two labels are different: This means that there is a boundary
					if startPoint.X < 0 {
						// We are not currently in an ongoing boundary segment, so start one
						startPoint = f32.Point{X: float32(x + 1), Y: float32(y)}
					}
				} else {
					if startPoint.X >= 0 {
						// The boundary has ended
						collectedSegments <- outlineSegment{
							startPoint: startPoint,
							endPoint:   f32.Point{X: float32(x + 1), Y: float32(y)},
						}
						startPoint = f32.Point{X: -1, Y: -1}
					}
				}
			}
			if startPoint.X >= 0 {
				collectedSegments <- outlineSegment{
					startPoint: startPoint,
					endPoint:   f32.Point{X: float32(x), Y: float32(height)},
				}
			}
		}(x)
	}

	// Horizontal border segments between vertically adjacent pixels.
	for y := 0; y < height-1; y++ {
		wg.Add(1)
		go func(y int) {
			defer wg.Done()
			startPoint := f32.Point{X: -1, Y: -1}
			for x := 0; x < width; x++ {
				matchA := mask.ValueAt(x, y) == targetValue
				matchB := mask.ValueAt(x, y+1) == targetValue
				if matchA != matchB {
					// The two labels are different: This means that there is a boundary
					if startPoint.X < 0 {
						// We are not currently in an ongoing boundary segment, so start one
						startPoint = f32.Point{X: float32(x), Y: float32(y + 1)}
					}
				} else {
					if startPoint.X >= 0 {
						collectedSegments <- outlineSegment{
							startPoint: startPoint,
							endPoint:   f32.Point{X: float32(x), Y: float32(y + 1)},
						}
						startPoint = f32.Point{X: -1, Y: -1}
					}
				}
			}
			if startPoint.X >= 0 {
				collectedSegments <- outlineSegment{
					startPoint: startPoint,
					endPoint:   f32.Point{X: float32(width), Y: float32(y)},
				}
			}
		}(y)
	}

	go func() {
		wg.Wait()
		close(collectedSegments)
	}()

	// BUILD OUTLINE
	ops := new(op.Ops)
	var p clip.Path
	p.Begin(ops)
	for seg := range collectedSegments {
		if seg.startPoint.X < 0 || seg.endPoint.X < 0 || seg.startPoint.Y < 0 || seg.endPoint.Y < 0 {
			log.Println("Invalid segment:", seg.startPoint, "->", seg.endPoint)
			continue
		}
		p.MoveTo(seg.startPoint)
		p.LineTo(seg.endPoint)
	}

	return p.End()
}
