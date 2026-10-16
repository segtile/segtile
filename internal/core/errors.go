package core

import "fmt"

// Settings file could not be read by OS
type SettingsFileReadError struct {
	path string
	err  error
}

func (e SettingsFileReadError) Error() string {
	return fmt.Sprintf("Error reading settings file %s: %v", e.path, e.err)
}

// YAML could not be unmarshaled
type SettingsFileYamlParseError struct {
	err error
}

func (e SettingsFileYamlParseError) Error() string {
	return fmt.Sprintf("Error parsing project settings file: %v", e.err)
}

// Input directory could not be parsed
type SettingsInputDirectoryError struct {
	path string
	err  error
}

func (e SettingsInputDirectoryError) Error() string {
	return fmt.Sprintf("Error parsing input directory %s: %v", e.path, e.err)
}

// Input directory could be parsed, but contains no images
type SettingsNoImagesInInputDirectoryError struct {
	path string
}

func (e SettingsNoImagesInInputDirectoryError) Error() string {
	return fmt.Sprintf("Input directory %s contains no images", e.path)
}

// Save error
type SettingsSaveError struct {
	path string
	err  error
}

func (e SettingsSaveError) Error() string {
	return fmt.Sprintf("Error saving settings into %s: %v", e.path, e.err)
}
