package app_config

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"time"

	"gioui.org/f32"
	"gioui.org/op/paint"
	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/rendering"

	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

func (appState *AppState) LoadNextImage() {
	if appState.ImageFilesIndex < len(appState.ImageFiles)-1 {
		appState.loadImageAtIndex(appState.ImageFilesIndex + 1)
	} else {
		appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "This is the last image.", time.Second)
	}
}

func (appState *AppState) LoadPreviousImage() {
	if appState.ImageFilesIndex > 0 {
		appState.loadImageAtIndex(appState.ImageFilesIndex - 1)
	} else {
		appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "This is the first image.", time.Second)
	}
}

func (appState *AppState) LoadFirstImage() {
	if len(appState.ImageFiles) > 0 {
		appState.loadImageAtIndex(0)
	}
}

func (appState *AppState) LoadLastImage() {
	if len(appState.ImageFiles) > 0 {
		appState.loadImageAtIndex(len(appState.ImageFiles) - 1)
	}
}

// LoadNextUnlabeledImage finds the next image without an existing mask file and loads it.
func (appState *AppState) LoadNextUnlabeledImage() {
	for i := appState.ImageFilesIndex + 1; i < len(appState.ImageFiles); i++ {
		maskPath := appState.maskPathForIndex(i)
		if _, err := os.Stat(maskPath); os.IsNotExist(err) {
			appState.loadImageAtIndex(i)
			return
		}
	}
}

// LoadPreviousUnlabeledImage finds the previous image without an existing mask file and loads it.
func (appState *AppState) LoadPreviousUnlabeledImage() {
	for i := appState.ImageFilesIndex - 1; i >= 0; i-- {
		maskPath := appState.maskPathForIndex(i)
		if _, err := os.Stat(maskPath); os.IsNotExist(err) {
			appState.loadImageAtIndex(i)
			return
		}
	}
}

// loadImageAtIndex saves the current mask (if any), loads the image at index i,
// runs the superpixel algorithm, and restores any previously saved mask.
func (appState *AppState) loadImageAtIndex(i int) error {

	// Save the current mask, if a selection is made
	if appState.ActiveImage != nil && appState.SelectionMade() {
		appState.SaveCurrentMask(false, false)
	}

	// Update the current image, start with a fresh, empty selection and run the superpixel algorithm.
	appState.ActiveImage = io_utils.ToNRGBA(io_utils.LoadImage(appState.ImageFiles[i]))
	appState.SelectionMask = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
	err := appState.RecalculateSuperpixels(false)
	if err != nil {
		return err
	}

	// Reset the outline for showing the existing mask
	appState.ActiveImageExistingMask = nil
	appState.ExistingMaskPathSpec = rendering.GetEmptySelectionOutline()
	appState.ProcessedSelectionMask = nil
	appState.AwaitingMaskDecision = false
	appState.LastPaintedSpLabel = -1

	maskPath := appState.maskPathForIndex(i)
	if _, err := os.Stat(maskPath); err == nil {
		loadedMask := superpixel_map.NewMaskFromImage(io_utils.LoadImage(maskPath))
		if !loadedMask.IsEmpty() {
			appState.ActiveImageExistingMask = loadedMask
			appState.ExistingMaskPathSpec = rendering.CalculateOutline(loadedMask, 0)
			appState.AwaitingMaskDecision = true
		}
	}

	// Rebuild image paint op.
	appState.ImgBaseOp = paint.NewImageOp(appState.ActiveImage)
	appState.ImgBaseOp.Filter = paint.FilterNearest

	// Reset view and hover.
	appState.FitToWindow = true
	appState.HoverPathSpec = rendering.GetEmptySelectionOutline()

	appState.SyncEditors()

	appState.ImageFilesIndex = i
	appState.Window.Invalidate()

	return nil

}

// SaveCurrentMask writes the current selection (after Shrink/Grow + Smooth) as a Gray16 PNG to the output directory.
func (appState *AppState) SaveCurrentMask(evenIfEmpty bool, recalculateOutline bool) {

	// Apply any pending brush strokes so the saved mask reflects the latest input.
	appState.FlushPendingManualFixes()

	if appState.ActiveSp.SpCount != 0 && (appState.SelectionMade() || evenIfEmpty) {

		// Ensure that a processed selection mask is available
		appState.ReprocessSelection()

		// Save as image
		outPath := appState.maskPathForIndex(appState.ImageFilesIndex)

		mask := appState.ProcessedSelectionMask
		if mask == nil {
			mask = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
		}

		err := io_utils.SavePngImage(outPath, mask.ToImageGray16())

		if err != nil {
			log.Printf("Failed to save mask to %s: %v", outPath, err)
			appState.Toasts.ShowType(time.Now(), giotoast.TypeError, fmt.Sprintf("Failed to save %s", filepath.Base(outPath)), 2*time.Second)
			return
		} else {

			if recalculateOutline {
				// The saved mask is now the on-disk state too.
				appState.ExistingMaskPathSpec = rendering.CalculateOutline(mask, 0)
			}

			// Add toast
			appState.Toasts.ShowType(time.Now(), giotoast.TypeSuccess, fmt.Sprintf("Saved %s", filepath.Base(outPath)), 500*time.Millisecond)
		}
	}

}

// maskPathForIndex returns the output PNG path for image at index i.
// The base name of the source file is kept but the extension is replaced with .png.
func (appState *AppState) maskPathForIndex(i int) string {
	base := filepath.Base(appState.ImageFiles[i])
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + ".png"
	return filepath.Join(filepath.Dir(appState.Settings.SettingsFile), name)
}

// FitImageToWindow updates the scale and offset to fit the active image within the given canvas size.
func (appState *AppState) FitImageToWindow(canvasSize image.Point) {

	imageWidth := float32(appState.ActiveImage.Bounds().Dx())
	imageHeight := float32(appState.ActiveImage.Bounds().Dy())
	canvasWidth := float32(canvasSize.X)
	canvasHeight := float32(canvasSize.Y)

	// Fit image to window on first render or when requested
	scaleX := canvasWidth / imageWidth
	scaleY := canvasHeight / imageHeight
	appState.ViewScale = scaleX
	if scaleY < appState.ViewScale {
		appState.ViewScale = scaleY
	}
	appState.ViewOffset = f32.Pt((canvasWidth-imageWidth*appState.ViewScale)/2, (canvasHeight-imageHeight*appState.ViewScale)/2)

}
