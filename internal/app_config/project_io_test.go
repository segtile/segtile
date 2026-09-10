package app_config

import (
	"testing"

	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"
)

func TestLoadOkProjectSettings(t *testing.T) {

	settingsPath := "testdata/proj_ok/settings.yaml"
	result := Load(settingsPath)

	// This should not contain an error

	if result.Err != nil {
		t.Fatalf("Failed to load project settings: %v", result.Err)
	}
	if result.Settings == nil {
		t.Fatal("Expected non-nil project settings")
	}

}

func TestLoadUnknownAlgorithmProjectSettings(t *testing.T) {

	settingsPath := "testdata/proj_bad_algorithm/settings.yaml"
	result := Load(settingsPath)

	// Expect an error
	if result.Err == nil {
		t.Fatal("Expected an error due to unknown algorithm, but got none")
	}

}

func TestBadInputDirectoryProjectSettings(t *testing.T) {

	settingsPath := "testdata/proj_bad_in_dir/settings.yaml"
	result := Load(settingsPath)

	// Expect an error
	if result.Err == nil {
		t.Fatal("Expected an error due to bad input directory, but got none")
	}

}

func TestNonExistentSettingsYaml(t *testing.T) {

	settingsPath := "testdata/proj_does_not_exist/settings.yaml"
	result := Load(settingsPath)

	// Expect an error
	if result.Err == nil {
		t.Fatal("Expected an error due to non-existent settings.yaml, but got none")
	}

}
