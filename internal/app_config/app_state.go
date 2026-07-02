package app_config

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"slices"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/widget"
	"github.com/sqweek/dialog"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/rendering"
	"github.com/struffel/annotation-tool/internal/theme"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

const defaultSpAlgorithmId = "slic"

// SuperpixelSettings stores the algorithm name and its parameters.
// This is used for yaml persistence.
type SuperpixelSettings struct {
	SpAlgorithmId         string         `yaml:"algorithm"`
	SpAlgorithmParameters map[string]any `yaml:"parameters"`
}

// ProjectSettings stores all settings of a project,
// it is used for yaml persistence.
type ProjectSettings struct {
	OutputDirectory string             `yaml:"-"`
	InputDirectory  string             `yaml:"input_directory"`
	Superpixel      SuperpixelSettings `yaml:"superpixel"`
}

type AppState struct {

	// Project
	Settings   ProjectSettings
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
	Window                     *app.Window
	CanvasTag                  bool
	Theme                      *theme.AppTheme
	SuperpixelParameterEditors map[string]*widget.Editor
	RenderSelectionOutlines    bool
	FirstButton                widget.Clickable
	LastButton                 widget.Clickable
	PrevUnlabeledButton        widget.Clickable
	PrevButton                 widget.Clickable
	NextUnlabeledButton        widget.Clickable
	NextButton                 widget.Clickable
	OpenProjectButton          widget.Clickable
	NewProjectButton           widget.Clickable
	OptimizeButton             widget.Clickable

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
	defaultParameters, _ := superpixel.GetDefaultParameters(defaultSpAlgorithmId)
	return SuperpixelSettings{
		SpAlgorithmId:         defaultSpAlgorithmId,
		SpAlgorithmParameters: defaultParameters,
	}
}

func NewAppState() *AppState {

	settings := ProjectSettings{
		InputDirectory: "",
		Superpixel:     NewSuperpixelSettings(),
	}

	appState := AppState{
		Settings:                   settings,
		FitToWindow:                true,
		RenderSelectionOutlines:    true,
		Theme:                      theme.Theme,
		SuperpixelParameterEditors: make(map[string]*widget.Editor),
		ProjectLoadResult:          make(chan ProjectLoadingResult, 1),
	}

	appState.RebuildEditors()

	return &appState
}

// Set up the editors for the current superpixel parameters.
func (appState *AppState) RebuildEditors() {
	appState.SuperpixelParameterEditors = make(map[string]*widget.Editor)
	for paramName, paramValue := range appState.Settings.Superpixel.SpAlgorithmParameters {
		ed := &widget.Editor{SingleLine: true, Submit: true, Filter: "-.0123456789"}
		ed.SetText(fmt.Sprintf("%g", paramValue))
		appState.SuperpixelParameterEditors[paramName] = ed
	}
	appState.SyncEditors()
}

// SyncEditors updates each superpixel parameter editor's text to reflect
// the current values in the superpixel implementation.
func (appState *AppState) SyncEditors() error {

	for paramName, paramValue := range appState.Settings.Superpixel.SpAlgorithmParameters {
		if ed, ok := appState.SuperpixelParameterEditors[paramName]; ok {
			ed.SetText(fmt.Sprintf("%g", paramValue))
		} else {
			return fmt.Errorf("editor for parameter %s not found", paramName)
		}
	}

	return nil
}

func (appState *AppState) SetSpAlgorithm(id string) error {

	// Set the new algorithm id
	if !slices.Contains(superpixel.ListAlgorithms(), id) {
		return fmt.Errorf("unknown superpixel algorithm %s", id)
	}
	appState.Settings.Superpixel.SpAlgorithmId = id

	// Load the right set of parameters
	defaultParameters, err := superpixel.GetDefaultParameters(id)
	if err != nil {
		return err
	}
	appState.Settings.Superpixel.SpAlgorithmParameters = defaultParameters

	// Rebuild the editors for the new params
	appState.RebuildEditors()

	return nil
}

// ApplyProject configures the AppState for the given project, saves any current mask,
// and loads the first image.
func (appState *AppState) ApplyProject(projectSettings ProjectSettings) error {

	// Save the old image before loading the new project
	if appState.CurrentImage != nil {
		appState.saveCurrentMask()
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
	appState.ApplyProject(*r.Settings)
}

func (appState *AppState) RecalculateSuperpixels() error {

	fmt.Println("Calculating superpixels...")
	start := time.Now().UnixMicro()

	// Snapshot the current selection as a mask before the labels change
	var oldSelectionMask *image.Gray16
	if appState.ActiveResult.SpCount > 0 && len(appState.SelectionIndex) > 0 {
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
	if appState.CurrentImage != nil && appState.selectionMade() {
		appState.saveCurrentMask()
	}

	// Update the current image and run the superpixel algorithm
	appState.CurrentImage = io_utils.ToNRGBA(io_utils.LoadImage(appState.ImageFiles[i]))
	err := appState.RecalculateSuperpixels()
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
			appState.ExistingMaskPathSpec = rendering.CalculateOutlineFromMask(*loadedMask)
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

// saveCurrentMask writes the current selection as a Gray16 PNG to the output directory.
func (appState *AppState) saveCurrentMask() {

	/*if appState.ActiveResult.SpCount == 0 {
		return
	}*/
	mask := appState.ActiveResult.SelectionToMask(appState.SelectionIndex)
	outPath := appState.maskPathForIndex(appState.CurrentIndex)
	io_utils.SaveImage(outPath, mask)
}

// maskPathForIndex returns the output PNG path for image at index i.
// The base name of the source file is kept but the extension is replaced with .png.
func (appState *AppState) maskPathForIndex(i int) string {
	base := filepath.Base(appState.ImageFiles[i])
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + ".png"
	return filepath.Join(appState.Settings.OutputDirectory, name)
}

func (appState *AppState) selectionMade() bool {
	for _, selected := range appState.SelectionIndex {
		if selected {
			return true
		}
	}
	return false
}
