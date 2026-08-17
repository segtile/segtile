package app_config

import "github.com/struffel/annotation-tool/internal/rendering"

// StartSelectionFromScratch resolves the pending existing-mask decision by discarding the
// on-disk mask entirely and letting the user begin a fresh, empty selection.
func (appState *AppState) StartSelectionFromScratch() {
	appState.AwaitingMaskDecision = false
	appState.ActiveImageExistingMask = nil
	appState.ExistingMaskPathSpec = rendering.GetEmptySelectionOutline()
	appState.Window.Invalidate()
}

// EditExistingMaskWithCurrentSettings resolves the pending existing-mask decision by
// approximating the on-disk mask as a selection on the current superpixel segmentation,
// so the user can keep editing it.
func (appState *AppState) EditExistingMaskWithCurrentSettings() {
	appState.AwaitingMaskDecision = false
	if appState.ActiveImageExistingMask != nil {
		appState.SelectionIndex = appState.ActiveSp.SelectFromMask(appState.ActiveImageExistingMask, 0.5)
		appState.RecalculateProcessedSelection()
	}
	appState.Window.Invalidate()
}
