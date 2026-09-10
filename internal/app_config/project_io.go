package app_config

import (
	"errors"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/ncruces/zenity"
	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

const ConfigFileName = "settings.yaml"
const defaultSpAlgorithmId = "snic"

// ProjectLoadingResult carries the outcome of a background project-load dialog.
type ProjectLoadingResult struct {
	Settings *ProjectSettings
	Err      error
}

// ProjectSettings stores all settings of a project that need to be persisted to disk.
type ProjectSettings struct {
	OutputDirectory     string             `yaml:"-"` // The output directory is not persisted in the yaml settings file, it is derived from the location of the settings file itself.
	InputDirectory      string             `yaml:"input_directory"`
	Superpixel          SuperpixelSettings `yaml:"superpixel"`
	EnableAutoFloodFill bool               `yaml:"auto_flood_fill"`
	ShrinkGrowAmount    float32            `yaml:"shrink_grow_amount"`
	SmoothAmount        float32            `yaml:"smooth_amount"`
}

// SuperpixelSettings stores the algorithm name and its parameters.
// This is used for yaml persistence.
type SuperpixelSettings struct {
	SpAlgorithmId         string             `yaml:"algorithm"`
	SpAlgorithmParameters map[string]float64 `yaml:"parameters"`
}

func NewSuperpixelSettings() SuperpixelSettings {
	defaultParameters, _ := superpixel.GetDefaultParameterValues(defaultSpAlgorithmId)
	return SuperpixelSettings{
		SpAlgorithmId:         defaultSpAlgorithmId,
		SpAlgorithmParameters: defaultParameters,
	}
}

// normalizeSuperpixelSettings ensures that the SuperpixelSettings are consistent and complete.
// It fills in default values for missing parameters and removes any unknown parameters.
func (settings *SuperpixelSettings) normalizeSuperpixelSettings() error {

	// Get information about the intended parameters for the algorithm
	parameterInfo, err := superpixel.ListParameterInfo(settings.SpAlgorithmId)
	if err != nil {
		return err
	}

	// If the parameter map is nil, initialize it with default values for all parameters
	if settings.SpAlgorithmParameters == nil {
		settings.SpAlgorithmParameters = make(map[string]float64, len(parameterInfo))
	}

	// Ensure that all parameters are present in the map
	for _, paramInfo := range parameterInfo {
		if _, ok := settings.SpAlgorithmParameters[paramInfo.Id]; !ok {
			settings.SpAlgorithmParameters[paramInfo.Id] = paramInfo.Default
		}
	}

	// Remove any parameters that are not part of the algorithm's parameter list
	for paramId, _ := range settings.SpAlgorithmParameters {
		_, err := superpixel.GetParameterInfo(settings.SpAlgorithmId, paramId)
		if err != nil {
			delete(settings.SpAlgorithmParameters, paramId)
		}
	}

	return nil
}

// Load reads and unmarshals a config file from the given path.
func Load(path string) ProjectLoadingResult {

	// Read the file
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ProjectLoadingResult{Settings: nil, Err: &SettingsFileReadError{path: path, err: err}}
	}

	// Parse yaml
	var settings ProjectSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return ProjectLoadingResult{Settings: nil, Err: &SettingsFileYamlParseError{err: err}}
	}

	// Set the output directory based on the location of the settings file
	settings.OutputDirectory = filepath.Dir(path)

	// Test if the input directory exists
	if _, err := os.Stat(settings.InputDirectory); errors.Is(err, fs.ErrNotExist) {

		// If it does not exist, try treating it as a relative path from the settings
		settings.InputDirectory = filepath.Join(settings.OutputDirectory, settings.InputDirectory)

		if _, err = os.Stat(settings.InputDirectory); errors.Is(err, fs.ErrNotExist) {

			// If this also fails, return an error indicating that the input directory could not be found.
			return ProjectLoadingResult{Settings: nil, Err: &SettingsInputDirectoryError{path: settings.InputDirectory, err: err}}
		}

	}

	if err := settings.Superpixel.normalizeSuperpixelSettings(); err != nil {
		return ProjectLoadingResult{Settings: nil, Err: err}
	}

	return ProjectLoadingResult{Settings: &settings, Err: nil}
}

// Save marshals cfg and writes it to outputDir/settings.yaml,
// creating the directory if it does not exist.
func Save(settings *ProjectSettings) error {

	targetPath := filepath.Join(settings.OutputDirectory, ConfigFileName)

	if settings.OutputDirectory == "" {
		return &SettingsSaveError{path: "", err: fmt.Errorf("output directory is not set")}
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return &SettingsSaveError{path: targetPath, err: err}
	}

	if err := os.MkdirAll(settings.OutputDirectory, 0750); err != nil {
		return &SettingsSaveError{path: targetPath, err: err}
	}

	if err := os.WriteFile(targetPath, data, 0600); err != nil {
		return &SettingsSaveError{path: targetPath, err: err}
	}

	return nil
}

// NewProjectSettings creates a new ProjectSettings with the given directories and superpixel
// settings, saves it to disk and returns the result.
func NewProjectSettings(outputDir, inputDir string, spSettings SuperpixelSettings) (*ProjectSettings, error) {
	cfg := &ProjectSettings{
		InputDirectory:   inputDir,
		OutputDirectory:  outputDir,
		Superpixel:       spSettings,
		ShrinkGrowAmount: 0,
	}
	if err := Save(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ApplyProjectLoadingResult processes a result returned by a background dialog goroutine.
func (appState *AppState) ApplyProjectLoadingResult(loadingResult ProjectLoadingResult) {

	appState.CurrentlyLoadingProject = false

	if loadingResult.Err == zenity.ErrCanceled || loadingResult.Settings == nil {
		// User canceled the operation
		return
	}

	if loadingResult.Err != nil {
		log.Printf("Failed to open project: %v", loadingResult.Err)
		appState.Toasts.ShowType(time.Now(), giotoast.TypeError, loadingResult.Err.Error(), 5*time.Second)
		return
	}

	// Save the old image before loading the new project
	if appState.ActiveImage != nil {
		appState.SaveCurrentMask(false, true)
		appState.ActiveImage = nil
	}

	// Apply the new settings
	appState.Settings = loadingResult.Settings
	appState.EnableAutoFloodFill.Value = loadingResult.Settings.EnableAutoFloodFill
	appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(loadingResult.Settings.ShrinkGrowAmount)
	appState.SmoothAmount.Value = loadingResult.Settings.SmoothAmount

	// Rebuild editors
	appState.RebuildEditors()

	// List all image files in the input directory
	imageFiles, err := io_utils.ListFiles(loadingResult.Settings.InputDirectory)
	if err != nil || len(imageFiles) == 0 {
		log.Printf("Could not list input directory %q: %v", loadingResult.Settings.InputDirectory, err)
		appState.Toasts.ShowType(time.Now(), giotoast.TypeError, err.Error(), 5*time.Second)
		return
	}

	appState.ImageFiles = imageFiles
	appState.ImageFilesIndex = 0
	appState.loadImageAtIndex(0)
}

// RecreateExistingMaskFromDisk resolves the pending existing-mask decision by
// approximating the on-disk mask as a selection on the current superpixel segmentation,
// so the user can keep editing it.
func (appState *AppState) RecreateExistingMaskFromDisk() {
	appState.AwaitingMaskDecision = false
	if appState.ActiveImageExistingMask != nil {
		appState.SelectionIndex = appState.ActiveSp.SelectFromMask(appState.ActiveImageExistingMask, 0.5)
		appState.ReprocessSelection()
	}
	appState.Window.Invalidate()
}
