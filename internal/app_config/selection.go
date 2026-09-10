package app_config

import (
	"image"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func (appState *AppState) SelectionMade() bool {
	mask := appState.EffectiveSelectionMask()
	return mask != nil && superpixel_utils.HasNonZeroPixels(mask)
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
		appState.HoverPathSpec = rendering.GetEmptySelectionOutline()
		return
	}

	if appState.ManualFixMode {
		appState.HoverPathSpec = rendering.CalculateCircleOutline(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y), appState.ManualFixBrushRadius)
		return
	}

	label := appState.ActiveSp.LabelMap.Gray16At(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y)).Y
	appState.HoverPathSpec = rendering.CalculateSelectionOutlineFromIndex(appState.ActiveSp.LabelMap, label)
}
