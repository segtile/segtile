package app_config

import (
	"image"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func (appState *AppState) SelectionMade() bool {

	for x := appState.ActiveSp.LabelMap.Bounds().Min.X; x < appState.ActiveSp.LabelMap.Bounds().Max.X; x++ {
		for y := appState.ActiveSp.LabelMap.Bounds().Min.Y; y < appState.ActiveSp.LabelMap.Bounds().Max.Y; y++ {
			spSelected := appState.SelectionIndex[superpixel_utils.UnsafeGray16At(appState.ActiveSp.LabelMap, x, y)]
			notCancelled := superpixel_utils.UnsafeGray16At(appState.ManualFixMask, x, y) != ManualFixRemove
			manuallyAdded := superpixel_utils.UnsafeGray16At(appState.ManualFixMask, x, y) == ManualFixAdd
			if (spSelected && notCancelled) || manuallyAdded {
				return true
			}
		}
	}

	return false
}

func (appState *AppState) EffectiveSelectionMask() *image.Gray16 {
	var baseMask *image.Gray16
	if appState.ActiveSp.LabelMap != nil {
		baseMask = image.NewGray16(appState.ActiveSp.LabelMap.Bounds())
		if len(appState.SelectionIndex) > 0 {
			baseMask = appState.ActiveSp.SelectionToMask(appState.SelectionIndex)
		}
	}

	combined := ApplyManualFixMask(baseMask, appState.ManualFixMask)
	if combined == nil {
		return nil
	}
	return combined
}

func (appState *AppState) ResetManualFixes() {
	if appState.ActiveImage == nil {
		appState.ManualFixMask = nil
		return
	}

	appState.ManualFixMask = image.NewGray16(appState.ActiveImage.Bounds())
}

func (appState *AppState) ClearSelection(showToast bool) {
	for i := range appState.SelectionIndex {
		appState.SelectionIndex[i] = false
	}
	appState.ResetManualFixes()
	appState.ReprocessSelection()
	appState.Window.Invalidate()
	if showToast {
		appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "Cleared selection.", 1*time.Second)
	}
}

func (appState *AppState) SetManualFixMode(enabled bool) {
	if appState.ManualFixMode == enabled {
		return
	}
	appState.ManualFixMode = enabled
	appState.RefreshHoverPreview()
	appState.Window.Invalidate()
}

func (appState *AppState) RefreshHoverPreview() {
	if !appState.InImageBounds || appState.ActiveSp.LabelMap == nil {
		if appState.HoverCached {
			appState.HoverPathSpec = rendering.GetEmptySelectionOutline()
			appState.HoverCached = false
		}
		return
	}

	if appState.ManualFixMode {
		x := int(appState.LastHoverImagePosition.X)
		y := int(appState.LastHoverImagePosition.Y)
		radius := appState.ManualFixBrushRadius
		if appState.HoverCached && appState.HoverCacheManual && appState.HoverCacheX == x && appState.HoverCacheY == y && appState.HoverCacheRadius == radius {
			return
		}
		appState.HoverPathSpec = rendering.CalculateCircleOutline(x, y, radius)
		appState.HoverCached = true
		appState.HoverCacheManual = true
		appState.HoverCacheX = x
		appState.HoverCacheY = y
		appState.HoverCacheRadius = radius
		return
	}

	label := appState.ActiveSp.LabelMap.Gray16At(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y)).Y
	if appState.HoverCached && !appState.HoverCacheManual && appState.HoverCacheLabel == label {
		return
	}
	appState.HoverPathSpec = rendering.CalculateSelectionOutlineFromIndex(appState.ActiveSp.LabelMap, label)
	appState.HoverCached = true
	appState.HoverCacheManual = false
	appState.HoverCacheLabel = label
}
