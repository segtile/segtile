package core

import (
	"encoding/json"
	"fmt"
	"image"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"gioui.org/f32"
	"gioui.org/op/paint"
	"github.com/reiver/go-giotoast"
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

// loadImageAtIndex saves the current mask (if any), then loads the image and its saved selection.
func (appState *AppState) loadImageAtIndex(i int) error {

	// Save the current mask, if a selection is made
	if appState.ActiveImage != nil && appState.SelectionMade() {
		appState.SaveCurrentMask(false, false)
	}

	// Update the current image and start with a fresh, empty selection.
	appState.ActiveImage = ToNRGBA(LoadImage(appState.ImageFiles[i]))
	appState.SelectionMask = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
	appState.ProcessedSelectionMask = nil
	appState.LastPaintedSpLabel = -1

	metadata, savedSelection, found, err := appState.loadSavedImageState(i)
	if err != nil {
		log.Printf("Failed to restore image state for %s: %v", filepath.Base(appState.ImageFiles[i]), err)
		appState.Toasts.ShowType(time.Now(), giotoast.TypeError, fmt.Sprintf("Failed to restore %s", filepath.Base(appState.ImageFiles[i])), 3*time.Second)
	}
	hadActiveImageMetadata := appState.ActiveImageHasMetadata
	appState.ActiveImageMetadata = ImageMetadata{}
	if found {
		appState.ActiveImageMetadata = metadata
		appState.ActiveImageMetadata.Superpixel = cloneSuperpixelSettings(metadata.Superpixel)
		appState.Settings.Superpixel = cloneSuperpixelSettings(metadata.Superpixel)
		appState.Settings.SmoothAmount = metadata.SmoothAmount
		appState.Settings.ShrinkGrowAmount = metadata.ShrinkGrowAmount
		appState.SmoothAmount.Value = metadata.SmoothAmount
		appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(metadata.ShrinkGrowAmount)
	} else if hadActiveImageMetadata && appState.ProjectSettingsDefaults != nil {
		defaults := appState.ProjectSettingsDefaults
		appState.Settings.Superpixel = cloneProjectSettings(defaults).Superpixel
		appState.Settings.SmoothAmount = defaults.SmoothAmount
		appState.Settings.ShrinkGrowAmount = defaults.ShrinkGrowAmount
		appState.SmoothAmount.Value = defaults.SmoothAmount
		appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(defaults.ShrinkGrowAmount)
	}
	if found || (hadActiveImageMetadata && appState.ProjectSettingsDefaults != nil) {
		appState.RebuildEditors()
	}
	appState.ActiveImageHasMetadata = found

	if err := appState.RecalculateSuperpixels(false); err != nil {
		return err
	}
	if found {
		if savedSelection.Bounds() != appState.ActiveImage.Bounds() {
			err := fmt.Errorf("saved selection dimensions %v do not match image dimensions %v", savedSelection.Bounds(), appState.ActiveImage.Bounds())
			log.Printf("Failed to restore image state for %s: %v", filepath.Base(appState.ImageFiles[i]), err)
			appState.Toasts.ShowType(time.Now(), giotoast.TypeError, fmt.Sprintf("Failed to restore %s", filepath.Base(appState.ImageFiles[i])), 3*time.Second)
		} else {
			appState.SelectionMask = savedSelection
			appState.ReprocessSelection()
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
	if appState.Window != nil {
		appState.Window.Invalidate()
	}

	return nil

}

// SaveCurrentMask writes the processed mask to the project root and the editable state to .meta.
func (appState *AppState) SaveCurrentMask(evenIfEmpty bool, recalculateOutline bool) {

	// Apply any pending brush strokes so the saved mask reflects the latest input.
	appState.FlushPendingManualFixes()

	if appState.ActiveSp.SpCount != 0 && (appState.SelectionMade() || evenIfEmpty) {

		// Ensure that a processed selection mask is available
		appState.ReprocessSelection()

		mask := appState.ProcessedSelectionMask
		if mask == nil {
			mask = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
		}

		outPath := appState.maskPathForIndex(appState.ImageFilesIndex)
		selectionPath := appState.selectionMaskPathForIndex(appState.ImageFilesIndex)
		metadataPath := appState.imageMetadataPathForIndex(appState.ImageFilesIndex)
		currentUser, err := user.Current()
		if err != nil {
			appState.reportMaskSaveError(metadataPath, fmt.Errorf("could not determine annotator: %w", err))
			return
		}
		if err := os.MkdirAll(filepath.Dir(outPath), 0750); err != nil {
			appState.reportMaskSaveError(outPath, err)
			return
		}
		if err := os.MkdirAll(filepath.Dir(selectionPath), 0750); err != nil {
			appState.reportMaskSaveError(selectionPath, err)
			return
		}
		if err := SavePngImage(outPath, mask.ToImageGray16()); err != nil {
			appState.reportMaskSaveError(outPath, err)
			return
		}
		if err := SavePngImage(selectionPath, appState.SelectionMask.ToImageGray16()); err != nil {
			appState.reportMaskSaveError(selectionPath, err)
			return
		}

		metadata := ImageMetadata{
			Annotator:        currentUser.Username,
			Superpixel:       appState.Settings.Superpixel,
			SmoothAmount:     appState.SmoothAmount.Value,
			ShrinkGrowAmount: SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value),
		}
		data, err := json.MarshalIndent(metadata, "", "  ")
		if err != nil {
			appState.reportMaskSaveError(metadataPath, err)
			return
		}
		if err := os.WriteFile(metadataPath, data, 0600); err != nil {
			appState.reportMaskSaveError(metadataPath, err)
			return
		}
		appState.ActiveImageHasMetadata = true
		appState.ActiveImageMetadata = metadata
		appState.ActiveImageMetadata.Superpixel = cloneSuperpixelSettings(metadata.Superpixel)

		if recalculateOutline {
			appState.SelectionPathSpec = rendering.CalculateOutline(mask, 0)
		}
		appState.Toasts.ShowType(time.Now(), giotoast.TypeSuccess, fmt.Sprintf("Saved %s", filepath.Base(outPath)), 500*time.Millisecond)
	}

}

func (appState *AppState) loadSavedImageState(i int) (ImageMetadata, *superpixel_map.Mask, bool, error) {
	metadataPath := appState.imageMetadataPathForIndex(i)
	selectionPath := appState.selectionMaskPathForIndex(i)

	metadataBytes, metadataErr := os.ReadFile(metadataPath)
	_, selectionErr := os.Stat(selectionPath)
	metadataMissing := os.IsNotExist(metadataErr)
	selectionMissing := os.IsNotExist(selectionErr)
	if metadataMissing && selectionMissing {
		return ImageMetadata{}, nil, false, nil
	}
	if metadataErr != nil {
		return ImageMetadata{}, nil, false, metadataErr
	}
	if selectionErr != nil {
		return ImageMetadata{}, nil, false, selectionErr
	}

	var metadata ImageMetadata
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return ImageMetadata{}, nil, false, fmt.Errorf("could not parse %s: %w", filepath.Base(metadataPath), err)
	}
	if err := metadata.Superpixel.normalizeSuperpixelSettings(); err != nil {
		return ImageMetadata{}, nil, false, fmt.Errorf("invalid superpixel metadata: %w", err)
	}

	selectionFile, err := os.Open(selectionPath)
	if err != nil {
		return ImageMetadata{}, nil, false, err
	}
	defer selectionFile.Close()
	selectionImage, _, err := image.Decode(selectionFile)
	if err != nil {
		return ImageMetadata{}, nil, false, fmt.Errorf("could not decode %s: %w", filepath.Base(selectionPath), err)
	}
	return metadata, superpixel_map.NewMaskFromImage(selectionImage), true, nil
}

func (appState *AppState) RevertToSavedMask() error {
	if appState.ActiveImage == nil || appState.Settings == nil || appState.ImageFilesIndex < 0 || appState.ImageFilesIndex >= len(appState.ImageFiles) {
		return fmt.Errorf("no project image is currently loaded")
	}

	metadata, savedSelection, found, err := appState.loadSavedImageState(appState.ImageFilesIndex)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no saved mask state exists for this image")
	}
	if savedSelection.Bounds() != appState.ActiveImage.Bounds() {
		return fmt.Errorf("saved selection dimensions %v do not match image dimensions %v", savedSelection.Bounds(), appState.ActiveImage.Bounds())
	}

	appState.Settings.Superpixel = cloneSuperpixelSettings(metadata.Superpixel)
	appState.Settings.SmoothAmount = metadata.SmoothAmount
	appState.Settings.ShrinkGrowAmount = metadata.ShrinkGrowAmount
	appState.SmoothAmount.Value = metadata.SmoothAmount
	appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(metadata.ShrinkGrowAmount)
	appState.ActiveImageHasMetadata = true
	appState.ActiveImageMetadata = metadata
	appState.ActiveImageMetadata.Superpixel = cloneSuperpixelSettings(metadata.Superpixel)
	appState.RebuildEditors()

	appState.SelectionMask = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
	if err := appState.RecalculateSuperpixels(false); err != nil {
		return err
	}
	appState.SelectionMask = savedSelection
	appState.ReprocessSelection()
	if appState.Window != nil {
		appState.Window.Invalidate()
	}
	return nil
}

