package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

const DefaultConfigFileName = "settings.yaml"
const ProjectMetadataDirectoryName = ".meta"
const defaultSpAlgorithmId = "slic"

// ProjectSettings stores all settings of a project that need to be persisted to disk.
type ProjectSettings struct {
	SettingsFile              string             `yaml:"-"` // Don't store the path to the file in itself.
	InputDirectory            string             `yaml:"input_directory"`
	Superpixel                SuperpixelSettings `yaml:"superpixel"`
	EnableAutoFloodFill       bool               `yaml:"auto_flood_fill"`
	UpdateSelectionOnSpChange bool               `yaml:"update_selection_on_sp_change"`
	ShrinkGrowAmount          float32            `yaml:"shrink_grow_amount"`
	SmoothAmount              float32            `yaml:"smooth_amount"`
	Err                       error              `yaml:"-"` // Don't store the error
}

// SuperpixelSettings stores the algorithm name and its parameters.
type SuperpixelSettings struct {
	SpAlgorithmId         string             `yaml:"algorithm" json:"algorithm"`
	SpAlgorithmParameters map[string]float64 `yaml:"parameters" json:"parameters"`
}

// ImageMetadata stores metadata for an individual image within the project.
type ImageMetadata struct {
	Annotator        string             `json:"annotator"`
	Superpixel       SuperpixelSettings `json:"superpixel"`
	SmoothAmount     float32            `json:"smooth_amount"`
	ShrinkGrowAmount float32            `json:"shrink_grow_amount"`
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
func LoadProjectSettings(settingsYamlPath string) *ProjectSettings {

	// Ensure the path is clean
	settingsYamlPath = filepath.Clean(settingsYamlPath)

	// Check if the given path exists
	stat, err := os.Stat(settingsYamlPath)
	if err != nil {
		return &ProjectSettings{Err: &SettingsFileReadError{path: settingsYamlPath, err: err}}
	}

	// If the path is a directory,
	if stat.IsDir() {
		settingsYamlPath = filepath.Join(settingsYamlPath, DefaultConfigFileName)
	}

	// Then try reading the settings file
	data, err := os.ReadFile(settingsYamlPath)
	if err != nil {
		return &ProjectSettings{Err: &SettingsFileReadError{path: settingsYamlPath, err: err}}
	}

	var settings ProjectSettings
	err = yaml.Unmarshal(data, &settings)
	if err != nil {
		return &ProjectSettings{Err: &SettingsFileYamlParseError{err: err}}
	}

	// Resolve relative input paths against the settings file that was selected.
	settings.SettingsFile = settingsYamlPath

	if !filepath.IsAbs(settings.InputDirectory) {
		settings.InputDirectory = filepath.Join(filepath.Dir(settings.SettingsFile), settings.InputDirectory)
	}
	_, err = os.Stat(settings.InputDirectory)
	if err != nil {
		return &ProjectSettings{Err: &SettingsInputDirectoryError{path: settings.InputDirectory, err: err}}
	}

	// Test if the input directory contains any images
	imageFiles, err := ListFiles(settings.InputDirectory, true)
	if err != nil {
		return &ProjectSettings{Err: &SettingsInputDirectoryError{path: settings.InputDirectory, err: err}}
	}
	if len(imageFiles) == 0 {
		return &ProjectSettings{Err: &SettingsNoImagesInInputDirectoryError{path: settings.InputDirectory}}
	}

	// Normalize the parsed SP settings before returning the final project settings.
	err = settings.Superpixel.normalizeSuperpixelSettings()
	if err != nil {
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

func cloneProjectSettings(settings *ProjectSettings) *ProjectSettings {
	if settings == nil {
		return nil
	}
	cloned := *settings
	cloned.Superpixel = cloneSuperpixelSettings(settings.Superpixel)
	return &cloned
}

func cloneSuperpixelSettings(settings SuperpixelSettings) SuperpixelSettings {
	cloned := settings
	cloned.SpAlgorithmParameters = make(map[string]float64, len(settings.SpAlgorithmParameters))
	for id, value := range settings.SpAlgorithmParameters {
		cloned.SpAlgorithmParameters[id] = value
	}
	return cloned
}

func (appState *AppState) SaveProjectSettings() error {
	if err := SaveProjectSettings(appState.Settings); err != nil {
		return err
	}
	appState.ProjectSettingsDefaults = cloneProjectSettings(appState.Settings)
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

// projectOutputDirectory is the directory containing the settings file; final masks and
// the .meta metadata directory both live there.
func projectOutputDirectory(settings *ProjectSettings) string {
	return filepath.Dir(settings.SettingsFile)
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

	// Save the old image before loading the new project, but only if it was actually changed.
	if appState.ActiveImage != nil {
		if appState.SelectionModified {
			appState.SaveCurrentMask(false, true)
		}
		appState.ActiveImage = nil
	}

	// Apply the new settings
	appState.Settings = settings
	appState.ProjectSettingsDefaults = cloneProjectSettings(settings)
	appState.EnableAutoFloodFill.Value = settings.EnableAutoFloodFill
	appState.UpdateSelectionOnSpChange.Value = settings.UpdateSelectionOnSpChange
	appState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(settings.ShrinkGrowAmount)
	appState.SmoothAmount.Value = settings.SmoothAmount

	// Rebuild editors
	appState.RebuildEditors()

	// List all image files in the input directory
	imageFiles, err := ListFiles(settings.InputDirectory, true)
	if err != nil {
		log.Printf("Could not list input directory %q: %v", settings.InputDirectory, err)
		appState.Toasts.ShowType(time.Now(), giotoast.TypeError, err.Error(), 5*time.Second)
		return
	}

	if len(imageFiles) == 0 {
		err := fmt.Errorf("No image files found in input directory %q", settings.InputDirectory)
		log.Printf("%v", err)
		appState.Toasts.ShowType(time.Now(), giotoast.TypeError, err.Error(), 5*time.Second)
		return
	}

	appState.ImageFiles = imageFiles
	appState.ImageFilesIndex = 0
	appState.loadImageAtIndex(0)
}
