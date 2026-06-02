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

func HandleEditors(appState *app_config.AppState, gtx layout.Context) {

	recalculate := false
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
				appState.Settings.Superpixel.Parameters[string(p)] = v
			}
			if err := app_config.Save(appState.Settings, appState.OutputDir); err != nil {
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
				Target:  &appState.canvasTag,
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
		if appState.currentImage == nil {
			continue
		}

		// Always set the keyboard focus to the canvas when hovering on it, so that key commands work without requiring an extra click
		gtx.Execute(key.FocusCmd{Tag: &appState.keyboardTag})

		// From the raw position of the pointer, calculate the position in image space
		imageX := (pe.Position.X - appState.viewOffsetX) / appState.viewScale
		imageY := (pe.Position.Y - appState.viewOffsetY) / appState.viewScale

		// Check if the position is within the bounds of the image
		inImageBounds := imageX >= 0 && imageY >= 0 && int(imageX) < appState.currentImage.Bounds().Dx() && int(imageY) < appState.currentImage.Bounds().Dy()

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
			appState.viewOffsetX = pe.Position.X - (pe.Position.X-appState.viewOffsetX)*zoomFactor
			appState.viewOffsetY = pe.Position.Y - (pe.Position.Y-appState.viewOffsetY)*zoomFactor
			appState.viewScale *= zoomFactor
			appState.fitToWindow = false
			appState.window.Invalidate()
			continue
		}

		// Handle middle mouse panning
		if pe.Kind == pointer.Press && pe.Buttons&pointer.ButtonTertiary != 0 {
			appState.isPanning = true
			appState.lastPanPosition = pe.Position
			continue
		}
		if appState.isPanning {
			if pe.Kind == pointer.Drag {
				appState.viewOffsetX += pe.Position.X - appState.lastPanPosition.X
				appState.viewOffsetY += pe.Position.Y - appState.lastPanPosition.Y
				appState.lastPanPosition = pe.Position
				appState.fitToWindow = false
				appState.window.Invalidate()
			} else if pe.Kind == pointer.Release {
				appState.isPanning = false
			}
			continue
		}

		// Handle simple move and update the hover preview
		if pe.Kind == pointer.Move || pe.Kind == pointer.Drag {
			if inImageBounds {
				labelMap := appState.superpixelImplementation.GetResult().LabelMap
				appState.hoverPathSpec = rendering.CalculateSelectionOutlineFromIndex(*labelMap, labelMap.Gray16At(int(imageX), int(imageY)).Y)
			} else {
				appState.hoverPathSpec = rendering.GetEmptySelectionOutline()
			}
		}

		//  Handle primary/secondary click for de-/selection

		if inImageBounds {
			// Sample the current superpixel label at the pointer position
			labelMap := appState.superpixelImplementation.GetResult().LabelMap
			label := labelMap.Gray16At(int(imageX), int(imageY)).Y

			// Toggle selection based on primary/secondary button and update the selection outline if it changed
			selectionChanged := false
			if pe.Buttons&pointer.ButtonPrimary != 0 && !appState.selectionIndex[label] {
				if pe.Modifiers&key.ModShift != 0 {
					// If shift is held, perform a flood fill selection
					appState.selectionIndex = superpixel_interaction.SuperpixelFloodFillSelection(labelMap, int(imageX), int(imageY), appState.selectionIndex)
				} else {
					// Perform a normal selection of a single superpixel
					appState.selectionIndex[label] = true
				}
				selectionChanged = true
			} else if pe.Buttons&pointer.ButtonSecondary != 0 && appState.selectionIndex[label] {
				appState.selectionIndex[label] = false
				selectionChanged = true
			}

			if selectionChanged {
				labelMap := appState.superpixelImplementation.GetResult().LabelMap
				appState.selectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(*labelMap, appState.selectionIndex)
				appState.window.Invalidate()
			}
		}
	}
}

func HandleKeyboard(appState *app_config.AppState, gtx layout.Context) {
	for {
		ev, ok := gtx.Event(
			key.FocusFilter{Target: &appState.keyboardTag},
			key.Filter{Focus: &appState.keyboardTag, Name: "F"},
			key.Filter{Focus: &appState.keyboardTag, Name: "C"},
			key.Filter{Focus: &appState.keyboardTag, Name: "D"},
			key.Filter{Focus: &appState.keyboardTag, Name: key.NameTab, Optional: key.ModShift},
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
			appState.fitToWindow = true
			appState.window.Invalidate()
		}

		// [C]lear selection
		if ke.State == key.Press && ke.Name == "C" {
			for i := range appState.selectionIndex {
				appState.selectionIndex[i] = false
			}
			appState.selectionPathSpec = rendering.GetEmptySelectionOutline()
			appState.window.Invalidate()
		}

		// [D]isable preview (while held)
		if ke.Name == "D" {
			if ke.State == key.Press {
				appState.renderSelectionOutlines = false
				appState.window.Invalidate()
			} else if ke.State == key.Release {
				appState.renderSelectionOutlines = true
				appState.window.Invalidate()
			}
		}

		// Tab → next image, Shift+Tab → previous image
		if ke.State == key.Press && ke.Name == key.NameTab {
			if ke.Modifiers.Contain(key.ModShift) {
				if appState.currentIndex > 0 {
					appState.loadImageAtIndex(appState.currentIndex - 1)
				}
			} else {
				if appState.currentIndex < len(appState.imageFiles)-1 {
					appState.loadImageAtIndex(appState.currentIndex + 1)
				}
			}
		}
	}
}
