package ui

import (
	"log"
	"strconv"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/app_config"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
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
		appState.RecalculateProcessedSelection()
		appState.Window.Invalidate()
	}
	smoothWasDragging := appState.SmoothAmount.Dragging()
	if appState.SmoothAmount.Update(gtx) {
		appState.Settings.SmoothAmount = appState.SmoothAmount.Value
		appState.RecalculateProcessedSelection()
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

func HandlePointer(appState *app_config.AppState, gtx layout.Context) {

	if appState.UiLocked() {
		return
	}

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
		if appState.ActiveImage == nil {
			continue
		}

		// Always set the keyboard focus to the canvas when hovering on it, so that key commands work without requiring an extra click
		gtx.Execute(key.FocusCmd{Tag: &appState.KeyboardTag})

		// From the raw position of the pointer, calculate the position in image space
		imageX := (pe.Position.X - appState.ViewOffsetX) / appState.ViewScale
		imageY := (pe.Position.Y - appState.ViewOffsetY) / appState.ViewScale

		// Check if the position is within the bounds of the image
		inImageBounds := appState.ActiveSp.LabelMap != nil && imageX >= 0 && imageY >= 0 && int(imageX) < appState.ActiveImage.Bounds().Dx() && int(imageY) < appState.ActiveImage.Bounds().Dy()

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
				appState.HoverPathSpec = rendering.CalculateSelectionOutlineFromIndex(*appState.ActiveSp.LabelMap, appState.ActiveSp.LabelMap.Gray16At(int(imageX), int(imageY)).Y)
			} else {
				appState.HoverPathSpec = rendering.GetEmptySelectionOutline()
			}
		}

		//  Handle primary/secondary click for de-/selection

		if inImageBounds {
			// Sample the current superpixel label at the pointer position
			label := appState.ActiveSp.LabelMap.Gray16At(int(imageX), int(imageY)).Y

			// Toggle selection based on primary/secondary button and update the selection outline if it changed
			selectionChanged := false
			if pe.Buttons&pointer.ButtonPrimary != 0 && !appState.SelectionIndex[label] {
				if pe.Modifiers&key.ModShift != 0 {
					// If shift is held, perform a flood fill selection
					appState.SelectionIndex = superpixel_utils.SuperpixelFloodFillFromPoint(appState.ActiveSp.LabelMap, int(imageX), int(imageY), appState.SelectionIndex)
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
				// A click while a mask decision is pending implicitly means "start from scratch".
				if appState.AwaitingMaskDecision {
					appState.AwaitingMaskDecision = false
					appState.ActiveImageExistingMask = nil
					appState.ExistingMaskPathSpec = rendering.GetEmptySelectionOutline()
					appState.Window.Invalidate()
				}

				// Perform auto flood fill selection to select all superpixels that are enclosed by the current selection
				if appState.EnableAutoFloodFill.Value {
					appState.SelectionIndex = superpixel_utils.SuperpixelAutoFloodFillSelection(appState.ActiveSp.LabelMap, appState.SelectionIndex)
				}

				appState.RecalculateProcessedSelection()
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
	if appState.CurrentlyOptimizing {
		return
	}

	if button == &appState.ImagesSectionButton {
		appState.ImagesSectionOpen = !appState.ImagesSectionOpen
		return
	}

	if button == &appState.SuperpixelSectionButton {
		appState.SuperpixelSectionOpen = !appState.SuperpixelSectionOpen
		return
	}

	if button == &appState.MiscSectionButton {
		appState.MiscSectionOpen = !appState.MiscSectionOpen
		return
	}

	if button == &appState.ChangeSpAlgorithmButton {
		appState.SpAlgorithmMenuOpen = !appState.SpAlgorithmMenuOpen
		return
	}

	if button == &appState.EditExistingMaskButton {
		appState.RecreateExistingMaskFromDisk()
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

	if button == &appState.OptimizeButton {
		triggerOptimization(appState)
		return
	}

	if button == &appState.FirstButton {
		appState.LoadFirstImage()
		return
	}

	if button == &appState.LastButton {
		appState.LoadLastImage()
		return
	}

	if button == &appState.SaveButton {
		if appState.SelectionMade() || showSaveEmptyMaskWarning() {
			appState.SaveCurrentMask(true, true)
		}

		return
	}

	for i := range appState.SpAlgorithmMenuButtons {

		if button != &appState.SpAlgorithmMenuButtons[i] {
			continue
		}

		err := appState.SetSpAlgorithm(appState.SpAlgorithmMenuAlgorithmIds[i])
		if err != nil {
			log.Printf("Failed to change superpixel algorithm: %v", err)
			return
		}
		appState.SpAlgorithmMenuOpen = false

		if appState.Window != nil {
			appState.Window.Invalidate()
		}
		return
	}

}

func HandleKeyboard(appState *app_config.AppState, gtx layout.Context) {
	if appState.CurrentlyOptimizing {
		return
	}
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &appState.KeyboardTag},
			key.Filter{Focus: &appState.KeyboardTag, Name: "F"},
			key.Filter{Focus: &appState.KeyboardTag, Name: "C"},
			key.Filter{Focus: &appState.KeyboardTag, Name: "S"},
			key.Filter{Focus: &appState.KeyboardTag, Name: "D"},
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
				appState.Window.Invalidate()
			} else if ke.State == key.Release {
				appState.RenderSelectionOutlines = true
				appState.Window.Invalidate()
			}
			continue
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
