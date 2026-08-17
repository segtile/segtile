package app_config

import (
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// Scale factors converting the -1..1 / 0..1 slider values into pixel radii, relative to image width.
const shrinkGrowMaxFactor = 0.10
const smoothMaxFactor = 0.05

// ShrinkGrowAmount's widget.Float is restricted to gioui's fixed 0..1 range, but Shrink/Grow is a
// signed amount (negative = shrink, positive = grow). SliderToSignedShrinkGrow/SignedShrinkGrowToSlider
// convert between the widget's 0..1 representation and the persisted/logical -1..1 signed amount,
// so that a zero-value (e.g. missing from an old settings file) maps to a neutral slider position.
func SliderToSignedShrinkGrow(sliderValue float32) float32 {
	return sliderValue*2 - 1
}

func SignedShrinkGrowToSlider(signedAmount float32) float32 {
	return (signedAmount + 1) / 2
}

// RecalculateProcessedSelection rebuilds ProcessedSelectionMask and the yellow selection outline
// (SelectionPathSpec) from the raw selection plus the current Shrink/Grow and Smooth amounts.
// The orange outline (ExistingMaskPathSpec) is unaffected - it only ever shows the on-disk mask.
func (appState *AppState) RecalculateProcessedSelection() {
	if !appState.SelectionMade() {
		appState.ProcessedSelectionMask = nil
		appState.SelectionPathSpec = rendering.GetEmptySelectionOutline()
		return
	}

	rawMask := appState.ActiveSp.SelectionToMask(appState.SelectionIndex)
	width := float32(rawMask.Bounds().Dx())

	shrinkGrowRadius := int(SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value) * width * shrinkGrowMaxFactor)
	smoothRadius := int(appState.SmoothAmount.Value * width * smoothMaxFactor)

	processed := superpixel_utils.GrowShrink(rawMask, shrinkGrowRadius)
	processed = superpixel_utils.SmoothMask(processed, smoothRadius)

	appState.ProcessedSelectionMask = processed
	appState.SelectionPathSpec = rendering.CalculateOutlineFromMask(processed)
}
