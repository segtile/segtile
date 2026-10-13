package core

import (
	"gioui.org/op/clip"

	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// Scale factors converting the -1..1 / 0..1 slider values into pixel radius, relative to image width.
const shrinkGrowMaxFactor = 0.10
const smoothMaxFactor = 0.05

func (appState *AppState) ApplyManualFixAt(x int, y int, add bool) {

	// Early return if there is no image
	if appState.ActiveImage == nil || appState.SelectionMask == nil {
		return
	}

	// Cap radius and precompute squared radius
	if appState.ManualFixBrushRadius < 1 {
		appState.ManualFixBrushRadius = 1
	}
	radius2 := appState.ManualFixBrushRadius * appState.ManualFixBrushRadius

	// Precompute the target area for the circular brush
	targetBbox := superpixel_utils.BoundingBox{}
	targetBbox.AddPoint(x-appState.ManualFixBrushRadius, y-appState.ManualFixBrushRadius)
	targetBbox.AddPoint(x+appState.ManualFixBrushRadius, y+appState.ManualFixBrushRadius)
	targetBounds := targetBbox.Bounds().Intersect(appState.SelectionMask.Bounds())

	for py := targetBounds.Min.Y; py < targetBounds.Max.Y; py++ {
		for px := targetBounds.Min.X; px < targetBounds.Max.X; px++ {
			dx := px - x
			dy := py - y
			if dx*dx+dy*dy <= radius2 {
				if add {
					appState.SelectionMask.SetValueAt(px, py, true)
				} else {
					appState.SelectionMask.SetValueAt(px, py, false)
				}
			}
		}
	}

	// Defer the expensive mask rebuild so multiple brush stamps within one frame only trigger
	// a single ReprocessSelection at the start of the next frame.
	appState.SelectionMaskDirty = true
	if appState.Window != nil {
		appState.Window.Invalidate()
	}
}

// FlushPendingManualFixes rebuilds the processed selection once if any brush stamps are pending.
// Call once per frame, before building the UI. Returns true if a rebuild happened.
func (appState *AppState) FlushPendingManualFixes() bool {
	if !appState.SelectionMaskDirty {
		return false
	}
	appState.SelectionMaskDirty = false
	appState.ReprocessSelection()
	return true
}

func SliderToSignedShrinkGrow(sliderValue float32) float32 {
	return sliderValue*2 - 1
}

func SignedShrinkGrowToSlider(signedAmount float32) float32 {
	return (signedAmount + 1) / 2
}

// ReprocessSelection rebuilds ProcessedSelectionMask and the selection outlines.
// Callers that know the inputs changed (mask mutated, smoothing parameters
// adjusted, superpixels rebuilt) call this directly; callers on high-frequency
// input paths that merely suspect a change should use ReprocessSelectionIfChanged.
func (appState *AppState) ReprocessSelection() {

	if appState.ActiveSp.LabelMap == nil {
		appState.ProcessedSelectionMask = nil
		appState.SelectionPathSpec = rendering.GetEmptySelectionOutline()
		return
	}

	if appState.SelectionMask == nil || appState.SelectionMask.IsEmpty() {
		appState.ProcessedSelectionMask = nil
		appState.SelectionPathSpec = rendering.GetEmptySelectionOutline()
		appState.SelectionMaskDirty = false
		return
	}

	if !appState.SmoothSelectionOutlines {
		appState.ProcessedSelectionMask = appState.SelectionMask
		appState.SelectionPathSpec = rendering.CalculateOutline(appState.ProcessedSelectionMask, 0)
		appState.SelectionMaskDirty = false
		return
	}

	width := float32(appState.SelectionMask.Width)
	shrinkGrowRadius := int(SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value) * width * shrinkGrowMaxFactor)
	smoothRadius := int(appState.SmoothAmount.Value * width * smoothMaxFactor)

	processed := superpixel_utils.GrowShrink(appState.SelectionMask, shrinkGrowRadius)
	processed = superpixel_utils.SmoothMask(processed, smoothRadius)

	appState.ProcessedSelectionMask = processed
	appState.SelectionPathSpec = rendering.CalculateOutline(processed, 0)
	appState.SelectionMaskDirty = false
}

func (appState *AppState) ReprocessSelectionIfChanged() {
	if !appState.SelectionMaskDirty && appState.SelectionPathSpec != (clip.PathSpec{}) {
		return
	}
	appState.ReprocessSelection()
}
