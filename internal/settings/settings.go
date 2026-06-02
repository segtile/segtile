package settings

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

const ConfigFileName = "settings.yaml"

// SuperpixelSettings stores the algorithm name and its parameters.
type SuperpixelSettings struct {
	Algorithm  string             `yaml:"algorithm"`
	Parameters map[string]float64 `yaml:"parameters"`
}

// ProjectSettings is the top-level structure persisted to disk.
type ProjectSettings struct {
	InputDirectory string             `yaml:"input_directory"`
	Superpixel     SuperpixelSettings `yaml:"superpixel"`
}

// Load reads and unmarshals a config file from the given path.
func Load(path string) (*ProjectSettings, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var cfg ProjectSettings
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
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
