package app_config

import (
	"testing"

	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"
)

func TestLoadOkProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_ok.yaml"
	result := Load(settingsPath)

	// This should not contain an error

	if result.Err != nil {
		t.Fatal(result.Err)
	}
	if result.Settings == nil {
		t.Fatal("Settings is nil")
	}

}

func TestLoadUnknownAlgorithmProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_bad_algorithm.yaml"
	result := Load(settingsPath)

	// Expect an error
	if result.Err == nil {
		t.Fatal("Expected an error due to unknown algorithm, but got none")
	}

}

func TestBadInputDirectoryProjectSettings(t *testing.T) {

	settingsPath := "testdata/settings_bad_in_dir.yaml"
	result := Load(settingsPath)

	// Expect an error
	if result.Err == nil {
		t.Fatal("Expected an error due to bad input directory, but got none")
	}

}

func TestNonExistentSettingsYaml(t *testing.T) {

	settingsPath := "testdata/settings_does_not_exist.yaml"
	result := Load(settingsPath)

	// Expect an error
	if result.Err == nil {
		t.Fatal("Expected an error due to non-existent settings.yaml, but got none")
	}

}
