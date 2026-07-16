package app_config

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"github.com/reiver/go-giotoast"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/widget"
	"github.com/sqweek/dialog"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type AppState struct {

	// Project
	Settings   *ProjectSettings
	ImageFiles []string

	// ActiveResult
	ActiveResult superpixel.SpResult

	// Images
	CurrentImage *image.NRGBA
	ExistingMask *image.Gray16
	CurrentIndex int

	// Selected superpixels
	SelectionIndex []bool

	// Render cache (rebuilt only when data changes, not every frame)
	ImgBaseOp            paint.ImageOp
	SelectionPathSpec    clip.PathSpec
	HoverPathSpec        clip.PathSpec
	ExistingMaskPathSpec clip.PathSpec

	// UI state
	Window                          *app.Window
	Toasts                          giotoast.Queue
	CanvasTag                       bool
	Theme                           *app_theme.AppTheme
	SuperpixelParameterEditors      []widget.Editor
	SuperpixelParameterEditorIds    []string
	SuperpixelParameterEditorTitles []string
	RenderSelectionOutlines         bool
	FirstButton                     widget.Clickable
	LastButton                      widget.Clickable
	PrevUnlabeledButton             widget.Clickable
	PrevButton                      widget.Clickable
	NextUnlabeledButton             widget.Clickable
	NextButton                      widget.Clickable
	OpenProjectButton               widget.Clickable
	NewProjectButton                widget.Clickable
	OptimizeButton                  widget.Clickable
	SaveButton                      widget.Clickable

	ChangeSpAlgorithmButton     widget.Clickable
	SpAlgorithmMenuButtons      []widget.Clickable
	SpAlgorithmMenuAlgorithmIds []string
	SpAlgoritmMenuTitles        []string
	SpAlgorithmMenuOpen         bool

	// Loading state
	CurrentlyLoadingProject bool
	ProjectLoadResult       chan ProjectLoadingResult

	// Hover state
	IsPanning        bool
	LastPanPosition  f32.Point
	KeyboardTag      bool
	InitialFocusDone bool

	// View transform
	ViewScale, ViewOffsetX, ViewOffsetY float32
	FitToWindow                         bool

	// Display settings

}

func NewSuperpixelSettings() SuperpixelSettings {
	defaultParameters, _ := superpixel.GetDefaultParameterValues(defaultSpAlgorithmId)
	return SuperpixelSettings{
		SpAlgorithmId:         defaultSpAlgorithmId,
		SpAlgorithmParameters: defaultParameters,
	}
}

func NewAppState() *AppState {

	appState := &AppState{
		FitToWindow:             true,
		RenderSelectionOutlines: true,
		Theme:                   app_theme.Theme,
		ProjectLoadResult:       make(chan ProjectLoadingResult, 1),
	}

	// Initialize the superpixel algorithm menu. The possible values do not change after compilation.
	for _, id := range superpixel.ListAlgorithms() {

		button := new(widget.Clickable)
		appState.SpAlgorithmMenuButtons = append(appState.SpAlgorithmMenuButtons, *button)
		appState.SpAlgorithmMenuAlgorithmIds = append(appState.SpAlgorithmMenuAlgorithmIds, id)

		title, _, _, err := superpixel.GetAlgorithmInfo(id)
		if err != nil {
			title = fmt.Sprintf("Untitled (%s)", id)
		}
		appState.SpAlgoritmMenuTitles = append(appState.SpAlgoritmMenuTitles, title)
	}

	appState.RebuildEditors()

	return appState
}

// Set up the editors for the current superpixel parameters.
func (appState *AppState) RebuildEditors() {

	// Overwrite existing editor data.
	appState.SuperpixelParameterEditors = nil
	appState.SuperpixelParameterEditorIds = nil
	appState.SuperpixelParameterEditorTitles = nil

	// No editors if there are no settings
	if appState.Settings == nil {
		return
	}

	if err := appState.Settings.Superpixel.normalizeSuperpixelSettings(); err != nil {
		log.Printf("Failed to normalize superpixel settings: %v", err)
		return
	}

	parameterInfo, err := superpixel.ListParameterInfo(appState.Settings.Superpixel.SpAlgorithmId)
	if err != nil {
		log.Printf("Failed to list superpixel parameter info: %v", err)
		return
	}

	// Turn the parameter list into aligned editor data for the UI.
	for _, paramInfo := range parameterInfo {
		paramValue := appState.Settings.Superpixel.SpAlgorithmParameters[paramInfo.Id]
		editor := widget.Editor{SingleLine: true, Submit: true, Filter: "-.0123456789", Alignment: text.End}
		editor.SetText(fmt.Sprintf("%g", paramValue))
		appState.SuperpixelParameterEditors = append(appState.SuperpixelParameterEditors, editor)
		appState.SuperpixelParameterEditorIds = append(appState.SuperpixelParameterEditorIds, paramInfo.Id)
		appState.SuperpixelParameterEditorTitles = append(appState.SuperpixelParameterEditorTitles, paramInfo.Title)
	}
}

