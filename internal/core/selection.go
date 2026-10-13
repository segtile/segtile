package core

import (
	"image"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func (appState *AppState) SelectionMade() bool {
	return appState.SelectionMask != nil && !appState.SelectionMask.IsEmpty()
}

func (appState *AppState) ClearSelection(showToast bool) {
	if appState.SelectionMask == nil {
		return
	}
	appState.SelectionMask.Clear()
	appState.SelectionMaskDirty = true
	appState.ReprocessSelection()
	appState.Window.Invalidate()
	if showToast {
		appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "Cleared selection.", 1*time.Second)
	}
}

// SetSpSelected selects or deselects a single superpixel by painting all of its pixels
// into the selection mask. Returns true if any pixel was changed.
func (appState *AppState) SetSpSelected(label uint32, selected bool) bool {

	if appState.ActiveSp.LabelMap == nil || appState.SelectionMask == nil {
		return false
	}

	bounds := appState.ActiveSp.BoundingBoxes[label].Bounds()
	changed := false

	// Only mark the mask dirty (and report a change) when a pixel value actually
	// flips. This keeps a held pointer drag inside an already-selected superpixel
	// from retriggering the expensive outline rebuild on every Drag event.
	target := uint16(65535)
	if !selected {
		target = uint16(0)
	}

	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if appState.ActiveSp.LabelMap.ValueAt(x, y) == label && (appState.SelectionMask.PositiveAt(x, y) != (target != 0)) {
				appState.SelectionMask.SetValueAt(x, y, selected)
				changed = true
			}
		}
	}
	if changed {
		appState.SelectionMaskDirty = true
	}
	return changed
}

// FloodFillSelectFromPoint selects the entire connected region of unselected pixels
// containing the point (x, y). Works directly on the selection mask, so brush strokes
// are respected pixel-accurately. Returns true if any pixel was changed.
func (appState *AppState) FloodFillSelectFromPoint(x, y int) bool {
	if appState.SelectionMask == nil {
		return false
	}
	changed := superpixel_utils.FloodFillMaskFromPoint(appState.SelectionMask, x, y)
	if changed {
		appState.SelectionMaskDirty = true
	}
	return changed
}

// AutoFloodFillSelection selects all pixels that are fully enclosed by the current selection
// (i.e. not connected to the image border via unselected pixels). Works directly on the
// selection mask. Returns true if any pixel was changed.
func (appState *AppState) AutoFloodFillSelection() bool {
	if appState.SelectionMask == nil {
		return false
	}
	changed := superpixel_utils.FillEnclosedMaskRegions(appState.SelectionMask)
	if changed {
		appState.SelectionMaskDirty = true
	}
	return changed
}

func (appState *AppState) SetManualFixMode(enabled bool) {
	if appState.ManualFixMode == enabled {
		return
	}
	appState.ManualFixMode = enabled
	appState.RefreshHoverPreview()
	appState.Window.Invalidate()
}

func (appState *AppState) RefreshHoverPreview() {
	if !appState.InImageBounds || appState.ActiveSp.LabelMap == nil {
		if appState.HoverCached {
			appState.HoverPathSpec = rendering.GetEmptySelectionOutline()
			appState.HoverCached = false
		}
		return
	}

	labelMap := appState.ActiveSp.LabelMap
	hoverX, hoverY := appState.LastHoverImagePosition.X, appState.LastHoverImagePosition.Y
	if !(hoverX >= 0 && hoverY >= 0 && hoverX < float32(labelMap.Width) && hoverY < float32(labelMap.Height)) {
		appState.InImageBounds = false
		if appState.HoverCached {
			appState.HoverPathSpec = rendering.GetEmptySelectionOutline()
			appState.HoverCached = false
		}
		return
	}
	x, y := int(hoverX), int(hoverY)

	if appState.ManualFixMode {
		radius := appState.ManualFixBrushRadius
		if appState.HoverCached && appState.HoverCacheManual && appState.HoverCachePos == image.Pt(x, y) && appState.HoverCacheRadius == radius {
			return
		}
		appState.HoverPathSpec = rendering.CalculateCircleOutline(x, y, radius)
		appState.HoverCached = true
		appState.HoverCacheManual = true
		appState.HoverCachePos = image.Pt(x, y)
		appState.HoverCacheRadius = radius
		return
	}

	label := labelMap.ValueAt(x, y)
	if appState.HoverCached && !appState.HoverCacheManual && appState.HoverCacheLabel == label {
		return
	}
	appState.HoverPathSpec = rendering.CalculateOutline(appState.ActiveSp.LabelMap, label)
	appState.HoverCached = true
	appState.HoverCacheManual = false
	appState.HoverCacheLabel = label
}
