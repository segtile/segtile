package ui

import (
	"fmt"
	"path/filepath"

	"github.com/sqweek/dialog"
	"github.com/struffel/annotation-tool/internal/app_config"
	"github.com/struffel/annotation-tool/pkg/superpixel_factory"
)

// triggerOpenProject spawns a background goroutine to open an existing project.
// Results are delivered via projectLoadResult for the event loop to apply.
func triggerOpenProject(appState *app_config.AppState) {
	if appState.ProjectLoading {
		return
	}
	appState.ProjectLoading = true
	go func() {
		cfg, outputDir, err := showOpenProjectDialogs()
		appState.ProjectLoadResult <- app_config.ProjectResult{cfg: cfg, outputDir: outputDir, err: err}
		appState.Window.Invalidate()
	}()
}

// triggerNewProject spawns a background goroutine to create a new project.
// Results are delivered via projectLoadResult for the event loop to apply.
func triggerNewProject(appState *app_config.AppState) {
	if appState.ProjectLoading {
		return
	}
	appState.ProjectLoading = true
	go func() {
		cfg, outputDir, err := showNewProjectDialogs()
		appState.ProjectLoadResult <- app_config.ProjectResult{cfg: cfg, outputDir: outputDir, err: err}
		appState.Window.Invalidate()
	}()
}

// showOpenProjectDialogs prompts the user to pick an existing config file.
// Must be called from a background goroutine.
func showOpenProjectDialogs() (*app_config.ProjectSettings, string, error) {
	path, err := dialog.File().
		Title("Open Configuration File").
		Filter("YAML config", "yaml", "yml").
		Load()
	if err != nil {
		return nil, "", err // includes dialog.ErrCancelled
	}
	cfg, err := app_config.Load(path)
	if err != nil {
		return nil, "", fmt.Errorf("failed to load config %q: %w", path, err)
	}
	return cfg, filepath.Dir(path), nil
}

// showNewProjectDialogs prompts the user to select an input directory and then
// an output directory, then initialises a new project.
// Must be called from a background goroutine.
func showNewProjectDialogs() (*app_config.ProjectSettings, string, error) {
	inputDir, err := dialog.Directory().
		Title("Select Input Directory (raw images)").Browse()
	if err != nil {
		return nil, "", err
	}

	outputDir, err := dialog.Directory().
		Title("Select Output Directory (stores masks and config)").Browse()
	if err != nil {
		return nil, "", err
	}

	impl, err := superpixel_factory.GetSuperpixelImplementationByName("slic")
	if err != nil {
		return nil, "", err
	}
	impl.SetRecommendedParameters()
	params := make(map[string]float64)
	for _, p := range impl.GetSupportedParameters() {
		v, _ := impl.GetParameter(p)
		params[string(p)] = v
	}
	spCfg := app_config.SuperpixelSettings{Algorithm: "slic", Parameters: params}

	cfg, err := app_config.InitNew(outputDir, inputDir, spCfg)
	if err != nil {
		return nil, "", fmt.Errorf("failed to initialize project: %w", err)
	}
	return cfg, outputDir, nil
}
