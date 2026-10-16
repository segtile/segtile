package ui

import (
	"log"
	"strconv"

	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/core"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
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
					if v != appState.ActiveSettings.Superpixel.SpAlgorithmParameters[param] {
						appState.SelectionMaskDiffersVersionOnDisk = true
					}
					appState.ActiveSettings.Superpixel.SpAlgorithmParameters[param] = v
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

	HandleThemeEditors(appState, gtx)

}

// HandleThemeEditors processes the theme section's color editors and stroke width sliders
// and persists changes to the project settings.
func HandleThemeEditors(appState *core.AppState, gtx layout.Context) {
	if appState.ActiveSettings == nil || appState.ActiveSettings.Theme == nil {
		return
	}
	theme := appState.ActiveSettings.Theme

	// Handle outline color channel submissions.
	themeColors := theme.OutlineColors()
	colorChanged := false
	for colorIndex, c := range themeColors {
		channels := [3]*uint8{&c.R, &c.G, &c.B}
		for channelIndex, channel := range channels {
			editor := &appState.ThemeColorEditors[colorIndex][channelIndex]
			for {
				ev, ok := editor.Update(gtx)
				if !ok {
					break
				}
				if _, ok := ev.(widget.SubmitEvent); ok {
					if v, err := app_theme.ParseColorChannel(editor.Text()); err == nil {
						if v != *channel {
							*channel = v
							colorChanged = true
						}
					} else {
						// Invalid input: revert the editor to the current value.
						editor.SetText(app_theme.FormatColorChannel(*channel))
					}
				}
			}
		}
	}
	if colorChanged {
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
		appState.Window.Invalidate()
	}

	// Handle stroke width slider changes; persist when the user lets go.
	selectionWidthWasDragging := appState.ThemeSelectionWidthSlider.Dragging()
	if appState.ThemeSelectionWidthSlider.Update(gtx) {
		theme.SelectionStrokeWidth = core.SliderToStrokeWidth(appState.ThemeSelectionWidthSlider.Value)
		appState.Window.Invalidate()
	}
	hoverWidthWasDragging := appState.ThemeHoverWidthSlider.Dragging()
	if appState.ThemeHoverWidthSlider.Update(gtx) {
		theme.HoverStrokeWidth = core.SliderToStrokeWidth(appState.ThemeHoverWidthSlider.Value)
		appState.Window.Invalidate()
	}
	if (selectionWidthWasDragging && !appState.ThemeSelectionWidthSlider.Dragging()) ||
		(hoverWidthWasDragging && !appState.ThemeHoverWidthSlider.Dragging()) {
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
	}
}
