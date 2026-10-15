package ui

import (
	"log"
	"strconv"

	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/core"
)

// HandleEditors processes all superpixel parameter editor events and applies the changes.
func HandleEditors(appState *core.AppState, gtx layout.Context) {
	if appState.UiLocked() {
		return
	}

	recalculate := false

	// Handle settings changes.
	for i := range appState.SuperpixelParameterEditors {
		param := appState.SuperpixelParameterEditors[i].Id
		editor := &appState.SuperpixelParameterEditors[i].Editor
		// Drain events and apply the value on Submit
		for {
			ev, ok := editor.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				if v, err := strconv.ParseFloat(editor.Text(), 64); err == nil {
					if v != appState.Settings.Superpixel.SpAlgorithmParameters[param] {
						appState.SelectionDiffersVersionOnDisk = true
					}
					appState.Settings.Superpixel.SpAlgorithmParameters[param] = v
					log.Printf("Set %s to %v\n", param, v)

					recalculate = true
				}
			}
		}
	}

	// Handle Shrink/Grow and Smooth slider changes.
	shrinkGrowWasDragging := appState.ShrinkGrowAmount.Dragging()
	if appState.ShrinkGrowAmount.Update(gtx) {
		newValue := core.SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value)
		if newValue != appState.Settings.ShrinkGrowAmount {
			appState.SelectionDiffersVersionOnDisk = true
		}
		appState.Settings.ShrinkGrowAmount = newValue
		appState.ReprocessSelection()
		appState.Window.Invalidate()
	}
	smoothWasDragging := appState.SmoothAmount.Dragging()
	if appState.SmoothAmount.Update(gtx) {
		if appState.SmoothAmount.Value != appState.Settings.SmoothAmount {
			appState.SelectionDiffersVersionOnDisk = true
		}
		appState.Settings.SmoothAmount = appState.SmoothAmount.Value
		appState.ReprocessSelection()
		appState.Window.Invalidate()
	}

	// Persist config if user just let go of slider.
	if (shrinkGrowWasDragging && !appState.ShrinkGrowAmount.Dragging()) || (smoothWasDragging && !appState.SmoothAmount.Dragging()) {
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}

	if recalculate {

		appState.SyncEditors()

		if appState.ActiveImage != nil {
			appState.RecalculateSuperpixels(true)
		}

		// Persist updated parameters to config.
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}

	}

	// Persist the auto flood fill setting whenever it is toggled.
	if appState.EnableAutoFloodFill.Update(gtx) {
		appState.Settings.EnableAutoFloodFill = appState.EnableAutoFloodFill.Value
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}

	// Persist the keep-selection setting whenever it is toggled.
	if appState.UpdateSelectionOnSpChange.Update(gtx) {
		appState.Settings.UpdateSelectionOnSpChange = appState.UpdateSelectionOnSpChange.Value
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}

}
