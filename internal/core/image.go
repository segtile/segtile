package core

import (
	"encoding/json"
	"fmt"
	"image"
	"image/draw"
	"image/png"
	"log"
	"os"
	"os/user"
	"path/filepath"
	"time"

	"gioui.org/f32"
	"gioui.org/op/paint"
	"github.com/reiver/go-giotoast"

	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

func (appState *AppState) LoadNextImage() {
	if appState.ActiveImageIndex < len(appState.ImageFiles)-1 {
		appState.loadImageAtIndex(appState.ActiveImageIndex + 1)
	} else {
		appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "This is the last image.", time.Second)
	}
}

func (appState *AppState) LoadPreviousImage() {
	if appState.ActiveImageIndex > 0 {
		appState.loadImageAtIndex(appState.ActiveImageIndex - 1)
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
	for i := appState.ActiveImageIndex + 1; i < len(appState.ImageFiles); i++ {
		maskPath := appState.maskPathForIndex(i)
		if _, err := os.Stat(maskPath); os.IsNotExist(err) {
			appState.loadImageAtIndex(i)
			return
		}
	}
}

// LoadPreviousUnlabeledImage finds the previous image without an existing mask file and loads it.
func (appState *AppState) LoadPreviousUnlabeledImage() {
	for i := appState.ActiveImageIndex - 1; i >= 0; i-- {
		maskPath := appState.maskPathForIndex(i)
		if _, err := os.Stat(maskPath); os.IsNotExist(err) {
			appState.loadImageAtIndex(i)
			return
		}
	}
}

// loadImageAtIndex saves the current mask (if any), then loads the image and its saved selection.
func (appState *AppState) loadImageAtIndex(i int) error {

	if appState.Eval != nil {
		// Evaluation mode: record the timing of the image that is being left
		// and never save anything.
		appState.Eval.EndImage()
	} else if appState.ActiveImage != nil && appState.SelectionMaskDiffersVersionOnDisk {
		// Save the current mask only if the user actually changed the selection.
		appState.SaveCurrentMask(false)
	}

	// Update the current image and start with a fresh, empty selection.
	appState.ActiveImage = ToNRGBA(LoadImage(appState.ImageFiles[i]))
	appState.SelectionMaskRaw = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
	appState.ProcessedSelectionMask = nil
	appState.LastPaintedSpLabel = -1

	var gtMask *superpixel_map.Mask
	var savedSelection *superpixel_map.Mask
	found := false
	if appState.Eval != nil {
		// Evaluation mode: the final mask on disk is ground truth, not saved
		// state. The selection always starts empty and per-image metadata is
		// ignored so all images use the same project-default settings.
		appState.ActiveImageMetadata = nil
		gt, err := appState.loadGroundTruthMask(i)
		if err != nil {
			log.Printf("Failed to load ground truth for %s: %v", filepath.Base(appState.ImageFiles[i]), err)
			appState.Toasts.ShowType(time.Now(), giotoast.TypeError, fmt.Sprintf("No usable ground truth for %s", filepath.Base(appState.ImageFiles[i])), 3*time.Second)
		} else {
			gtMask = gt
		}
	} else {
		metadata, loadedSelection, loadedFound, err := appState.loadSavedImageState(i)
		if err != nil {
			log.Printf("Failed to restore image state for %s: %v", filepath.Base(appState.ImageFiles[i]), err)
			appState.Toasts.ShowType(time.Now(), giotoast.TypeError, fmt.Sprintf("Failed to restore %s", filepath.Base(appState.ImageFiles[i])), 3*time.Second)
		}
		hadActiveImageMetadata := appState.ActiveImageMetadata != nil
		appState.ActiveImageMetadata = nil
		found = loadedFound
		savedSelection = loadedSelection
		if found {
			appState.ActiveImageMetadata = &metadata
			appState.ActiveImageMetadata.Superpixel = metadata.Superpixel.Clone()
			appState.ActiveSettings.Superpixel = metadata.Superpixel.Clone()
			appState.ActiveSettings.SmoothAmount = metadata.SmoothAmount
			appState.ActiveSettings.ShrinkGrowAmount = metadata.ShrinkGrowAmount
			appState.SmoothAmount.Value = metadata.SmoothAmount
			appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(metadata.ShrinkGrowAmount)
		} else if hadActiveImageMetadata && appState.DefaultSettings != nil {
			defaults := appState.DefaultSettings
			appState.ActiveSettings.Superpixel = defaults.Clone().Superpixel
			appState.ActiveSettings.SmoothAmount = defaults.SmoothAmount
			appState.ActiveSettings.ShrinkGrowAmount = defaults.ShrinkGrowAmount
			appState.SmoothAmount.Value = defaults.SmoothAmount
			appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(defaults.ShrinkGrowAmount)
		}
		if found || (hadActiveImageMetadata && appState.DefaultSettings != nil) {
			appState.RebuildEditors()
		}
	}
	appState.SelectionMaskDiffersVersionOnDisk = false

	if err := appState.RecalculateSuperpixels(false); err != nil {
		return err
	}
	if found {
		if savedSelection.Bounds() != appState.ActiveImage.Bounds() {
			err := fmt.Errorf("saved selection dimensions %v do not match image dimensions %v", savedSelection.Bounds(), appState.ActiveImage.Bounds())
			log.Printf("Failed to restore image state for %s: %v", filepath.Base(appState.ImageFiles[i]), err)
			appState.Toasts.ShowType(time.Now(), giotoast.TypeError, fmt.Sprintf("Failed to restore %s", filepath.Base(appState.ImageFiles[i])), 3*time.Second)
		} else {
			appState.SelectionMaskRaw = savedSelection
			appState.ReprocessSelection()
		}
	}

	// Rebuild image paint op.
	appState.ImgBaseOp = paint.NewImageOp(appState.ActiveImage)
	appState.ImgBaseOp.Filter = paint.FilterNearest

	// Reset view and hover.
	appState.FitToWindow = true
	appState.HoverPathSpec = GetEmptySelectionOutline()

	appState.SyncEditors()

	appState.ActiveImageIndex = i
	if appState.Window != nil {
		appState.Window.Invalidate()
	}

	if appState.Eval != nil {
		appState.Eval.BeginImage(i, appState.ImageFiles[i], appState.ActiveImage, gtMask)
		appState.SetManualFixMode(appState.EvalManualOnly())
	}

	return nil

}

// loadGroundTruthMask loads the final mask of image i from the project output
// directory for use as ground truth in evaluation mode.
func (appState *AppState) loadGroundTruthMask(i int) (*superpixel_map.Mask, error) {
	maskPath := appState.maskPathForIndex(i)
	if _, err := os.Stat(maskPath); err != nil {
		return nil, err
	}
	mask := superpixel_map.NewMaskFromImage(LoadImage(maskPath))
	if mask.Bounds() != appState.ActiveImage.Bounds() {
		return nil, fmt.Errorf("ground truth dimensions %v do not match image dimensions %v", mask.Bounds(), appState.ActiveImage.Bounds())
	}
	return mask, nil
}

// SaveCurrentMask writes the processed mask to the project root and the editable state to .meta.
func (appState *AppState) SaveCurrentMask(evenIfEmpty bool) {

	// Evaluation mode never saves anything; the masks on disk are the ground truth.
	if appState.Eval != nil {
		return
	}

	appState.ReprocessSelection()

	if appState.ActiveSp.SpCount != 0 && (appState.SelectionMade() || appState.SelectionMaskDiffersVersionOnDisk || evenIfEmpty) {

		// Safety fallback in case map is somehow nil
		if appState.ProcessedSelectionMask == nil {
			appState.ProcessedSelectionMask = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
		}

		// Determine output paths for the processed mask, selection mask, and metadata.
		outPath := appState.maskPathForIndex(appState.ActiveImageIndex)
		selectionPath := appState.selectionMaskPathForIndex(appState.ActiveImageIndex)
		metadataPath := appState.imageMetadataPathForIndex(appState.ActiveImageIndex)

		// Get the current user for annotator metadata.
		userName := "unknown"
		currentUser, err := user.Current()
		if err != nil {
			// If we cannot determine the current user, report an error and stick with "unknown" as the annotator.
			appState.reportMaskSaveError(metadataPath, fmt.Errorf("could not determine annotator: %w", err))
		} else {
			userName = currentUser.Username
		}

		// Ensure the directories for the output files exist.
		if err := os.MkdirAll(filepath.Dir(outPath), 0750); err != nil {
			appState.reportMaskSaveError(outPath, err)
			return
		}
		if err := os.MkdirAll(filepath.Dir(selectionPath), 0750); err != nil {
			appState.reportMaskSaveError(selectionPath, err)
			return
		}
		if err := SavePngImage(outPath, appState.ProcessedSelectionMask.ToImageGray16()); err != nil {
			appState.reportMaskSaveError(outPath, err)
			return
		}

		// Save the selection mask to its designated path.
		if err := SavePngImage(selectionPath, appState.SelectionMaskRaw.ToImageGray16()); err != nil {
			appState.reportMaskSaveError(selectionPath, err)
			return
		}

		// Save the metadata to its designated path.
		metadata := &ImageMetadata{
			Annotator:        userName,
			Superpixel:       appState.ActiveSettings.Superpixel,
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

		// Update the application state to reflect the newly saved metadata and masks.
		appState.ActiveImageMetadata = metadata
		appState.ActiveImageMetadata.Superpixel = metadata.Superpixel.Clone()
		appState.SelectionMaskDiffersVersionOnDisk = false

		appState.ReprocessSelectionOutline()
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
		return appState.importLegacyMask(i)
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
	if err := metadata.Superpixel.normalize(); err != nil {
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

// importLegacyMask handles images that have a final mask in the output directory but no
// metadata in .meta: the mask is adopted pixel-identically as the unprocessed selection
// and copied to .meta, with Grow/Shrink/Smooth set to 0 so it is reproduced as-is.
// The current superpixel settings are kept.
func (appState *AppState) importLegacyMask(i int) (ImageMetadata, *superpixel_map.Mask, bool, error) {
	maskPath := appState.maskPathForIndex(i)
	if _, err := os.Stat(maskPath); err != nil {
		if os.IsNotExist(err) {
			return ImageMetadata{}, nil, false, nil
		}
		return ImageMetadata{}, nil, false, err
	}

	maskFile, err := os.Open(maskPath)
	if err != nil {
		return ImageMetadata{}, nil, false, err
	}
	defer maskFile.Close()
	maskImage, _, err := image.Decode(maskFile)
	if err != nil {
		return ImageMetadata{}, nil, false, fmt.Errorf("could not decode %s: %w", filepath.Base(maskPath), err)
	}
	selection := superpixel_map.NewMaskFromImage(maskImage)

	metadata := ImageMetadata{
		Annotator:  "unknown",
		Superpixel: appState.ActiveSettings.Superpixel.Clone(),
	}

	selectionPath := appState.selectionMaskPathForIndex(i)
	metadataPath := appState.imageMetadataPathForIndex(i)
	if err := os.MkdirAll(filepath.Dir(selectionPath), 0750); err != nil {
		return ImageMetadata{}, nil, false, err
	}
	if err := SavePngImage(selectionPath, selection.ToImageGray16()); err != nil {
		return ImageMetadata{}, nil, false, err
	}
	data, err := json.MarshalIndent(metadata, "", "  ")
	if err != nil {
		return ImageMetadata{}, nil, false, err
	}
	if err := os.WriteFile(metadataPath, data, 0600); err != nil {
		return ImageMetadata{}, nil, false, err
	}

	return metadata, selection, true, nil
}

func (appState *AppState) RevertToSavedMask() error {
	if appState.ActiveImage == nil || appState.ActiveSettings == nil || appState.ActiveImageIndex < 0 || appState.ActiveImageIndex >= len(appState.ImageFiles) {
		return fmt.Errorf("no project image is currently loaded")
	}

	metadata, savedSelection, found, err := appState.loadSavedImageState(appState.ActiveImageIndex)
	if err != nil {
		return err
	}
	if !found {
		return fmt.Errorf("no saved mask state exists for this image")
	}
	if savedSelection.Bounds() != appState.ActiveImage.Bounds() {
		return fmt.Errorf("saved selection dimensions %v do not match image dimensions %v", savedSelection.Bounds(), appState.ActiveImage.Bounds())
	}

	appState.ActiveSettings.Superpixel = metadata.Superpixel.Clone()
	appState.ActiveSettings.SmoothAmount = metadata.SmoothAmount
	appState.ActiveSettings.ShrinkGrowAmount = metadata.ShrinkGrowAmount
	appState.SmoothAmount.Value = metadata.SmoothAmount
	appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(metadata.ShrinkGrowAmount)
	appState.ActiveImageMetadata = &metadata
	appState.ActiveImageMetadata.Superpixel = metadata.Superpixel.Clone()
	appState.RebuildEditors()

	appState.SelectionMaskRaw = superpixel_map.NewMask(appState.ActiveImage.Bounds().Dx(), appState.ActiveImage.Bounds().Dy())
	if err := appState.RecalculateSuperpixels(false); err != nil {
		return err
	}
	appState.SelectionMaskRaw = savedSelection
	appState.ReprocessSelection()
	appState.SelectionMaskDiffersVersionOnDisk = false
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
	return filepath.Join(projectOutputDirectory(appState.ActiveSettings), imageMaskBaseName(appState.ImageFiles[i]))
}

func (appState *AppState) selectionMaskPathForIndex(i int) string {
	return filepath.Join(projectOutputDirectory(appState.ActiveSettings), ProjectMetadataDirectoryName, imageMaskBaseName(appState.ImageFiles[i]))
}

func (appState *AppState) imageMetadataPathForIndex(i int) string {
	name := imageMaskBaseName(appState.ImageFiles[i])
	return filepath.Join(projectOutputDirectory(appState.ActiveSettings), ProjectMetadataDirectoryName, name[:len(name)-len(filepath.Ext(name))]+".json")
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

func LoadImage(path string) image.Image {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	return src
}

func ToNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func ToGray16(src image.Image) *image.Gray16 {
	if g, ok := src.(*image.Gray16); ok {
		return g
	}
	b := src.Bounds()
	dst := image.NewGray16(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func SavePngImage(path string, img image.Image) error {
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		return err
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		return fmt.Errorf("Failed to save image: %v", err)
	}
	return nil
}

func CopyImage(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}
