package config

import (
	"os"
	"path/filepath"

	"github.com/goccy/go-yaml"
)

const ConfigFileName = "config.yaml"

// SuperpixelConfig stores the algorithm name and its parameters.
type SuperpixelConfig struct {
	Algorithm  string             `yaml:"algorithm"`
	Parameters map[string]float64 `yaml:"parameters"`
}

// ProjectConfig is the top-level structure persisted to disk.
type ProjectConfig struct {
	InputDirectory string           `yaml:"input_directory"`
	Superpixel     SuperpixelConfig `yaml:"superpixel"`
}

// Load reads and unmarshals a config file from the given path.
func Load(path string) (*ProjectConfig, error) {
	data, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return nil, err
	}
	var cfg ProjectConfig
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// Save marshals cfg and writes it to outputDir/config.yaml,
// creating the directory if it does not exist.
func Save(cfg *ProjectConfig, outputDir string) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(outputDir, 0750); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outputDir, ConfigFileName), data, 0600)
}

// InitNew creates a new ProjectConfig with the given directories and superpixel
// settings, saves it to disk and returns the result.
func InitNew(outputDir, inputDir string, spCfg SuperpixelConfig) (*ProjectConfig, error) {
	cfg := &ProjectConfig{
		InputDirectory: inputDir,
		Superpixel:     spCfg,
	}
	if err := Save(cfg, outputDir); err != nil {
		return nil, err
	}
	return cfg, nil
}
