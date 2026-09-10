package app_config

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

const DefaultConfigFileName = "settings.yaml"
const defaultSpAlgorithmId = "snic"

// ProjectSettings stores all settings of a project that need to be persisted to disk.
type ProjectSettings struct {
	SettingsFile        string             `yaml:"-"` // The path to the settings file itself, not persisted in the yaml.
	InputDirectory      string             `yaml:"input_directory"`
	Superpixel          SuperpixelSettings `yaml:"superpixel"`
	EnableAutoFloodFill bool               `yaml:"auto_flood_fill"`
	ShrinkGrowAmount    float32            `yaml:"shrink_grow_amount"`
	SmoothAmount        float32            `yaml:"smooth_amount"`
	Err                 error              `yaml:"-"` // The error is also not persisted to disk.
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

// LoadProjectSettings reads and unmarshals a config file from the given path.
// This can be either a direct path to a settings yaml file or the path to a directory,
// in which case it will look for a settings yaml file within that directory.
func LoadProjectSettings(settingsPath string) *ProjectSettings {

	// Check if the given path exists and whether it is a file or directory
	stat, err := os.Stat(settingsPath)
	if err != nil {
		return &ProjectSettings{Err: &SettingsFileReadError{path: settingsPath, err: err}}
	}

	// If the path is a directory, append the default config file name
	if stat.IsDir() {
		settingsPath = filepath.Join(settingsPath, DefaultConfigFileName)
	}

	// Then try reading the settings
	data, err := os.ReadFile(filepath.Clean(settingsPath))
	if err != nil {
		return &ProjectSettings{Err: &SettingsFileReadError{path: settingsPath, err: err}}
	}

	// Parse yaml
	var settings ProjectSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return &ProjectSettings{Err: &SettingsFileYamlParseError{err: err}}
	}

	// Store the settings file path
	settings.SettingsFile = settingsPath

	// Test if the input directory exists
	if _, err := os.Stat(settings.InputDirectory); err != nil {

		// If it can not be read, try treating it as a relative path from the directory of the settings file
		settings.InputDirectory = filepath.Join(filepath.Dir(settings.SettingsFile), settings.InputDirectory)

		if _, err = os.Stat(settings.InputDirectory); err != nil {

			// If this also fails, return an error indicating that the input directory could not be found.
			return &ProjectSettings{Err: &SettingsInputDirectoryError{path: settings.InputDirectory, err: err}}
		}

	}

	if err := settings.Superpixel.normalizeSuperpixelSettings(); err != nil {
		return &ProjectSettings{Err: err}
	}

	return &settings
}

// SaveProjectSettings marshals cfg and writes it to the settings file
// creating the directory if it does not exist.
func SaveProjectSettings(settings *ProjectSettings) error {

	if settings.SettingsFile == "" {
		return &SettingsSaveError{path: "", err: fmt.Errorf("settings file path is not set")}
	}

	data, err := yaml.Marshal(settings)
	if err != nil {
		return &SettingsSaveError{path: settings.SettingsFile, err: err}
	}

	if err := os.MkdirAll(filepath.Dir(settings.SettingsFile), 0750); err != nil {
		return &SettingsSaveError{path: settings.SettingsFile, err: err}
	}

	if err := os.WriteFile(settings.SettingsFile, data, 0600); err != nil {
		return &SettingsSaveError{path: settings.SettingsFile, err: err}
	}

	return nil
}

// NewProjectSettings creates a new ProjectSettings with the given directories and superpixel settings, saves it to disk and returns the result.
func NewProjectSettings(outputDir, inputDir string, spSettings SuperpixelSettings) (*ProjectSettings, error) {
	cfg := &ProjectSettings{
		SettingsFile:     filepath.Join(outputDir, DefaultConfigFileName),
		InputDirectory:   inputDir,
		Superpixel:       spSettings,
		ShrinkGrowAmount: 0,
	}
	if err := SaveProjectSettings(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

// ApplyProjectSettings processes a result returned by a background dialog goroutine.
func (appState *AppState) ApplyProjectSettings(settings *ProjectSettings) {

	appState.CurrentlyLoadingProject = false

	if settings == nil {
		// User canceled the operation
		return
	}

	if settings.Err != nil {
		log.Printf("Failed to open project: %v", settings.Err)
		if appState.Settings == nil {
			// No project was loaded before, so surface the error on the create/open project screen.
			appState.Settings = settings
		} else {
			// A project is already loaded; keep it and just notify about the failed attempt.
			appState.Toasts.ShowType(time.Now(), giotoast.TypeError, settings.Err.Error(), 5*time.Second)
		}
		return
	}

	// Save the old image before loading the new project
	if appState.ActiveImage != nil {
		appState.SaveCurrentMask(false, true)
		appState.ActiveImage = nil
	}

	// Apply the new settings
	appState.Settings = settings
	appState.EnableAutoFloodFill.Value = settings.EnableAutoFloodFill
	appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(settings.ShrinkGrowAmount)
	appState.SmoothAmount.Value = settings.SmoothAmount

	// Rebuild editors
	appState.RebuildEditors()

	// List all image files in the input directory
	imageFiles, err := io_utils.ListFiles(settings.InputDirectory)
	if err != nil || len(imageFiles) == 0 {
		log.Printf("Could not list input directory %q: %v", settings.InputDirectory, err)
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
