package app_config

import (
	"fmt"
	"log"
	"slices"

	"gioui.org/text"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

// RebuildEditors sets up the editors in the UI for the current superpixel parameters.
func (appState *AppState) RebuildEditors() {

	appState.SuperpixelParameterEditors = nil
	appState.SuperpixelParameterEditorIds = nil
	appState.SuperpixelParameterEditorTitles = nil
	appState.OptimizeParameterCheckboxes = nil

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

		appState.OptimizeParameterCheckboxes = append(appState.OptimizeParameterCheckboxes, widget.Bool{Value: false})
	}
}

// SelectedOptimizeParameters returns the ids of the superpixel parameters currently
// checked for optimization.
func (appState *AppState) SelectedOptimizeParameters() []string {
	var selected []string
	for i, checkbox := range appState.OptimizeParameterCheckboxes {
		if checkbox.Value {
			selected = append(selected, appState.SuperpixelParameterEditorIds[i])
		}
	}
	return selected
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

func (appState *AppState) UiLocked() bool {
	return appState.CurrentlyLoadingProject || appState.CurrentlyOptimizing
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
	if appState.ActiveImage != nil {
		appState.RecalculateSuperpixels(true, true)
	}

	return nil
}
