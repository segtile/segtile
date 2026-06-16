package ui

import (
	"fmt"
	"strconv"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/app_config"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/internal/superpixel_interaction"
)

// HandleEditors processes all superpixel parameter editor events and applies the changes.
func HandleEditors(appState *app_config.AppState, gtx layout.Context) {

	recalculate := false

	// Handle settings changes
	for param, editor := range appState.SuperpixelParameterEditors {
		// Drain events and apply the value on Submit
		for {
			ev, ok := editor.Update(gtx)
			if !ok {
				break
			}
			if _, ok := ev.(widget.SubmitEvent); ok {
				if v, err := strconv.ParseFloat(editor.Text(), 64); err == nil {
					appState.SuperpixelImplementation.SetParameter(param, v)
					fmt.Printf("Set %s to %v\n", param.Title(), v)
					recalculate = true
				}
			}
		}
	}
	if recalculate {
		if appState.CurrentImage != nil {
			appState.RecalculateSuperpixels()
		}

		// Persist updated parameters to config.
		if appState.Settings != nil {
			for _, p := range appState.SuperpixelImplementation.GetSupportedParameters() {
				v, _ := appState.SuperpixelImplementation.GetParameter(p)
				if appState.Settings.Superpixel.Parameters == nil {
					appState.Settings.Superpixel.Parameters = make(map[string]float64)
				}
				appState.Settings.Superpixel.Parameters[string(p)] = v
			}
			if err := app_config.Save(appState.Settings, appState.OutputDirectory); err != nil {
				fmt.Println("Warning: could not save config:", err)
			}
		}
	}
}

func HandlePointer(appState *app_config.AppState, gtx layout.Context) {

	// Iterate over all pointer events that occurred since the last frame
	for {

		// Filter for all events that should be handled
		ev, ok := gtx.Event(
			pointer.Filter{
				Target:  &appState.CanvasTag,
				Kinds:   pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll | pointer.Move,
				ScrollY: pointer.ScrollRange{Min: -100, Max: 100},
			},
		)
		if !ok {
			// No more events to process
			break
		}
		pe, ok := ev.(pointer.Event)
		if !ok {
			// Not a pointer event, ignore
			continue
		}

		// No image loaded — skip all image-space pointer logic.
		if appState.CurrentImage == nil {
			continue
		}

		// Always set the keyboard focus to the canvas when hovering on it, so that key commands work without requiring an extra click
		gtx.Execute(key.FocusCmd{Tag: &appState.KeyboardTag})

		// From the raw position of the pointer, calculate the position in image space
		imageX := (pe.Position.X - appState.ViewOffsetX) / appState.ViewScale
		imageY := (pe.Position.Y - appState.ViewOffsetY) / appState.ViewScale

		// Check if the position is within the bounds of the image
		inImageBounds := imageX >= 0 && imageY >= 0 && int(imageX) < appState.CurrentImage.Bounds().Dx() && int(imageY) < appState.CurrentImage.Bounds().Dy()

		// We start with actions that always fire, even if the cursor is outside the image bounds

		// Handle scroll (zoom)
		if pe.Kind == pointer.Scroll {
			// Use fixed 10% per notch regardless of raw scroll magnitude
			var zoomFactor float32
			if pe.Scroll.Y > 0 {
				zoomFactor = 1 / 1.1
			} else {
				zoomFactor = 1.1
			}
			appState.ViewOffsetX = pe.Position.X - (pe.Position.X-appState.ViewOffsetX)*zoomFactor
			appState.ViewOffsetY = pe.Position.Y - (pe.Position.Y-appState.ViewOffsetY)*zoomFactor
			appState.ViewScale *= zoomFactor
			appState.FitToWindow = false
			appState.Window.Invalidate()
			continue
		}

		// Handle middle mouse panning
		if pe.Kind == pointer.Press && pe.Buttons&pointer.ButtonTertiary != 0 {
			appState.IsPanning = true
			appState.LastPanPosition = pe.Position
			continue
		}
		if appState.IsPanning {
			if pe.Kind == pointer.Drag {
				appState.ViewOffsetX += pe.Position.X - appState.LastPanPosition.X
				appState.ViewOffsetY += pe.Position.Y - appState.LastPanPosition.Y
				appState.LastPanPosition = pe.Position
				appState.FitToWindow = false
				appState.Window.Invalidate()
			} else if pe.Kind == pointer.Release {
				appState.IsPanning = false
			}
			continue
		}

		// Handle simple move and update the hover preview
		if pe.Kind == pointer.Move || pe.Kind == pointer.Drag {
			if inImageBounds {
				labelMap := appState.SuperpixelImplementation.GetResult().LabelMap
				appState.HoverPathSpec = rendering.CalculateSelectionOutlineFromIndex(*labelMap, labelMap.Gray16At(int(imageX), int(imageY)).Y)
			} else {
				appState.HoverPathSpec = rendering.GetEmptySelectionOutline()
			}
		}

		//  Handle primary/secondary click for de-/selection

		if inImageBounds {
			// Sample the current superpixel label at the pointer position
			labelMap := appState.SuperpixelImplementation.GetResult().LabelMap
			label := labelMap.Gray16At(int(imageX), int(imageY)).Y

			// Toggle selection based on primary/secondary button and update the selection outline if it changed
			selectionChanged := false
			if pe.Buttons&pointer.ButtonPrimary != 0 && !appState.SelectionIndex[label] {
				if pe.Modifiers&key.ModShift != 0 {
					// If shift is held, perform a flood fill selection
					appState.SelectionIndex = superpixel_interaction.SuperpixelFloodFillSelection(labelMap, int(imageX), int(imageY), appState.SelectionIndex)
				} else {
					// Perform a normal selection of a single superpixel
					appState.SelectionIndex[label] = true
				}
				selectionChanged = true
			} else if pe.Buttons&pointer.ButtonSecondary != 0 && appState.SelectionIndex[label] {
				appState.SelectionIndex[label] = false
				selectionChanged = true
			}

			if selectionChanged {
				labelMap := appState.SuperpixelImplementation.GetResult().LabelMap
				appState.SelectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(*labelMap, appState.SelectionIndex)
				appState.Window.Invalidate()
			}
		}
	}
}

