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
		settings, err := showOpenProjectDialogs()
		appState.ProjectLoadResult <- app_config.ProjectLoadingResult{Settings: settings, Err: err}
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
		cfg, err := showNewProjectDialogs()
		appState.ProjectLoadResult <- app_config.ProjectLoadingResult{Settings: cfg, Err: err}
		appState.Window.Invalidate()
	}()
}

// Prompts the user to pick an existing config file.
// Must be called from a background goroutine.
func showOpenProjectDialogs() (*app_config.ProjectSettings, error) {
	path, err := zenity.SelectFile(
		zenity.Title("Open Configuration File"),
		zenity.FileFilter{
			Name:     "YAML config",
			Patterns: []string{"*.yaml", "*.yml"},
		},
	)
	if err != nil {
		return nil, err // includes zenity.ErrCanceled
	}
	cfg, err := app_config.Load(path)
	if err != nil {
		return nil, fmt.Errorf("failed to load config %q: %w", path, err)
	}
	return cfg, nil
}

// showNewProjectDialogs prompts the user to select an input directory and then
// an output directory, then initialises a new project.
// Must be called from a background goroutine.
func showNewProjectDialogs() (*app_config.ProjectSettings, error) {

	// Prompt for the input directory
	inputDir, err := zenity.SelectFile(
		zenity.Title("Select Input Directory (raw images)"),
		zenity.Directory(),
	)
	if err != nil {
		return nil, err
	}

	// Prompt for the output directory
	outputDir, err := zenity.SelectFile(
		zenity.Title("Select Output Directory (stores masks and config)"),
		zenity.Directory(),
	)
	if err != nil {
		return nil, err
	}

	cfg, err := app_config.NewProjectSettings(outputDir, inputDir, app_config.NewSuperpixelSettings())
	if err != nil {
		return nil, fmt.Errorf("failed to initialize project: %w", err)
	}
	return cfg, nil
}

func showSaveEmptyMaskWarning() (ok bool) {
	return zenity.Question(
		"No annotation has been made. Save empty annotation?",
		zenity.Title("Save empty annotation?"),
	) == nil
}
