package core

import (
	"fmt"
	"log"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/segtile/segtile/pkg/superpixel"
)

func (appState *AppState) RecalculateSuperpixels(showToast bool) error {

	log.Println("Starting to calculate superpixels...")
	startTimestamp := time.Now().UnixMicro()

	// Re-run the core of the superpixel algorithm
	newResult, err := superpixel.Run(
		appState.ActiveSettings.Superpixel.SpAlgorithmId,
		appState.ActiveImage,
		appState.ActiveSettings.Superpixel.SpAlgorithmParameters)
	if err != nil {
		return err
	}

	// Merge the new SP into the app state
	appState.ActiveSp = newResult

	// The label map has changed, so any cached hover outline is now invalid.
	appState.HoverCached = false

	// Snap the existing selection mask to the new superpixel segmentation or keep it as-is.
	if appState.UpdateSelectionOnSpChange.Value && appState.SelectionMaskRaw != nil && appState.SelectionMaskRaw.Bounds() == appState.ActiveImage.Bounds() {
		newSpSelection := appState.ActiveSp.SelectFromMask(appState.SelectionMaskRaw, 0.5)
		snapped := appState.ActiveSp.SelectionToMask(newSpSelection)
		if !snapped.Equals(appState.SelectionMaskRaw) {
			appState.SelectionMaskRaw = snapped
			appState.SelectionMaskDirty = true
			appState.SelectionMaskDiffersVersionOnDisk = true
		}
	}

	// The label map changed, so any tracked superpixel labels are stale.
	appState.LastPaintedSpLabel = -1

	// Rebuild the selection outlines
	appState.ReprocessSelection()

	endTimestamp := time.Now().UnixMicro()
	log.Printf("Finished recalculating superpixels in %.1f ms\n", float64(endTimestamp-startTimestamp)/1000)

	if showToast {
		// Add a toast
		appState.Toasts.ShowType(time.Now(), giotoast.TypeSuccess, fmt.Sprintf("Rebuilt superpixels (%.1f ms)", float64(endTimestamp-startTimestamp)/1000), 500*time.Millisecond)
	}
	return nil

}
