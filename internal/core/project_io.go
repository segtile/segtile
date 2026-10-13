package core

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/goccy/go-yaml"
	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

const DefaultConfigFileName = "settings.yaml"
const ProjectMetadataDirectoryName = ".meta"
const defaultSpAlgorithmId = "snic"

// ProjectSettings stores all settings of a project that need to be persisted to disk.
type ProjectSettings struct {
	SettingsFile              string             `yaml:"-"` // The path to the settings file itself, not persisted in the yaml.
	InputDirectory            string             `yaml:"input_directory"`
	Superpixel                SuperpixelSettings `yaml:"superpixel"`
	EnableAutoFloodFill       bool               `yaml:"auto_flood_fill"`
	UpdateSelectionOnSpChange bool               `yaml:"update_selection_on_sp_change"`
	ShrinkGrowAmount          float32            `yaml:"shrink_grow_amount"`
	SmoothAmount              float32            `yaml:"smooth_amount"`
	Err                       error              `yaml:"-"` // The error is also not persisted to disk.
}

// SuperpixelSettings stores the algorithm name and its parameters.
// This is used for yaml persistence.
type SuperpixelSettings struct {
	SpAlgorithmId         string             `yaml:"algorithm" json:"algorithm"`
	SpAlgorithmParameters map[string]float64 `yaml:"parameters" json:"parameters"`
}

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
// This can be either a direct path to a settings yaml file or the path to a directory,
// in which case it will look for a settings yaml file within that directory.
func LoadProjectSettings(settingsPath string) *ProjectSettings {

	// Check if the given path exists and whether it is a file or directory
	stat, err := os.Stat(settingsPath)
	if err != nil {
		return &ProjectSettings{Err: &SettingsFileReadError{path: settingsPath, err: err}}
	}

	projectDirectory := ""
	if stat.IsDir() {
		projectDirectory = settingsPath
		settingsPath, err = findProjectSettingsFile(projectDirectory)
		if err != nil {
			return &ProjectSettings{Err: &SettingsFileReadError{path: projectDirectory, err: err}}
		}
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

	// Resolve relative input paths against the settings file that was selected.
	settings.SettingsFile = settingsPath

	// Resolve relative paths before migrating the settings file to .meta so the input path
	// continues to point to the same directory after relocation.
	if !filepath.IsAbs(settings.InputDirectory) {
		if _, err := os.Stat(settings.InputDirectory); err != nil {
			settings.InputDirectory = filepath.Join(filepath.Dir(settings.SettingsFile), settings.InputDirectory)
		}
		settings.InputDirectory, err = filepath.Abs(settings.InputDirectory)
		if err != nil {
			return &ProjectSettings{Err: &SettingsInputDirectoryError{path: settings.InputDirectory, err: err}}
		}
	}
	if _, err := os.Stat(settings.InputDirectory); err != nil {
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

	if err := settings.Superpixel.normalizeSuperpixelSettings(); err != nil {
		return &ProjectSettings{Err: err}
	}

	if projectDirectory != "" {
		settings.SettingsFile = filepath.Join(projectDirectory, ProjectMetadataDirectoryName, DefaultConfigFileName)
		if err := SaveProjectSettings(&settings); err != nil {
			return &ProjectSettings{Err: err}
		}
	}

	return &settings
}

func findProjectSettingsFile(projectDirectory string) (string, error) {
	preferredPath := filepath.Join(projectDirectory, ProjectMetadataDirectoryName, DefaultConfigFileName)
	if info, err := os.Stat(preferredPath); err == nil && !info.IsDir() {
		return preferredPath, nil
	} else if err != nil && !os.IsNotExist(err) {
		return "", err
	}

	var candidates []string
	var scanDirectory func(string, int) error
	scanDirectory = func(directory string, depth int) error {
		entries, err := os.ReadDir(directory)
		if err != nil {
			return err
		}
		for _, entry := range entries {
			entryPath := filepath.Join(directory, entry.Name())
			if entry.IsDir() {
				if depth < 2 {
					if err := scanDirectory(entryPath, depth+1); err != nil {
						return err
					}
				}
				continue
			}
			extension := strings.ToLower(filepath.Ext(entry.Name()))
			if extension == ".yaml" || extension == ".yml" {
				candidates = append(candidates, entryPath)
			}
		}
		return nil
	}
	if err := scanDirectory(projectDirectory, 0); err != nil {
		return "", err
	}
	if len(candidates) == 1 {
		return candidates[0], nil
	}
	if len(candidates) == 0 {
		return "", fmt.Errorf("no YAML settings file found within two directory levels")
	}
	return "", fmt.Errorf("multiple YAML settings files found; expected %s or exactly one YAML file", preferredPath)
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
		SettingsFile:     filepath.Join(outputDir, ProjectMetadataDirectoryName, DefaultConfigFileName),
		InputDirectory:   inputDir,
		Superpixel:       spSettings,
		ShrinkGrowAmount: 0,
	}
	if err := SaveProjectSettings(cfg); err != nil {
		return nil, err
	}
	return cfg, nil
}

func projectOutputDirectory(settings *ProjectSettings) string {
	settingsDirectory := filepath.Dir(settings.SettingsFile)
	if filepath.Base(settingsDirectory) == ProjectMetadataDirectoryName {
		return filepath.Dir(settingsDirectory)
	}
	return settingsDirectory
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
