package app_config

import (
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
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
	"github.com/struffel/annotation-tool/pkg/superpixel_factory"
)

type AppState struct {

	// Project
	Settings        *ProjectSettings
	OutputDirectory string
	ImageFiles      []string

	// Images
	CurrentImage *image.NRGBA
	ExistingMask *image.Gray16
	CurrentIndex int

	// Superpixel module
	SuperpixelImplementation superpixel.SpImplementation

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
	SuperpixelParameterEditors map[superpixel.SpParameter]*widget.Editor
	RenderSelectionOutlines    bool
	FirstButton                widget.Clickable
	LastButton                 widget.Clickable
	PrevUnlabeledButton        widget.Clickable
	PrevButton                 widget.Clickable
	NextUnlabeledButton        widget.Clickable
	NextButton                 widget.Clickable
	OpenProjectButton          widget.Clickable
	NewProjectButton           widget.Clickable

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

func NewAppState() *AppState {
	impl, _ := superpixel_factory.NewSuperpixelImplementationByName("slic")
	impl.SetRecommendedParameters()
	editors := make(map[superpixel.SpParameter]*widget.Editor)
	for _, param := range impl.GetSupportedParameters() {
		value, _ := impl.GetParameter(param)
		ed := &widget.Editor{SingleLine: true, Submit: true, Filter: "-.0123456789"}
		ed.SetText(fmt.Sprintf("%g", value))
		editors[param] = ed
	}
	return &AppState{
		FitToWindow:                true,
		RenderSelectionOutlines:    true,
		Theme:                      theme.Theme,
		SuperpixelImplementation:   impl,
		SuperpixelParameterEditors: editors,
		ProjectLoadResult:          make(chan ProjectLoadingResult, 1),
	}
}

// ApplyProject configures the AppState for the given project, saves any current mask,
// and loads the first image.
func (appState *AppState) ApplyProject(projectSettings *ProjectSettings, outputDir string) {
	if appState.CurrentImage != nil {
		appState.saveCurrentMask()
		appState.CurrentImage = nil
	}

	impl, err := superpixel_factory.NewSuperpixelImplementationByName(projectSettings.Superpixel.Algorithm)
	if err != nil {
		log.Printf("Unknown algorithm %q: %v", projectSettings.Superpixel.Algorithm, err)
		return
	}
	impl.SetRecommendedParameters()
	for k, v := range projectSettings.Superpixel.Parameters {
		for _, param := range impl.GetSupportedParameters() {
			if param.String() == k {
				impl.SetParameter(param, v)
				break
			}
		}
	}
	appState.SuperpixelImplementation = impl

	editors := make(map[superpixel.SpParameter]*widget.Editor)
	for _, param := range impl.GetSupportedParameters() {
		value, _ := impl.GetParameter(param)
		ed := &widget.Editor{SingleLine: true, Submit: true, Filter: "-.0123456789"}
		ed.SetText(fmt.Sprintf("%g", value))
		editors[param] = ed
	}
	appState.SuperpixelParameterEditors = editors

	imageFiles, err := io_utils.ListFiles(projectSettings.InputDirectory)
	if err != nil || len(imageFiles) == 0 {
		log.Printf("Could not list input directory %q: %v", projectSettings.InputDirectory, err)
		return
	}

	appState.Settings = projectSettings
	appState.OutputDirectory = outputDir
	appState.ImageFiles = imageFiles
	appState.CurrentIndex = 0
	appState.loadImageAtIndex(0)
}

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
	appState.ApplyProject(r.Settings, r.OutputDirectory)
}

func (appState *AppState) RecalculateSuperpixels() {

	fmt.Println("Calculating superpixels...")
	start := time.Now().UnixMicro()

	// Snapshot the current selection as a mask before the labels change
	oldResult := appState.SuperpixelImplementation.GetResult()
	var oldSelectionMask *image.Gray16
	if oldResult.SpCount > 0 && len(appState.SelectionIndex) > 0 {
		oldSelectionMask = oldResult.SelectionToMask(appState.SelectionIndex)
	}

	// Re-run the superpixel algorithm
	err := appState.SuperpixelImplementation.Run()
	if err != nil {
		log.Fatalf("Error running superpixel algorithm: %v", err)
	}

	// Restore the selection by matching old mask pixels to new superpixels,
	// or reset to an empty selection when there was no previous result.
	result := appState.SuperpixelImplementation.GetResult()
	if oldSelectionMask != nil {
		appState.SelectionIndex = result.SelectFromMask(oldSelectionMask, 0.5)
	} else {
		appState.SelectionIndex = make([]bool, result.SpCount)
	}
	//appState.updateSelectionOutline()
	labelMap := appState.SuperpixelImplementation.GetResult().LabelMap
	appState.SelectionPathSpec = rendering.CalculateSelectionOutlineFromSlice(*labelMap, appState.SelectionIndex)

	end := time.Now().UnixMicro()
	fmt.Printf("Finished calculating superpixels in %.1f milliseconds\n", float64(end-start)/1000)

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
func (appState *AppState) loadImageAtIndex(i int) {

	// Save the current mask, if a selection is made
	if appState.CurrentImage != nil && appState.selectionMade() {
		appState.saveCurrentMask()
	}

	appState.CurrentImage = io_utils.ToNRGBA(io_utils.LoadImage(appState.ImageFiles[i]))
	appState.SuperpixelImplementation.SetImage(appState.CurrentImage)
	appState.RecalculateSuperpixels()

	// If an existing non-empty mask is on disk, store it for display as an orange
	// outline. Do not approximate it into the superpixel selection.
	appState.ExistingMask = nil
	appState.ExistingMaskPathSpec = rendering.GetEmptySelectionOutline()
	maskPath := appState.maskPathForIndex(i)
	if _, err := os.Stat(maskPath); err == nil {
		loadedMask := io_utils.ToGray16(io_utils.LoadImage(maskPath))
		if isMaskNonEmpty(loadedMask) {
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

	// Sync editor text to current parameter values.
	for _, param := range appState.SuperpixelImplementation.GetSupportedParameters() {
		if ed, ok := appState.SuperpixelParameterEditors[param]; ok {
			value, _ := appState.SuperpixelImplementation.GetParameter(param)
			ed.SetText(fmt.Sprintf("%g", value))
		}
	}

	appState.CurrentIndex = i
	if appState.Window != nil {
		appState.Window.Invalidate()
	}
}

// saveCurrentMask writes the current selection as a Gray16 PNG to the output directory.
func (appState *AppState) saveCurrentMask() {
	result := appState.SuperpixelImplementation.GetResult()
	if result.SpCount == 0 {
		return
	}
	mask := result.SelectionToMask(appState.SelectionIndex)
	outPath := appState.maskPathForIndex(appState.CurrentIndex)
	io_utils.SaveImage(outPath, mask)
}

// maskPathForIndex returns the output PNG path for image at index i.
// The base name of the source file is kept but the extension is replaced with .png.
func (appState *AppState) maskPathForIndex(i int) string {
	base := filepath.Base(appState.ImageFiles[i])
	ext := filepath.Ext(base)
	name := base[:len(base)-len(ext)] + ".png"
	return filepath.Join(appState.OutputDirectory, name)
}

func (appState *AppState) selectionMade() bool {
	for _, selected := range appState.SelectionIndex {
		if selected {
			return true
		}
	}
	return false
}

func isMaskNonEmpty(mask *image.Gray16) bool {
	for _, p := range mask.Pix {
		if p != 0 {
			return true
		}
	}
	return false
}
