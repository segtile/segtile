package app_config

import (
	"image"
	"image/color"

	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// Scale factors converting the -1..1 / 0..1 slider values into pixel radius, relative to image width.
const shrinkGrowMaxFactor = 0.10
const smoothMaxFactor = 0.05

const (
	ManualFixNone   uint16 = 0
	ManualFixAdd    uint16 = 1
	ManualFixRemove uint16 = 2
)

// ApplyManualFixMask combines the base mask (from superpixel selection) and the manual fix mask (from user brush strokes).
func ApplyManualFixMask(baseMask *image.Gray16, manualFixMask *image.Gray16) *image.Gray16 {

	var bounds image.Rectangle

	switch {
	case baseMask != nil:
		bounds = baseMask.Bounds()
	case manualFixMask != nil:
		bounds = manualFixMask.Bounds()
	default:
		return image.NewGray16(image.Rect(0, 0, 0, 0))
	}

	result := image.NewGray16(bounds)
	width := bounds.Dx()
	height := bounds.Dy()

	// Copy the base mask into result via direct Pix access (row-by-row to respect strides).
	if baseMask != nil {
		srcStride := baseMask.Stride
		dstStride := result.Stride
		rowBytes := width * 2
		for y := 0; y < height; y++ {
			srcOff := (bounds.Min.Y+y-baseMask.Bounds().Min.Y)*srcStride + (bounds.Min.X-baseMask.Bounds().Min.X)*2
			dstOff := y * dstStride
			copy(result.Pix[dstOff:dstOff+rowBytes], baseMask.Pix[srcOff:srcOff+rowBytes])
		}
	}

	if manualFixMask != nil {
		fixPix := manualFixMask.Pix
		fixStride := manualFixMask.Stride
		resPix := result.Pix
		resStride := result.Stride
		// Manual fix values are small (0/1/2), so the high byte is always 0 and only the low byte matters.
		for y := 0; y < height; y++ {
			fixRow := y * fixStride
			resRow := y * resStride
			for x := 0; x < width; x++ {
				off := x * 2
				switch fixPix[fixRow+off+1] {
				case byte(ManualFixAdd):
					resPix[resRow+off] = 0xFF
					resPix[resRow+off+1] = 0xFF
				case byte(ManualFixRemove):
					resPix[resRow+off] = 0
					resPix[resRow+off+1] = 0
				}
			}
		}
	}

	return result
}

func (appState *AppState) ApplyManualFixAt(x int, y int, radius int, add bool) {

	// Early return if there is no image
	if appState.ActiveImage == nil {
		return
	}

	// Create a new manual fix mask if the current one is missing or has the wrong size
	if appState.ManualFixMask == nil || appState.ManualFixMask.Bounds() != appState.ActiveImage.Bounds() {
		appState.ManualFixMask = image.NewGray16(appState.ActiveImage.Bounds())
	}

	// Cap radius and precompute squared radius
	if radius < 1 {
		radius = 1
	}
	radius2 := radius * radius

	// Set manual fix mode
	manualFixValue := ManualFixRemove
	if add {
		manualFixValue = ManualFixAdd
	}

	if appState.ManualFixMask == nil {
		return
	}

	// Precompute the target area for the circular brush
	targetBbox := superpixel_utils.BoundingBox{}
	targetBbox.AddPoint(x-radius, y-radius)
	targetBbox.AddPoint(x+radius, y+radius)
	targetBounds := targetBbox.Bounds()

	for py := targetBounds.Min.Y; py < targetBounds.Max.Y; py++ {
		for px := targetBounds.Min.X; px < targetBounds.Max.X; px++ {
			dx := px - x
			dy := py - y
			if dx*dx+dy*dy <= radius2 {
				appState.ManualFixMask.SetGray16(px, py, color.Gray16{Y: manualFixValue})
			}
		}
	}

	// Defer the expensive mask rebuild so multiple brush stamps within one frame only trigger
	// a single ReprocessSelection at the start of the next frame.
	appState.ManualFixPending = true
	if appState.Window != nil {
		appState.Window.Invalidate()
	}
}

// FlushPendingManualFixes rebuilds the processed selection once if any brush stamps are pending.
// Call once per frame, before building the UI. Returns true if a rebuild happened.
func (appState *AppState) FlushPendingManualFixes() bool {
	if !appState.ManualFixPending {
		return false
	}
	appState.ManualFixPending = false
	appState.ReprocessSelection()
	return true
}

func SliderToSignedShrinkGrow(sliderValue float32) float32 {
	return sliderValue*2 - 1
}

func SignedShrinkGrowToSlider(signedAmount float32) float32 {
	return (signedAmount + 1) / 2
}

// ReprocessSelection rebuilds ProcessedSelectionMask and the yellow selection outline
func (appState *AppState) ReprocessSelection() {

	if appState.ActiveSp.LabelMap == nil {
		appState.ProcessedSelectionMask = nil
		appState.SelectionPathSpec = rendering.GetEmptySelectionOutline()
		return
	}

	fixedMask := appState.EffectiveSelectionMask()

	if fixedMask == nil || !superpixel_utils.HasNonZeroPixels(fixedMask) {
		appState.ProcessedSelectionMask = nil
		appState.SelectionPathSpec = rendering.GetEmptySelectionOutline()
		return
	}

	if !appState.SmoothSelectionOutlines {
		appState.ProcessedSelectionMask = fixedMask
		appState.SelectionPathSpec = rendering.CalculateOutlineFromMask(appState.ProcessedSelectionMask)
		return
	}

	width := float32(fixedMask.Bounds().Dx())
	shrinkGrowRadius := int(SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value) * width * shrinkGrowMaxFactor)
	smoothRadius := int(appState.SmoothAmount.Value * width * smoothMaxFactor)

	processed := superpixel_utils.GrowShrink(fixedMask, shrinkGrowRadius)
	processed = superpixel_utils.SmoothMask(processed, smoothRadius)

	appState.ProcessedSelectionMask = processed
	appState.SelectionPathSpec = rendering.CalculateOutlineFromMask(processed)
}