// SyncEditors updates each superpixel parameter editor's text to reflect
// the current values in the superpixel implementation.
func (appState *AppState) SyncEditors() error {
	for i, paramId := range appState.SuperpixelParameterEditorIds {
		paramValue, ok := appState.Settings.Superpixel.SpAlgorithmParameters[paramId]
		if !ok {
			return fmt.Errorf("Value for parameter %s not found", paramId)
		}
		appState.SuperpixelParameterEditors[i].SetText(fmt.Sprintf("%g", paramValue))
	}

	return nil
}

func (appState *AppState) SetSpAlgorithm(id string) error {

	// Set the new algorithm id
	if !slices.Contains(superpixel.ListAlgorithms(), id) {
		return fmt.Errorf("Unknown superpixel algorithm: %s", id)
	}
	appState.Settings.Superpixel.SpAlgorithmId = id

	// Load the right set of parameters
	defaultParameters, err := superpixel.GetDefaultParameterValues(id)
	if err != nil {
		return err
	}
	appState.Settings.Superpixel.SpAlgorithmParameters = defaultParameters

	// Rebuild the editors for the new params
	appState.RebuildEditors()

	// Recalculate the superpixels for the current image
	if appState.CurrentImage != nil {
		appState.RecalculateSuperpixels(true, true)
	}

	return nil
}

// ApplyProject configures the AppState for the given project, saves any current mask,
// and loads the first image.
func (appState *AppState) ApplyProject(projectSettings *ProjectSettings) error {

	// Save the old image before loading the new project
	if appState.CurrentImage != nil {
		appState.SaveCurrentMask(false, true)
		appState.CurrentImage = nil
	}

	// Apply the new settings
	appState.Settings = projectSettings

	// Rebuild editors
	appState.RebuildEditors()

	// List all image files in the input directory
	imageFiles, err := io_utils.ListFiles(projectSettings.InputDirectory)
	if err != nil || len(imageFiles) == 0 {
		return fmt.Errorf("Could not list input directory %q: %v", projectSettings.InputDirectory, err)
	}

	appState.ImageFiles = imageFiles
	appState.CurrentIndex = 0
	appState.loadImageAtIndex(0)

	return nil
}

// TODO: Do we need this custom struct? Perhaps it is possible to just pass in the pointer and error as two parameters.

// ApplyProjectLoadingResult processes a result returned by a background dialog goroutine.
func (appState *AppState) ApplyProjectLoadingResult(r ProjectLoadingResult) {
	appState.CurrentlyLoadingProject = false
	if r.Err == dialog.ErrCancelled || r.Settings == nil {
		return
	}
	if r.Err != nil {
		log.Printf("Failed to open project: %v", r.Err)
		return
	}
	appState.ApplyProject(r.Settings)
}

func (appState *AppState) RecalculateSuperpixels(preserveSelection bool, showToast bool) error {

	fmt.Println("Calculating superpixels...")
	start := time.Now().UnixMicro()

	// Snapshot the current selection as a mask before the labels change
	var oldSelectionMask *image.Gray16
	if preserveSelection && appState.ActiveResult.SpCount > 0 && len(appState.SelectionIndex) > 0 {
		oldSelectionMask = appState.ActiveResult.SelectionToMask(appState.SelectionIndex)
	}

	// Re-run the superpixel algorithm
	newResult, err := superpixel.Run(
		appState.Settings.Superpixel.SpAlgorithmId,
		appState.CurrentImage,
		appState.Settings.Superpixel.SpAlgorithmParameters)
	if err != nil {
		return err
	}

	appState.ActiveResult = newResult

	// Restore the selection by matching old mask pixels to new superpixels,
	// or reset to an empty selection when there was no previous result.
	if oldSelectionMask != nil {
		appState.SelectionIndex = appState.ActiveResult.SelectFromMask(oldSelectionMask, 0.5)
	} else {
		appState.SelectionIndex = make([]bool, appState.ActiveResult.SpCount)
	}

	// Rebuild the selection outline path spec for rendering
	appState.SelectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(*appState.ActiveResult.LabelMap, appState.SelectionIndex)

	// Log
	end := time.Now().UnixMicro()
	fmt.Printf("Finished calculating superpixels in %.1f milliseconds\n", float64(end-start)/1000)

	if showToast {
		// Add a toast
		appState.Toasts.EnqueueType(time.Now(), giotoast.TypeNeutral, "Rebuilt superpixels.", 1*time.Second)
	}
	return nil

}

