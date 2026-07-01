package app_config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
	"github.com/struffel/annotation-tool/internal/factory/superpixel_factory"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

const ConfigFileName = "settings.yaml"

// SuperpixelSettings stores the algorithm name and its parameters.
// This is used for yaml persistence.
type SuperpixelSettings struct {
	Algorithm  string             `yaml:"algorithm"`
	Parameters map[string]float64 `yaml:"parameters"`
}

// ProjectSettings stores all settings of a project,
// it is used for yaml persistence.
type ProjectSettings struct {
	InputDirectory string             `yaml:"input_directory"`
	Superpixel     SuperpixelSettings `yaml:"superpixel"`
}

// ProjectLoadingResult carries the outcome of a background project-load dialog.
type ProjectLoadingResult struct {
	Settings        *ProjectSettings
	OutputDirectory string
	Err             error
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
	return &settings, nil
}

// Save marshals cfg and writes it to outputDir/settings.yaml,
// creating the directory if it does not exist.
func Save(cfg *ProjectSettings, outputDir string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, ConfigFileName), data, 0600)
}

// InitNew creates a new ProjectSettings with the given directories and superpixel
// settings, saves it to disk and returns the result.
func InitNew(outputDir, inputDir string, spSettings SuperpixelSettings) (*ProjectSettings, error) {
	cfg := &ProjectSettings{
		InputDirectory: inputDir,
		Superpixel:     spSettings,
	}
	if err := Save(cfg, outputDir); err != nil {
		return nil, err
	}
	return cfg, nil
}

// SuperpixelSettingsFromImpl derives a SuperpixelSettings from a live SpImplementation.
// The algorithm name is resolved via the superpixel_factory, keeping all name
// registry logic in one place.
func SuperpixelSettingsFromImpl(impl superpixel.SpImplementation) (SuperpixelSettings, error) {
	name, err := superpixel_factory.GetSuperpixelImplementationName(impl)
	if err != nil {
		return SuperpixelSettings{}, err
	}
	params := make(map[string]float64)
	for _, p := range impl.GetSupportedParameters() {
		v, _ := impl.GetParameter(p)
		params[p.String()] = v
	}
	return SuperpixelSettings{Algorithm: name, Parameters: params}, nil
}
