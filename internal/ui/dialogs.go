package ui

import (
	"fmt"
	"image/color"

	"github.com/ncruces/zenity"
	"github.com/struffel/annotation-tool/internal/core"
)

// Creates a goroutine to open an existing project.
// Results are delivered via projectLoadResult for the event loop to apply.
func triggerOpenProject(appState *core.AppState) {
	if appState.CurrentlyLoadingProject {
		return
	}
	appState.CurrentlyLoadingProject = true
	go func() {
		result := showOpenProjectDialogs()
		appState.ProjectLoadResult <- result
		appState.Window.Invalidate()
	}()
}

// Creates a goroutine to create a new project.
// Results are delivered via projectLoadResult for the event loop to apply.
func triggerNewProject(appState *core.AppState) {
	if appState.CurrentlyLoadingProject {
		return
	}
	appState.CurrentlyLoadingProject = true
	go func() {
		result := showNewProjectDialogs()
		appState.ProjectLoadResult <- result
		appState.Window.Invalidate()
	}()
}

// Prompts the user to pick an existing project's settings file.
// Must be called from a background goroutine.
// Returns nil if the user canceled the dialog.
func showOpenProjectDialogs() *core.ProjectSettings {
	path, err := zenity.SelectFile(
		zenity.Title("Open Project Settings File"),
		zenity.FileFilter{
			Name:     "YAML config",
			Patterns: []string{"*.yaml", "*.yml"},
		},
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return nil
		}
		return &core.ProjectSettings{Err: err}
	}
	return core.LoadProjectSettings(path)
}

// showNewProjectDialogs prompts the user to select an input directory and then
// an output directory, then initialises a new project.
// Must be called from a background goroutine.
// Returns nil if the user canceled a dialog.
func showNewProjectDialogs() *core.ProjectSettings {

	// Prompt for the input directory
	inputDir, err := zenity.SelectFile(
		zenity.Title("Select Input Directory (raw images)"),
		zenity.Directory(),
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return nil
		}
		return &core.ProjectSettings{Err: err}
	}

	// Prompt for the output directory
	outputDir, err := zenity.SelectFile(
		zenity.Title("Select Output Directory (stores masks and metadata)"),
		zenity.Directory(),
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return nil
		}
		return &core.ProjectSettings{Err: err}
	}

	cfg, err := core.NewProjectSettings(outputDir, inputDir, core.NewSuperpixelSettings())
	if err != nil {
		return &core.ProjectSettings{Err: fmt.Errorf("failed to initialize project: %w", err)}
	}
	return cfg
}

func showSaveEmptyMaskWarning() (ok bool) {
	return zenity.Question(
		"No annotation has been made. Save empty annotation?",
		zenity.Title("Save empty annotation?"),
	) == nil
}

// triggerColorPick opens the native color picker dialog on a background goroutine
// for the theme outline color with the given index.
// The result is delivered via ColorPickResult for the event loop to apply.
func triggerColorPick(appState *core.AppState, colorIndex int) {
	if appState.ActiveSettings == nil || appState.ActiveSettings.Theme == nil {
		return
	}
	current := *appState.ActiveSettings.Theme.OutlineColors()[colorIndex]
	go func() {
		picked, err := zenity.SelectColor(
			zenity.Title("Select Outline Color"),
			zenity.Color(current),
		)
		var result *color.NRGBA
		if err == nil {
			r, g, b, _ := picked.RGBA()
			result = &color.NRGBA{
				R: uint8(r >> 8),
				G: uint8(g >> 8),
				B: uint8(b >> 8),
				A: current.A, // Keep the existing alpha; the picker only controls RGB.
			}
		}
		appState.ColorPickResult <- core.ThemeColorPickResult{ColorIndex: colorIndex, Color: result}
		if appState.Window != nil {
			appState.Window.Invalidate()
		}
	}()
}
