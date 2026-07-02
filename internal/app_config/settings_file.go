package app_config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

const ConfigFileName = "settings.yaml"

// ProjectLoadingResult carries the outcome of a background project-load dialog.
type ProjectLoadingResult struct {
	Settings *ProjectSettings
	Err      error
}

// Load reads and unmarshals a config file from the given path.
func Load(path string) (*ProjectSettings, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var settings ProjectSettings
	if err := yaml.Unmarshal(data, &settings); err != nil {
		return nil, err
	}

	// Set the output directory based on the location of the settings file
	settings.OutputDirectory = filepath.Dir(path)

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

// New creates a new ProjectSettings with the given directories and superpixel
// settings, saves it to disk and returns the result.
func New(outputDir, inputDir string, spSettings SuperpixelSettings) (*ProjectSettings, error) {
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
