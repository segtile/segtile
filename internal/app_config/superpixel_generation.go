package app_config

import (
	"fmt"
	"image"
	"log"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

const labelMapSmoothingMaxFactor = 0.05

func (appState *AppState) RecalculateSuperpixels(preserveSelection bool, showToast bool) error {

	log.Println("Recalculating superpixels...")
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

	log.Println("Smoothing label map ...")

	// Apply smoothing
	newResult.SmoothLabelMap(int(appState.LabelMapSmoothingStrength.Value * float32(newResult.LabelMap.Bounds().Dx()) * labelMapSmoothingMaxFactor))
	newResult.EnforceConnectivity()
	newResult.EnforceContiguousIndexing()

	appState.ActiveSp = newResult

	// Restore the selection by matching old mask pixels to new superpixels,
	// or reset to an empty selection when there was no previous result.
	if oldSelectionMask != nil {
		appState.SelectionIndex = appState.ActiveSp.SelectFromMask(oldSelectionMask, 0.5)
	} else {
		appState.SelectionIndex = make([]bool, appState.ActiveSp.SpCount)
	}

	// Rebuild the selection outline path spec for rendering
	appState.SelectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(*appState.ActiveSp.LabelMap, appState.SelectionIndex)

	// Log
	end := time.Now().UnixMicro()
	log.Printf("Finished calculating superpixels in %.1f milliseconds\n", float64(end-start)/1000)

	if showToast {
		// Add a toast
		appState.Toasts.ShowType(time.Now(), giotoast.TypeSuccess, fmt.Sprintf("Rebuilt superpixels (%.1f ms)", float64(end-start)/1000), 500*time.Millisecond)
	}
	return nil

}
