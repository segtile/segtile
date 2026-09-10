package ui

import (
	"log"
	"strconv"

	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/app_config"
)

// HandleEditors processes all superpixel parameter editor events and applies the changes.
func HandleEditors(appState *app_config.AppState, gtx layout.Context) {
	if appState.UiLocked() {
		return
	}

	recalculate := false

	// Handle settings changes.
	for i := range appState.SuperpixelParameterEditors {
		param := appState.SuperpixelParameterEditorIds[i]
		editor := &appState.SuperpixelParameterEditors[i]
		// Drain events and apply the value on Submit
		for {
			ev, ok := editor.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				if v, err := strconv.ParseFloat(editor.Text(), 64); err == nil {
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
		appState.Settings.ShrinkGrowAmount = app_config.SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value)
		appState.ReprocessSelection()
		appState.Window.Invalidate()
	}
	smoothWasDragging := appState.SmoothAmount.Dragging()
	if appState.SmoothAmount.Update(gtx) {
		appState.Settings.SmoothAmount = appState.SmoothAmount.Value
		appState.ReprocessSelection()
		appState.Window.Invalidate()
	}

	// Persist config if user just let go of slider.
	if (shrinkGrowWasDragging && !appState.ShrinkGrowAmount.Dragging()) || (smoothWasDragging && !appState.SmoothAmount.Dragging()) {
		if err := app_config.Save(appState.Settings); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}

	if recalculate {

		appState.SyncEditors()

		if appState.ActiveImage != nil {
			appState.RecalculateSuperpixels(true, true)
		}

		// Persist updated parameters to config.
		if err := app_config.Save(appState.Settings); err != nil {
			log.Println("Warning: could not save config:", err)
		}

	}

	// Persist the auto flood fill setting whenever it is toggled.
	if appState.EnableAutoFloodFill.Update(gtx) {
		appState.Settings.EnableAutoFloodFill = appState.EnableAutoFloodFill.Value
		if err := app_config.Save(appState.Settings); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}

}
