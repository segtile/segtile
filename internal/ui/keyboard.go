package ui

import (
	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/struffel/annotation-tool/internal/app_config"
)

func HandleKeyboard(appState *app_config.AppState, gtx layout.Context) {
	if appState.CurrentlyOptimizing {
		return
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &appState.KeyboardTag},
			key.Filter{Focus: &appState.KeyboardTag, Name: "F", Optional: key.ModCtrl},
			key.Filter{Focus: &appState.KeyboardTag, Name: "C", Optional: key.ModCtrl},
			key.Filter{Focus: &appState.KeyboardTag, Name: "S", Optional: key.ModCtrl},
			key.Filter{Focus: &appState.KeyboardTag, Name: "D", Optional: key.ModCtrl},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameAlt},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameSpace, Optional: key.ModShift},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameLeftArrow},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameRightArrow},
		)
		if !ok {
			break
		}
		ke, ok := ev.(key.Event)
		if !ok {
			continue
		}

		if ke.Name == key.NameAlt {
			appState.ManualFixMode = ke.State == key.Press
			refreshHoverPreview(appState)
			appState.Window.Invalidate()
			continue
		}

		// [F]it to window
		if ke.State == key.Press && ke.Name == "F" {
			appState.FitToWindow = true
			appState.Window.Invalidate()
			continue
		}

		// [C]lear selection
		if ke.State == key.Press && ke.Name == "C" {
			appState.ClearSelection(true)
			continue
		}

		// [S]ave mask
		if ke.State == key.Press && ke.Name == "S" {
			if appState.SelectionMade() || showSaveEmptyMaskWarning() {
				appState.SaveCurrentMask(true, true)
			}
			continue
		}

		// [D]isable preview (while held)
		if ke.Name == "D" {
			if ke.State == key.Press {
				appState.RenderSelectionOutlines = false
				appState.ReprocessSelection()
				appState.Window.Invalidate()
			} else if ke.State == key.Release {
				appState.RenderSelectionOutlines = true
				appState.ReprocessSelection()
				appState.Window.Invalidate()
			}
			continue
		}

		// Shift: Disable smoothing of selection outline (while held)
		/*if ke.Name == key.NameShift {
			if ke.State == key.Press {
				appState.SmoothSelectionOutlines = false
				appState.ReprocessSelection()
				log.Println("Disable smoothing of selection outline")
				appState.Window.Invalidate()
			} else if ke.State == key.Release {
				appState.SmoothSelectionOutlines = true
				appState.ReprocessSelection()
				log.Println("Enable smoothing of selection outline")
				appState.Window.Invalidate()
			}
		}*/

		// Space: next image, Shift+Space: previous image
		// Alternatively: Left/Right arrow keys
		if ke.State == key.Press && ke.Name == key.NameSpace {
			if ke.Modifiers.Contain(key.ModShift) {
				appState.LoadPreviousImage()
			} else {
				appState.LoadNextImage()
			}
			continue
		}
		if ke.State == key.Press && ke.Name == key.NameLeftArrow {
			appState.LoadPreviousImage()
			continue
		}
		if ke.State == key.Press && ke.Name == key.NameRightArrow {
			appState.LoadNextImage()
			continue
		}
	}
}
