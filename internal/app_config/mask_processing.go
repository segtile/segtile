package app_config

import (
	"image"
	"image/color"
	"image/draw"

	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// Scale factors converting the -1..1 / 0..1 slider values into pixel radius, relative to image width.
const shrinkGrowMaxFactor = 0.10
const smoothMaxFactor = 0.05

const (
	ManualFixNone uint16 = iota
	ManualFixAdd
	ManualFixRemove
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
	if baseMask != nil {
		draw.Draw(result, bounds, baseMask, bounds.Min, draw.Src)
	}

	if manualFixMask != nil {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				switch manualFixMask.Gray16At(x, y).Y {
				case ManualFixAdd:
					result.SetGray16(x, y, color.Gray16{Y: 65535})
				case ManualFixRemove:
					result.SetGray16(x, y, color.Gray16{})
				}
			}
		}
	}

	return result
}

func (appState *AppState) ApplyManualFixAt(x int, y int, add bool) {

	// Early return if there is no image
	if appState.ActiveImage == nil {
		return
	}

	// Create a new manual fix mask if the current one is missing or has the wrong size
	if appState.ManualFixMask == nil || appState.ManualFixMask.Bounds() != appState.ActiveImage.Bounds() {
		appState.ManualFixMask = image.NewGray16(appState.ActiveImage.Bounds())
	}

	radius := appState.ManualFixBrushRadius
	if radius < 1 {
		radius = 1
	}

	manualFixValue := ManualFixRemove
	if add {
		manualFixValue = ManualFixAdd
	}
	applyBrush := func(mask *image.Gray16) {
		if mask == nil {
			return
		}
		bounds := mask.Bounds()
		for py := bounds.Min.Y; py < bounds.Max.Y; py++ {
			for px := bounds.Min.X; px < bounds.Max.X; px++ {
				dx := px - x
				dy := py - y
				if dx*dx+dy*dy <= radius*radius {
					mask.SetGray16(px, py, color.Gray16{Y: manualFixValue})
				}
			}
		}
	}

	applyBrush(appState.ManualFixMask)

	appState.ReprocessSelection()
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
