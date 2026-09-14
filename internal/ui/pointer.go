package ui

import (
	"time"

	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/app_config"
	"github.com/struffel/annotation-tool/internal/rendering"
)

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
				Kinds:   pointer.Cancel | pointer.Press | pointer.Release | pointer.Drag | pointer.Scroll | pointer.Move | pointer.Enter | pointer.Leave,
				ScrollX: pointer.ScrollRange{Min: -100, Max: 100},
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

		if pe.Kind == pointer.Leave || pe.Kind == pointer.Cancel {
			appState.InImageBounds = false
			appState.HoverPathSpec = rendering.GetEmptySelectionOutline()
			appState.HoverCached = false
			appState.Window.Invalidate()
			continue
		}

		// Always set the keyboard focus to the canvas when hovering on it, so that key commands work without requiring an extra click
		gtx.Execute(key.FocusCmd{Tag: &appState.KeyboardTag})

		// From the raw position of the pointer, calculate the position in image space
		appState.LastHoverImagePosition.X = (pe.Position.X - appState.ViewOffsetX) / appState.ViewScale
		appState.LastHoverImagePosition.Y = (pe.Position.Y - appState.ViewOffsetY) / appState.ViewScale

		// Check if the position is within the bounds of the image
		appState.InImageBounds = appState.ActiveSp.LabelMap != nil && appState.LastHoverImagePosition.X >= 0 && appState.LastHoverImagePosition.Y >= 0 && int(appState.LastHoverImagePosition.X) < appState.ActiveImage.Bounds().Dx() && int(appState.LastHoverImagePosition.Y) < appState.ActiveImage.Bounds().Dy()

		// Handle manual fix mode
		if appState.ManualFixMode && appState.ActiveImage != nil {

			// Scroll event (change radius)
			if pe.Kind == pointer.Scroll {

				// Determine the new radius of the selection brush
				if pe.Scroll.Y > 0 {
					appState.ManualFixBrushRadius -= 2
				} else if pe.Scroll.Y < 0 {
					appState.ManualFixBrushRadius += 2
				}

				if appState.ManualFixBrushRadius < 1 {
					appState.ManualFixBrushRadius = 1
				}

				appState.RefreshHoverPreview()
				appState.Window.Invalidate()
				continue
			}

			// Move event
			if (pe.Kind == pointer.Move || pe.Kind == pointer.Drag) && appState.ManualFixMode {
				appState.RefreshHoverPreview()
			}

			// Paint event
			if appState.InImageBounds && (pe.Kind == pointer.Press || pe.Kind == pointer.Drag) {
				if pe.Buttons&pointer.ButtonPrimary != 0 {
					appState.ApplyManualFixAt(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y), true)
				} else if pe.Buttons&pointer.ButtonSecondary != 0 {
					appState.ApplyManualFixAt(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y), false)
				}
				continue
			}

			continue
		}

		// Now handle normal selections

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
			appState.RefreshHoverPreview()
		}

		// Whenever the pointer is released, apply any pending auto flood fill and reprocess the selection.
		if pe.Kind == pointer.Release {
			if appState.AutoFloodFillSelection() {
				appState.Toasts.ShowType(gtx.Now, giotoast.TypeInfo, "Auto-flood-fill has filled the selection.", 1500*time.Millisecond)
			}
			appState.ReprocessSelection()
			appState.Window.Invalidate()
			continue
		}

		//  Handle primary/secondary click for de-/selection

		if appState.InImageBounds {
			// Sample the current superpixel label at the pointer position
			label := appState.ActiveSp.LabelMap.Gray16At(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y)).Y

			// Toggle selection based on primary/secondary button and update the selection outline if it changed
			inputMade := false
			if pe.Buttons&pointer.ButtonPrimary != 0 {
				inputMade = true
				if pe.Modifiers.Contain(key.ModShift) {
					// If shift is held, perform a flood fill selection
					appState.FloodFillSelectFromPoint(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y))
				} else {
					// Perform a normal selection of a single superpixel
					appState.SetSpSelected(label, true)

				}
			} else if pe.Buttons&pointer.ButtonSecondary != 0 {
				appState.SetSpSelected(label, false)

				inputMade = true
			}

			if inputMade {
				// A click while a mask decision is pending implicitly means "start from scratch".
				if appState.AwaitingMaskDecision {
					appState.AwaitingMaskDecision = false
					appState.ActiveImageExistingMask = nil
					appState.ExistingMaskPathSpec = rendering.GetEmptySelectionOutline()
					appState.Window.Invalidate()
				}

				appState.ReprocessSelection()
				appState.Window.Invalidate()
			}
		}
	}
}
