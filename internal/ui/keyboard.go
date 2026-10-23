package ui

import (
	"gioui.org/io/key"
	"gioui.org/layout"
	"github.com/struffel/annotation-tool/internal/core"
)

func HandleKeyboard(appState *core.AppState, gtx layout.Context) {
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
			key.Filter{Focus: &appState.KeyboardTag, Name: "V", Optional: key.ModCtrl},
			key.Filter{Focus: &appState.KeyboardTag, Name: "R", Optional: key.ModCtrl},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameCtrl, Optional: key.ModCtrl},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameSpace, Optional: key.ModShift},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameLeftArrow},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameRightArrow},
		)
		if !ok {
			break
		}

		// If the keyboard focus is lost, always disable manual fix mode
		if focusEvent, ok := ev.(key.FocusEvent); ok {
			if !focusEvent.Focus {
				appState.SetManualFixMode(false)
			}
			continue
		}

		// If the event is not a key event, ignore it
		ke, ok := ev.(key.Event)
		if !ok {
			continue
		}

		// Ctrl key: Enable manual fix mode while held
		if ke.Name == key.NameCtrl {
			appState.SetManualFixMode(ke.State == key.Press)
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
				appState.SaveCurrentMask(true)
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

		// [R]evert to saved annotation
		if ke.State == key.Press && ke.Name == "R" {
			appState.RevertToSavedMask()
			continue
		}

		// [V]iew raw mask: Disable smoothing of selection outline (while held)
		if ke.Name == "V" {
			if ke.State == key.Press {
				appState.SmoothSelectionOutlines = false
				appState.ReprocessSelectionOutline()
				appState.Window.Invalidate()
			} else if ke.State == key.Release {
				appState.SmoothSelectionOutlines = true
				appState.ReprocessSelectionOutline()
				appState.Window.Invalidate()
			}
		}

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
