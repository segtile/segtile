package app_config

import (
	"fmt"
	"log"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

func (appState *AppState) RecalculateSuperpixels(showToast bool) error {

	log.Println("Starting to calculate superpixels...")
	startTimestamp := time.Now().UnixMicro()

	// Re-run the core of the superpixel algorithm
	newResult, err := superpixel.Run(
		appState.Settings.Superpixel.SpAlgorithmId,
		appState.ActiveImage,
		appState.Settings.Superpixel.SpAlgorithmParameters)
	if err != nil {
		return err
	}

	coreFinishedTimestamp := time.Now().UnixMicro()
	log.Printf("Finished core processing after %.1f ms\n", float64(coreFinishedTimestamp-startTimestamp)/1000)

	// Run the post-processing steps on the superpixel result
	newResult.EnforceConnectivity()
	newResult.EnforceContiguousIndexing()
	newResult.SetActualSpCount()

	// Merge the new SP into the app state
	appState.ActiveSp = newResult

	// The label map has changed, so any cached hover outline is now invalid.
	appState.HoverCached = false

	postprocessingFinishedTimestamp := time.Now().UnixMicro()
	log.Printf("Finished post-processing after %.1f ms\n", float64(postprocessingFinishedTimestamp-startTimestamp)/1000)

	// Snap the existing selection mask to the new superpixel segmentation or keep it as-is.
	if appState.UpdateSelectionOnSpChange.Value && appState.SelectionMask != nil && appState.SelectionMask.Bounds() == appState.ActiveImage.Bounds() {
		newSpSelection := appState.ActiveSp.SelectFromMask(appState.SelectionMask, 0.5)
		appState.SelectionMask = appState.ActiveSp.SelectionToMask(newSpSelection)
		appState.SelectionMaskDirty = true
	}

	// The label map changed, so any tracked superpixel labels are stale.
	appState.LastSelectedSpLabel = -1
	appState.LastDeselectedSpLabel = -1

	// Rebuild the selection outlines
	appState.ReprocessSelection()

	selectionRestored := time.Now().UnixMicro()
	log.Printf("Finished restoring selection after %.1f ms\n", float64(selectionRestored-startTimestamp)/1000)

	if showToast {
		// Add a toast
		appState.Toasts.ShowType(time.Now(), giotoast.TypeSuccess, fmt.Sprintf("Rebuilt superpixels (%.1f ms)", float64(selectionRestored-startTimestamp)/1000), 500*time.Millisecond)
	}
	return nil

}
