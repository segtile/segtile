package app_config

import (
	"image"
	"time"

	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func (appState *AppState) SelectionMade() bool {
	return appState.SelectionMask != nil && superpixel_utils.HasNonZeroPixels(appState.SelectionMask)
}

func (appState *AppState) ClearSelection(showToast bool) {
	if appState.SelectionMask == nil {
		return
	}
	superpixel_utils.ResetImage(appState.SelectionMask)
	appState.SelectionMaskDirty = true
	appState.ReprocessSelection()
	appState.Window.Invalidate()
	if showToast {
		appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "Cleared selection.", 1*time.Second)
	}
}

// SetSpSelected selects or deselects a single superpixel by painting all of its pixels
// into the selection mask. Returns true if any pixel was changed.
func (appState *AppState) SetSpSelected(label uint16, selected bool) bool {
	changed := appState.setSpLabelsSelected([]uint16{label}, selected)
	if changed {
		appState.SelectionMaskDirty = true
	}
	return changed
}

// setSpLabelsSelected paints every pixel of the given superpixel labels in the selection mask,
// overwriting any manual brush strokes inside them. The caller is responsible for reprocessing
// the selection and invalidating the window.
func (appState *AppState) setSpLabelsSelected(labels []uint16, selected bool) bool {

	if appState.ActiveSp.LabelMap == nil || appState.SelectionMask == nil || len(labels) == 0 {
		return false
	}

	lookup := make([]bool, appState.ActiveSp.SpCount)
	for _, label := range labels {
		if int(label) < len(lookup) {
			lookup[label] = true
		}
	}

	// Label map and selection mask share the same bounds (both derive from the active image),
	// so a single direct pass over the Pix buffers works for both reading and writing.
	bounds := appState.SelectionMask.Bounds()
	changed := false

	// Only mark the mask dirty (and report a change) when a pixel value actually
	// flips. This keeps a held pointer drag inside an already-selected superpixel
	// from retriggering the expensive outline rebuild on every Drag event.
	target := uint16(65535)
	if !selected {
		target = uint16(0)
	}
	//maskPix := appState.SelectionMask.Pix
	//maskStride := appState.SelectionMask.Stride
	labelMap := appState.ActiveSp.LabelMap
	for x := bounds.Min.X; x < bounds.Max.X; x++ {
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			if lookup[superpixel_utils.UnsafeGray16At(labelMap, x, y)] && superpixel_utils.UnsafeGray16At(appState.SelectionMask, x, y) != target {
				superpixel_utils.UnsafeSetGray16At(appState.SelectionMask, x, y, target)
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

	if appState.ManualFixMode {
		x := int(appState.LastHoverImagePosition.X)
		y := int(appState.LastHoverImagePosition.Y)
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

	label := appState.ActiveSp.LabelMap.Gray16At(int(appState.LastHoverImagePosition.X), int(appState.LastHoverImagePosition.Y)).Y
	if appState.HoverCached && !appState.HoverCacheManual && appState.HoverCacheLabel == label {
		return
	}
	appState.HoverPathSpec = rendering.CalculateOutline(appState.ActiveSp.LabelMap, label)
	appState.HoverCached = true
	appState.HoverCacheManual = false
	appState.HoverCacheLabel = label
}
