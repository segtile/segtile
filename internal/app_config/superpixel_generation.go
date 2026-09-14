package app_config

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

func (appState *AppState) RecalculateSuperpixels(preserveSelection bool, showToast bool) error {

	log.Println("Starting to calculate superpixels...")
	start := time.Now().UnixMicro()

	// Snapshot the current selection as a mask before the labels change
	var oldSelectionMask *image.Gray16
	if preserveSelection && appState.ActiveSp.SpCount > 0 && len(appState.SelectionIndex) > 0 {
		oldSelectionMask = appState.ActiveSp.SelectionToMask(appState.SelectionIndex)
	}

	// Re-run the superpixel algorithm
	newResult, err := superpixel.Run(
		appState.Settings.Superpixel.SpAlgorithmId,
		appState.ActiveImage,
		appState.Settings.Superpixel.SpAlgorithmParameters)
	if err != nil {
		return err
	}

	core := time.Now().UnixMicro()
	log.Printf("Finished core processing after %.1f ms\n", float64(core-start)/1000)

	newResult.EnforceConnectivity()
	newResult.EnforceContiguousIndexing()
	newResult.SetActualSpCount()

	appState.ActiveSp = newResult
	// The label map changed, so any cached hover outline is now invalid.
	appState.HoverCached = false
	postprocessed := time.Now().UnixMicro()
	log.Printf("Finished post-processing after %.1f ms\n", float64(postprocessed-start)/1000)

	// Restore the selection by matching old mask pixels to new superpixels,
	// or reset to an empty selection when there was no previous result.
	if oldSelectionMask != nil {
		appState.SelectionIndex = appState.ActiveSp.SelectFromMask(oldSelectionMask, 0.5)
	} else {
		appState.SelectionIndex = make([]bool, appState.ActiveSp.SpCount)
	}

	// Rebuild the yellow selection outline (processed) and orange on-disk outline is left untouched
	appState.ReprocessSelection()

	selectionRestored := time.Now().UnixMicro()
	log.Printf("Finished restoring selection after %.1f ms\n", float64(selectionRestored-start)/1000)

	if showToast {
		// Add a toast
		appState.Toasts.ShowType(time.Now(), giotoast.TypeSuccess, fmt.Sprintf("Rebuilt superpixels (%.1f ms)", float64(selectionRestored-start)/1000), 500*time.Millisecond)
	}
	return nil

}
