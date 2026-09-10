package ui

import (
	"fmt"

	"github.com/ncruces/zenity"
	"github.com/struffel/annotation-tool/internal/app_config"
)

// Creates a goroutine to open an existing project.
// Results are delivered via projectLoadResult for the event loop to apply.
func triggerOpenProject(appState *app_config.AppState) {
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
func triggerNewProject(appState *app_config.AppState) {
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

// Prompts the user to pick an existing config file.
// Must be called from a background goroutine.
// Returns nil if the user canceled the dialog.
func showOpenProjectDialogs() *app_config.ProjectSettings {
	path, err := zenity.SelectFile(
		zenity.Title("Open Configuration File"),
		zenity.FileFilter{
			Name:     "YAML config",
			Patterns: []string{"*.yaml", "*.yml"},
		},
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return nil
		}
		return &app_config.ProjectSettings{Err: err}
	}
	return app_config.LoadProjectSettings(path)
}

// showNewProjectDialogs prompts the user to select an input directory and then
// an output directory, then initialises a new project.
// Must be called from a background goroutine.
// Returns nil if the user canceled a dialog.
func showNewProjectDialogs() *app_config.ProjectSettings {

	// Prompt for the input directory
	inputDir, err := zenity.SelectFile(
		zenity.Title("Select Input Directory (raw images)"),
		zenity.Directory(),
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return nil
		}
		return &app_config.ProjectSettings{Err: err}
	}

	// Prompt for the output directory
	outputDir, err := zenity.SelectFile(
		zenity.Title("Select Output Directory (stores masks and config)"),
		zenity.Directory(),
	)
	if err != nil {
		if err == zenity.ErrCanceled {
			return nil
		}
		return &app_config.ProjectSettings{Err: err}
	}

	cfg, err := app_config.NewProjectSettings(outputDir, inputDir, app_config.NewSuperpixelSettings())
	if err != nil {
		return &app_config.ProjectSettings{Err: fmt.Errorf("failed to initialize project: %w", err)}
	}
	return cfg
}

func showSaveEmptyMaskWarning() (ok bool) {
	return zenity.Question(
		"No annotation has been made. Save empty annotation?",
		zenity.Title("Save empty annotation?"),
	) == nil
}
