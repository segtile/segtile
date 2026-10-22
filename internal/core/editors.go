package core

import (
	"fmt"
	"log"
	"slices"

	"gioui.org/text"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

// RebuildEditors sets up the editors in the UI for the current superpixel parameters.
func (appState *AppState) RebuildEditors() {

	appState.SuperpixelParameterEditors = nil

	// No editors if there are no settings
	if appState.ActiveSettings == nil {
		return
	}

	if err := appState.ActiveSettings.Superpixel.normalize(); err != nil {
		log.Printf("Failed to normalize superpixel settings: %v", err)
		return
	}

	parameterInfo, err := superpixel.ListParameterInfo(appState.ActiveSettings.Superpixel.SpAlgorithmId)
	if err != nil {
		log.Printf("Failed to list superpixel parameter info: %v", err)
		return
	}

	// Turn the parameter list into aligned editor data for the UI.
	for _, paramInfo := range parameterInfo {
		paramValue := appState.ActiveSettings.Superpixel.SpAlgorithmParameters[paramInfo.Id]
		editor := widget.Editor{SingleLine: true, Submit: true, Filter: "-.0123456789", Alignment: text.End}
		editor.SetText(fmt.Sprintf("%g", paramValue))
		appState.SuperpixelParameterEditors = append(appState.SuperpixelParameterEditors, SuperpixelParameterEditor{
			Editor: editor,
			Id:     paramInfo.Id,
			Title:  paramInfo.Title,
		})
	}
}

// SelectedOptimizeParameters returns the ids of the superpixel parameters currently
// checked for optimization.
func (appState *AppState) SelectedOptimizeParameters() []string {
	var selected []string
	for _, entry := range appState.SuperpixelParameterEditors {
		if entry.OptimizeCheckbox.Value {
			selected = append(selected, entry.Id)
		}
	}
	return selected
}

// SyncEditors updates each superpixel parameter editor's text to reflect
// the current values in the superpixel implementation.
func (appState *AppState) SyncEditors() error {
	for i := range appState.SuperpixelParameterEditors {
		paramId := appState.SuperpixelParameterEditors[i].Id
		paramValue, ok := appState.ActiveSettings.Superpixel.SpAlgorithmParameters[paramId]
		if !ok {
			return fmt.Errorf("Value for parameter %s not found", paramId)
		}
		appState.SuperpixelParameterEditors[i].Editor.SetText(fmt.Sprintf("%g", paramValue))
	}

	return nil
}

// maxStrokeWidth is the maximum outline stroke width selectable via the theme sliders.
const maxStrokeWidth = 5.0

// StrokeWidthToSlider maps an outline stroke width (in pixels) to a 0..1 slider value.
func StrokeWidthToSlider(width float32) float32 {
	return (width - app_theme.MinStrokeWidth) / (maxStrokeWidth - app_theme.MinStrokeWidth)
}

// SliderToStrokeWidth maps a 0..1 slider value to an outline stroke width (in pixels).
// The slider cannot go below app_theme.MinStrokeWidth.
func SliderToStrokeWidth(sliderValue float32) float32 {
	return app_theme.MinStrokeWidth + sliderValue*(maxStrokeWidth-app_theme.MinStrokeWidth)
}

// SyncThemeEditors updates the theme controls to reflect the current theme.
// The outline colors are read directly from the theme when rendering, so only
// the stroke width sliders need syncing.
func (appState *AppState) SyncThemeEditors() {
	theme := appState.Theme()
	appState.ThemeSelectionWidthSlider.Value = StrokeWidthToSlider(theme.SelectionStrokeWidth)
	appState.ThemeHoverWidthSlider.Value = StrokeWidthToSlider(theme.HoverStrokeWidth)
}

func (appState *AppState) UiLocked() bool {
	return appState.CurrentlyLoadingProject || appState.CurrentlyOptimizing
}

func (appState *AppState) SetSpAlgorithm(id string) error {

	// Set the new algorithm id
	if !slices.Contains(superpixel.ListAlgorithms(), id) {
		return fmt.Errorf("Unknown superpixel algorithm: %s", id)
	}
	if appState.ActiveSettings.Superpixel.SpAlgorithmId != id {
		appState.SelectionMaskDiffersVersionOnDisk = true
	}
	appState.ActiveSettings.Superpixel.SpAlgorithmId = id

	// Load the right set of parameters
	defaultParameters, err := superpixel.GetDefaultParameterValues(id)
	if err != nil {
		return err
	}
	appState.ActiveSettings.Superpixel.SpAlgorithmParameters = defaultParameters

	// Rebuild the editors for the new params
	appState.RebuildEditors()

	// Recalculate the superpixels for the current image
	if appState.ActiveImage != nil {
		appState.RecalculateSuperpixels(true)
	}

	return nil
}
