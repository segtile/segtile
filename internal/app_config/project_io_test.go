package app_config

import (
	"testing"

	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"
)

func TestLoadOkProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_ok.yaml"
	settings := LoadProjectSettings(settingsPath)

	// This should not contain an error
	if settings == nil {
		t.Fatal("Settings is nil")
	}
	if settings.Err != nil {
		t.Fatal(settings.Err)
	}
}

func TestLoadUnknownAlgorithmProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_bad_algorithm.yaml"
	settings := LoadProjectSettings(settingsPath)

	// Expect an error
	if settings.Err == nil {
		t.Fatal("Expected an error due to unknown algorithm, but got none")
	}

}

func TestBadInputDirectoryProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_bad_in_dir.yaml"
	settings := LoadProjectSettings(settingsPath)

	// Expect an error
	if settings.Err == nil {
		t.Fatal("Expected an error due to bad input directory, but got none")
	}

}

func TestNonExistentSettingsYaml(t *testing.T) {

	settingsPath := "testdata/settings_does_not_exist.yaml"
	settings := LoadProjectSettings(settingsPath)

	// Expect an error
	if settings.Err == nil {
		t.Fatal("Expected an error due to non-existent settings.yaml, but got none")
	}

}
