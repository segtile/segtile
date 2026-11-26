package core

import (
	"github.com/segtile/segtile/pkg/superpixel_utils"
)

// Scale factors converting the -1..1 / 0..1 slider values into pixel radius, relative to image width.
const shrinkGrowMaxFactor = 0.10
const smoothMaxFactor = 0.05

func (appState *AppState) ApplyManualFixAt(x int, y int, add bool) {

	// Early return if there is no image
	if appState.ActiveImage == nil || appState.SelectionMaskRaw == nil {
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
	targetBounds := targetBbox.Bounds().Intersect(appState.SelectionMaskRaw.Bounds())

	changed := false
	for py := targetBounds.Min.Y; py < targetBounds.Max.Y; py++ {
		for px := targetBounds.Min.X; px < targetBounds.Max.X; px++ {
			dx := px - x
			dy := py - y
			if dx*dx+dy*dy <= radius2 && appState.SelectionMaskRaw.PositiveAt(px, py) != add {
				appState.SelectionMaskRaw.SetValueAt(px, py, add)
				changed = true
			}
		}
	}

	if !changed {
		return
	}

	// Defer the expensive mask rebuild so multiple brush stamps within one frame only trigger
	// a single ReprocessSelection at the start of the next frame.
	appState.SelectionMaskDirty = true
	appState.SelectionMaskDiffersVersionOnDisk = true
	if appState.Window != nil {
		appState.Window.Invalidate()
	}
}

func SliderToSignedShrinkGrow(sliderValue float32) float32 {
	return sliderValue*2 - 1
}

func SignedShrinkGrowToSlider(signedAmount float32) float32 {
	return (signedAmount + 1) / 2
}

// ReprocessSelection rebuilds ProcessedSelectionMask and the selection outlines.
func (appState *AppState) ReprocessSelection() {

	// If there is no active superpixel label map, clear the processed selection.
	if appState.ActiveSp.LabelMap == nil {
		appState.ProcessedSelectionMask = nil
	}

	// If there is a non-empty selection mask, apply postprocessing to it.
	if appState.SelectionMaskRaw != nil && !appState.SelectionMaskRaw.IsEmpty() {
		//processed := applyPostprocessing(appState.SelectionMask, appState.ShrinkGrowAmount.Value, appState.SmoothAmount.Value)

		// Prepare the parameters for postprocessing.
		width := float32(appState.SelectionMaskRaw.Width)
		shrinkGrowRadius := int(SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value) * width * shrinkGrowMaxFactor)
		smoothRadius := int(appState.SmoothAmount.Value * width * smoothMaxFactor)

		// Prepare the new processed selection mask.
		processed := appState.SelectionMaskRaw.Clone()

		// Run the post-processing steps
		superpixel_utils.GrowShrink(processed, shrinkGrowRadius)
		superpixel_utils.SmoothMask(processed, smoothRadius)
		appState.ProcessedSelectionMask = processed
	} else {
		// The selection is empty (or gone entirely), so the processed mask must be
		// cleared as well. Otherwise its stale content keeps producing an outline.
		appState.ProcessedSelectionMask = nil
	}

	appState.ReprocessSelectionOutline()
	appState.SelectionMaskDirty = false
	// log.Println("Reprocessed Selection")
}