func (appState *AppState) LoadNextImage() {
	if appState.CurrentIndex < len(appState.ImageFiles)-1 {
		appState.loadImageAtIndex(appState.CurrentIndex + 1)
	}
}

func (appState *AppState) LoadPreviousImage() {
	if appState.CurrentIndex > 0 {
		appState.loadImageAtIndex(appState.CurrentIndex - 1)
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
	for i := appState.CurrentIndex + 1; i < len(appState.ImageFiles); i++ {
		maskPath := appState.maskPathForIndex(i)
		if _, err := os.Stat(maskPath); os.IsNotExist(err) {
			appState.loadImageAtIndex(i)
			return
		}
	}
}

// LoadPreviousUnlabeledImage finds the previous image without an existing mask file and loads it.
func (appState *AppState) LoadPreviousUnlabeledImage() {
	for i := appState.CurrentIndex - 1; i >= 0; i-- {
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
	if appState.CurrentImage != nil && appState.SelectionMade() {
		appState.SaveCurrentMask(false, false)
	}

	// Update the current image and run the superpixel algorithm
	appState.CurrentImage = io_utils.ToNRGBA(io_utils.LoadImage(appState.ImageFiles[i]))
	err := appState.RecalculateSuperpixels(false, false)
	if err != nil {
		return err
	}

	// Reset the outline for showing the existing mask
	appState.ExistingMask = nil
	appState.ExistingMaskPathSpec = rendering.GetEmptySelectionOutline()

	maskPath := appState.maskPathForIndex(i)
	if _, err := os.Stat(maskPath); err == nil {
		loadedMask := io_utils.ToGray16(io_utils.LoadImage(maskPath))
		if superpixel_utils.HasNonZeroPixels(loadedMask) {
			appState.ExistingMask = loadedMask
			appState.ExistingMaskPathSpec = rendering.CalculateOutlineFromMask(loadedMask)
		}
	}

	// Rebuild image paint op.
	appState.ImgBaseOp = paint.NewImageOp(appState.CurrentImage)
	appState.ImgBaseOp.Filter = paint.FilterNearest

	// Reset view and hover.
	appState.FitToWindow = true
	appState.HoverPathSpec = rendering.GetEmptySelectionOutline()

	appState.SyncEditors()

	appState.CurrentIndex = i
	appState.Window.Invalidate()

	return nil

}

// SaveCurrentMask writes the current selection as a Gray16 PNG to the output directory.
func (appState *AppState) SaveCurrentMask(evenIfEmpty bool, recalculateOutline bool) {

	if appState.ActiveResult.SpCount != 0 && (appState.SelectionMade() || evenIfEmpty) {

		// Rasterize the current selection
		mask := appState.ActiveResult.SelectionToMask(appState.SelectionIndex)

		// Save as image
		outPath := appState.maskPathForIndex(appState.CurrentIndex)
		err := io_utils.SaveImage(outPath, mask)

		if err != nil {
			log.Printf("Failed to save mask to %s: %v", outPath, err)
			appState.Toasts.EnqueueType(time.Now(), giotoast.TypeError, fmt.Sprintf("Failed to save %s", filepath.Base(outPath)), 2*time.Second)
			return
		} else {

			if recalculateOutline {
				// Recalculate the ExistingMaskPathSpec
				appState.ExistingMaskPathSpec = rendering.CalculateOutlineFromMask(mask)
			}

			// Add toast
			appState.Toasts.EnqueueType(time.Now(), giotoast.TypeSuccess, fmt.Sprintf("Saved %s", filepath.Base(outPath)), 2*time.Second)
		}
	}

}

// maskPathForIndex returns the output PNG path for image at index i.
// The base name of the source file is kept but the extension is replaced with .png.
func (appState *AppState) maskPathForIndex(i int) string {
	base := filepath.Base(appState.ImageFiles[i])
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + ".png"
	return filepath.Join(appState.Settings.OutputDirectory, name)
}

func (appState *AppState) SelectionMade() bool {
	for _, selected := range appState.SelectionIndex {
		if selected {
			return true
		}
	}
	return false
}

func (appState *AppState) ClearSelection(showToast bool) {
	for i := range appState.SelectionIndex {
		appState.SelectionIndex[i] = false
	}
	appState.SelectionPathSpec = rendering.GetEmptySelectionOutline()
	appState.Window.Invalidate()
	if showToast {
		appState.Toasts.EnqueueType(time.Now(), giotoast.TypeWarning, "Cleared selection.", 1*time.Second)
	}
}
