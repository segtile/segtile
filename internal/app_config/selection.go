package app_config

import (
	"image"
	"time"

	"github.com/reiver/go-giotoast"
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
