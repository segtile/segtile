package ui

import (
	"math"
	"time"

	"gioui.org/f32"
	"gioui.org/io/key"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"github.com/reiver/go-giotoast"
	"github.com/segtile/segtile/internal/core"
)

func HandlePointer(appState *core.AppState, gtx layout.Context) {

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

		// No image loaded, skip all image-space pointer logic.
		if appState.ActiveImage == nil {
			continue
		}

		if pe.Kind == pointer.Leave || pe.Kind == pointer.Cancel {
			appState.InImageBounds = false
			appState.HoverPathSpec = core.GetEmptySelectionOutline()
			appState.HoverCached = false
			appState.StrokeAnchorValid = false
			appState.LastPaintedSpLabel = -1
			appState.Window.Invalidate()
			continue
		}

		// Always set the keyboard focus to the canvas when hovering on it, so that key commands work without requiring an extra click
		gtx.Execute(key.FocusCmd{Tag: &appState.KeyboardTag})

		// From the raw position of the pointer, calculate the position in image space
		appState.LastHoverImagePosition.X = (pe.Position.X - appState.ViewOffset.X) / appState.ViewScale
		appState.LastHoverImagePosition.Y = (pe.Position.Y - appState.ViewOffset.Y) / appState.ViewScale

		// Check if the position is within the bounds of the image
		appState.InImageBounds = appState.ActiveSp.LabelMap != nil &&
			appState.LastHoverImagePosition.X >= 0 && appState.LastHoverImagePosition.Y >= 0 &&
			appState.LastHoverImagePosition.X < float32(appState.ActiveSp.LabelMap.Width) &&
			appState.LastHoverImagePosition.Y < float32(appState.ActiveSp.LabelMap.Height)

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
				if pe.Buttons&(pointer.ButtonPrimary|pointer.ButtonSecondary) != 0 {
					add := pe.Buttons&pointer.ButtonPrimary != 0
					bounds := appState.ActiveImage.Bounds()
					forEachStrokePoint(strokeFrom(appState, pe), appState.LastHoverImagePosition, bounds.Dx(), bounds.Dy(), func(x, y int) {
						appState.ApplyManualFixAt(x, y, add)
					})
					appState.StrokeAnchor = appState.LastHoverImagePosition
					appState.StrokeAnchorValid = true
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
			appState.ViewOffset.X = pe.Position.X - (pe.Position.X-appState.ViewOffset.X)*zoomFactor
			appState.ViewOffset.Y = pe.Position.Y - (pe.Position.Y-appState.ViewOffset.Y)*zoomFactor
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
				appState.ViewOffset.X += pe.Position.X - appState.LastPanPosition.X
				appState.ViewOffset.Y += pe.Position.Y - appState.LastPanPosition.Y
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
			if appState.EnableAutoFloodFill.Value {
				changed := appState.AutoFloodFillSelection()
				if changed {
					appState.Toasts.ShowType(gtx.Now, giotoast.TypeInfo, "Auto-flood-fill has filled the selection.", 1500*time.Millisecond)
				}
			}
			if appState.SelectionMaskDirty {
				appState.ReprocessSelection()
			}
			// A fresh press must re-evaluate the superpixel under the cursor.
			appState.LastPaintedSpLabel = -1
			appState.StrokeAnchorValid = false
			appState.Window.Invalidate()
			continue
		}

		//  Handle primary/secondary click for de-/selection

		if appState.InImageBounds {
			bounds := appState.ActiveImage.Bounds()

			// Toggle selection based on primary/secondary button and update the selection outline if it changed
			inputMade := false
			buttonPressed := false
			if pe.Buttons&pointer.ButtonPrimary != 0 {
				buttonPressed = true
				if pe.Modifiers.Contain(key.ModShift) {
					// If shift is held, perform a flood fill selection
					inputMade = appState.FloodFillSelectFromPoint(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y))
				} else {
					// Select every superpixel along the stroke. When the cursor jumped
					// since the last frame (slow hardware), bridge the gap with
					// interpolated in-between points so no superpixels are skipped.
					forEachStrokePoint(strokeFrom(appState, pe), appState.LastHoverImagePosition, bounds.Dx(), bounds.Dy(), func(x, y int) {
						// Sample the current superpixel label at this stroke point
						label := appState.ActiveSp.LabelMap.ValueAt(x, y)
						// Only paint when the cursor moved into a different superpixel;
						// a held drag inside the same one needs no full-image rescan.
						if int(label) != appState.LastPaintedSpLabel {
							if appState.SetSpSelected(label, true) {
								inputMade = true
							}
							appState.LastPaintedSpLabel = int(label)
						}
					})
				}
			} else if pe.Buttons&pointer.ButtonSecondary != 0 {
				buttonPressed = true
				// Same stroke interpolation for deselection.
				forEachStrokePoint(strokeFrom(appState, pe), appState.LastHoverImagePosition, bounds.Dx(), bounds.Dy(), func(x, y int) {
					label := appState.ActiveSp.LabelMap.ValueAt(x, y)
					if int(label) != appState.LastPaintedSpLabel {
						if appState.SetSpSelected(label, false) {
							inputMade = true
						}
						appState.LastPaintedSpLabel = int(label)
					}
				})
			}

			// Anchor the stroke at the current position for the next pointer event.
			if buttonPressed {
				appState.StrokeAnchor = appState.LastHoverImagePosition
				appState.StrokeAnchorValid = true
			}

			if inputMade {
				if appState.SelectionMaskDirty {
					appState.ReprocessSelection()
				}
				appState.Window.Invalidate()
			}
		}
	}
}

// strokeFrom returns the image-space point that the current stroke should be
// interpolated from: the position where the stroke last applied paint during a
// drag, or the pointer position itself for a fresh press (a single point).
func strokeFrom(appState *core.AppState, pe pointer.Event) f32.Point {
	if pe.Kind == pointer.Drag && appState.StrokeAnchorValid {
		return appState.StrokeAnchor
	}
	return appState.LastHoverImagePosition
}

// forEachStrokePoint calls fn for every integer image-space point on the line
// from "from" to "to" (excluding "from", including "to"), spaced at most one
// pixel apart.
func forEachStrokePoint(from, to f32.Point, width, height int, fn func(x, y int)) {
	dx := to.X - from.X
	dy := to.Y - from.Y
	steps := int(math.Ceil(math.Sqrt(float64(dx*dx + dy*dy))))
	if steps < 1 {
		steps = 1
	}
	for i := 1; i <= steps; i++ {
		t := float32(i) / float32(steps)
		x := int(from.X + dx*t)
		y := int(from.Y + dy*t)
		if x < 0 || y < 0 || x >= width || y >= height {
			continue
		}
		fn(x, y)
	}
}