// Next/Prev handling, both for buttons and keyboard shortcuts

func HandleUiButtonClick(appState *app_config.AppState, gtx layout.Context, button *widget.Clickable) {

	if !button.Clicked(gtx) {
		return
	}

	if button == &appState.PrevUnlabeledButton {
		appState.LoadPreviousUnlabeledImage()
		return
	}

	if button == &appState.PrevButton {
		appState.LoadPreviousImage()
		return
	}

	if button == &appState.NextUnlabeledButton {
		appState.LoadNextUnlabeledImage()
		return
	}

	if button == &appState.NextButton {
		appState.LoadNextImage()
		return
	}

	if button == &appState.OpenProjectButton {
		triggerOpenProject(appState)
		return
	}

	if button == &appState.NewProjectButton {
		triggerNewProject(appState)
		return
	}

}

func HandleKeyboard(appState *app_config.AppState, gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &appState.KeyboardTag},
			key.Filter{Focus: &appState.KeyboardTag, Name: "F"},
			key.Filter{Focus: &appState.KeyboardTag, Name: "C"},
			key.Filter{Focus: &appState.KeyboardTag, Name: "D"},
			key.Filter{Focus: &appState.KeyboardTag, Name: key.NameSpace, Optional: key.ModShift},
		)
		if !ok {
			break
		}
		ke, ok := ev.(key.Event)
		if !ok {
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
			for i := range appState.SelectionIndex {
				appState.SelectionIndex[i] = false
			}
			appState.SelectionPathSpec = rendering.GetEmptySelectionOutline()
			appState.Window.Invalidate()
			continue
		}

		// [D]isable preview (while held)
		if ke.Name == "D" {
			if ke.State == key.Press {
				appState.RenderSelectionOutlines = false
				appState.Window.Invalidate()
			} else if ke.State == key.Release {
				appState.RenderSelectionOutlines = true
				appState.Window.Invalidate()
			}
			continue
		}

		// Tab: next image, Shift+Tab: previous image
		if ke.State == key.Press && ke.Name == key.NameSpace {
			if ke.Modifiers.Contain(key.ModShift) {
				appState.LoadPreviousImage()
			} else {
				appState.LoadNextImage()
			}
			continue
		}
	}
}
