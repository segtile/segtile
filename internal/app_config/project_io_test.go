package app_config

import (
	"errors"
	"testing"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"
)

func TestLoadOkProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_ok.yaml"
	settings := LoadProjectSettings(settingsPath)

	if settings == nil {
		t.Fatal("Settings is nil")
	}
	if settings.Err != nil {
		t.Fatalf("%v", settings.Err)
	}
}

func TestLoadUnknownAlgorithmProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_bad_algorithm.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *superpixel.UnknownSpAlgorithmError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected UnknownSpAlgorithmError, got %v", settings.Err)
	}

}

func TestBadInputDirectoryProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_bad_input_directory.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *SettingsInputDirectoryError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected SettingsInputDirectoryError, got %v", settings.Err)
	}

}

func TestNonExistentSettingsYaml(t *testing.T) {

	settingsPath := "testdata/settings_does_not_exist.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *SettingsFileReadError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected SettingsFileReadError, got %v", settings.Err)
	}

}

func TestNoImagesInInputDirectory(t *testing.T) {

	settingsPath := "testdata/settings_no_images_in_directory.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *SettingsNoImagesInInputDirectoryError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected SettingsNoImagesInInputDirectoryError, got %v", settings.Err)
	}

}
