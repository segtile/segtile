package ui

import (
	"log"
	"strconv"
	"time"

	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/reiver/go-giotoast"
	"github.com/segtile/segtile/internal/core"
)

// HandleEditors processes all superpixel parameter editor events and applies the changes.
func HandleEditors(appState *core.AppState, gtx layout.Context) {
	if appState.UiLocked() {
		return
	}

	recalculate := false

	// Handle settings changes: If a submit event occurs, parse all editors and apply their values
	for i := range appState.SuperpixelParameterEditors {
		editor := &appState.SuperpixelParameterEditors[i].Editor
		// Drain events and apply the value on Submit
		for {
			ev, ok := editor.Update(gtx)
			if !ok {
				break
			}

			// If a submit event orrcurs, parse all editors and apply their values
			_, ok = ev.(widget.SubmitEvent)
			if ok {
				recalculate = true
			}
		}
	}

	if recalculate {
		for i := range appState.SuperpixelParameterEditors {

			// Not parseable
			paramValue, err := strconv.ParseFloat(appState.SuperpixelParameterEditors[i].Editor.Text(), 64)
			if err != nil {
				appState.Toasts.ShowType(time.Now(), giotoast.TypeError, "Invalid superpixel parameter value for "+appState.SuperpixelParameterEditors[i].Title, 3*time.Second)
				recalculate = false
				break
			}

			// Out of range
			if paramValue < appState.SuperpixelParameterEditors[i].MinValue || paramValue > appState.SuperpixelParameterEditors[i].MaxValue {
				appState.Toasts.ShowType(time.Now(), giotoast.TypeError, "Superpixel parameter value out of range for "+appState.SuperpixelParameterEditors[i].Title, 3*time.Second)
				recalculate = false
				break
			}

		}
	}

	if recalculate {
		for i := range appState.SuperpixelParameterEditors {
			param := appState.SuperpixelParameterEditors[i].Id
			editor := &appState.SuperpixelParameterEditors[i].Editor
			if v, err := strconv.ParseFloat(editor.Text(), 64); err == nil {
				if v != appState.ActiveSettings.Superpixel.SpAlgorithmParameters[param] {
					appState.SelectionMaskDiffersVersionOnDisk = true
				}
				appState.ActiveSettings.Superpixel.SpAlgorithmParameters[param] = v
				log.Printf("Set %s to %v\n", param, v)
				recalculate = true
			}
		}
	}

	// Handle Shrink/Grow and Smooth slider changes.
	shrinkGrowWasDragging := appState.ShrinkGrowAmount.Dragging()
	if appState.ShrinkGrowAmount.Update(gtx) {
		newValue := core.SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value)
		if newValue != appState.ActiveSettings.ShrinkGrowAmount {
			appState.SelectionMaskDiffersVersionOnDisk = true
		}
		appState.ActiveSettings.ShrinkGrowAmount = newValue
		appState.ReprocessSelection()
		appState.Window.Invalidate()
	}
	smoothWasDragging := appState.SmoothAmount.Dragging()
	if appState.SmoothAmount.Update(gtx) {
		if appState.SmoothAmount.Value != appState.ActiveSettings.SmoothAmount {
			appState.SelectionMaskDiffersVersionOnDisk = true
		}
		appState.ActiveSettings.SmoothAmount = appState.SmoothAmount.Value
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
		appState.ActiveSettings.EnableAutoFloodFill = appState.EnableAutoFloodFill.Value
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}

	// Persist the keep-selection setting whenever it is toggled.
	if appState.UpdateSelectionOnSpChange.Update(gtx) {
		appState.ActiveSettings.UpdateSelectionOnSpChange = appState.UpdateSelectionOnSpChange.Value
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}

}