func (appState *AppState) reportMaskSaveError(path string, err error) {
	log.Printf("Failed to save %s: %v", path, err)
	appState.Toasts.ShowType(time.Now(), giotoast.TypeError, fmt.Sprintf("Failed to save %s", filepath.Base(path)), 2*time.Second)
}

// maskPathForIndex returns the output PNG path for image at index i.
// The base name of the source file is kept but the extension is replaced with .png.
func (appState *AppState) maskPathForIndex(i int) string {
	return filepath.Join(projectOutputDirectory(appState.Settings), imageMaskBaseName(appState.ImageFiles[i]))
}

func (appState *AppState) selectionMaskPathForIndex(i int) string {
	return filepath.Join(projectOutputDirectory(appState.Settings), ProjectMetadataDirectoryName, imageMaskBaseName(appState.ImageFiles[i]))
}

func (appState *AppState) imageMetadataPathForIndex(i int) string {
	name := imageMaskBaseName(appState.ImageFiles[i])
	return filepath.Join(projectOutputDirectory(appState.Settings), ProjectMetadataDirectoryName, name[:len(name)-len(filepath.Ext(name))]+".json")
}

func imageMaskBaseName(imagePath string) string {
	base := filepath.Base(imagePath)
	ext := filepath.Ext(base)
	return base[:len(base)-len(ext)] + ".png"
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
