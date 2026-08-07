package app_config

import (
	"log"
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/ncruces/zenity"
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

// ProjectSettings stores all settings of a project that need to be persisted to disk.
type ProjectSettings struct {
	OutputDirectory     string             `yaml:"-"` // The output directory is not persisted in the yaml settings file, it is derived from the location of the settings file itself.
	InputDirectory      string             `yaml:"input_directory"`
	Superpixel          SuperpixelSettings `yaml:"superpixel"`
	EnableAutoFloodFill bool               `yaml:"auto_flood_fill"`
}

// Load reads and unmarshals a config file from the given path.
func Load(path string) (*ProjectSettings, error) {

	// Read the file
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}

	// Parse yaml
	var settings ProjectSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	// Set the output directory based on the location of the settings file
	settings.OutputDirectory = filepath.Dir(path)

	if err := settings.Superpixel.normalizeSuperpixelSettings(); err != nil {
		return nil, err
	}

	return &settings, nil
}

// Save marshals cfg and writes it to outputDir/settings.yaml,
// creating the directory if it does not exist.
func Save(settings *ProjectSettings) error {
	if settings.OutputDirectory == "" {
		return os.ErrInvalid
	}
	data, err := yaml.Marshal(settings)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(settings.OutputDirectory, 0750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(settings.OutputDirectory, ConfigFileName), data, 0600)
}

// NewProjectSettings creates a new ProjectSettings with the given directories and superpixel
// settings, saves it to disk and returns the result.
func NewProjectSettings(outputDir, inputDir string, spSettings SuperpixelSettings) (*ProjectSettings, error) {
	cfg := &ProjectSettings{
		InputDirectory:  inputDir,
		OutputDirectory: outputDir,
		Superpixel:      spSettings,
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
		return
	}
	if loadingResult.Err != nil {
		log.Printf("Failed to open project: %v", loadingResult.Err)
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

	// Rebuild editors
	appState.RebuildEditors()

	// List all image files in the input directory
	imageFiles, err := io_utils.ListFiles(loadingResult.Settings.InputDirectory)
	if err != nil || len(imageFiles) == 0 {
		log.Printf("Could not list input directory %q: %v", loadingResult.Settings.InputDirectory, err)
		return
	}

	appState.ImageFiles = imageFiles
	appState.ImageFilesIndex = 0
	appState.loadImageAtIndex(0)
}
