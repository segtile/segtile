package core

import (
	"encoding/json"
	"errors"
	"image"
	"math"
	"os"
	"os/user"
	"path/filepath"
	"testing"

	"gioui.org/f32"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

func TestApplyManualFixAt_ClipsBrushToImageBounds(t *testing.T) {
	state := &AppState{
		ActiveImage:   image.NewNRGBA(image.Rect(0, 0, 4, 4)),
		SelectionMask: superpixel_map.NewMask(4, 4),
		UiState:       UiState{ManualFixBrushRadius: 2},
	}

	state.ApplyManualFixAt(0, 0, true)

	selected := 0
	for _, value := range state.SelectionMask.Mask {
		if value {
			selected++
		}
	}
	if selected != 6 {
		t.Fatalf("selected pixels = %d, want 6", selected)
	}
}

func TestRefreshHoverPreviewRejectsOutOfBoundsCoordinates(t *testing.T) {
	for _, tc := range []struct {
		name string
		pos  f32.Point
	}{
		{name: "positive overflow", pos: f32.Pt(1e30, 1e30)},
		{name: "NaN", pos: f32.Pt(float32(math.NaN()), 0)},
		{name: "infinity", pos: f32.Pt(float32(math.Inf(1)), 0)},
	} {
		t.Run(tc.name, func(t *testing.T) {
			state := &AppState{
				ActiveSp: superpixel.SpResult{LabelMap: superpixel_map.NewLabelMap(2, 2)},
				UiState: UiState{
					InImageBounds: true,
					HoverCached:   true,
				},
			}
			state.LastHoverImagePosition = tc.pos

			state.RefreshHoverPreview()

			if state.InImageBounds {
				t.Fatal("expected invalid hover position to be marked out of bounds")
			}
			if state.HoverCached {
				t.Fatal("expected invalid hover position to invalidate the cached outline")
			}
		})
	}
}

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

func TestFindProjectSettingsFilePrefersMetadataSettings(t *testing.T) {
	projectDir := t.TempDir()
	metadataDir := filepath.Join(projectDir, ProjectMetadataDirectoryName)
	if err := os.MkdirAll(metadataDir, 0750); err != nil {
		t.Fatal(err)
	}
	preferredPath := filepath.Join(metadataDir, DefaultConfigFileName)
	for _, path := range []string{
		preferredPath,
		filepath.Join(projectDir, "other.yaml"),
		filepath.Join(projectDir, "config", "nested.yml"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	foundPath, err := findProjectSettingsFile(projectDir)
	if err != nil {
		t.Fatal(err)
	}
	if foundPath != preferredPath {
		t.Fatalf("settings path = %q, want %q", foundPath, preferredPath)
	}
}

func TestFindProjectSettingsFileRequiresUniqueFallback(t *testing.T) {
	projectDir := t.TempDir()
	for _, path := range []string{
		filepath.Join(projectDir, "first.yaml"),
		filepath.Join(projectDir, "nested", "second.yml"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0750); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(""), 0600); err != nil {
			t.Fatal(err)
		}
	}

	if _, err := findProjectSettingsFile(projectDir); err == nil {
		t.Fatal("expected multiple YAML files to be rejected")
	}
}

func TestFindProjectSettingsFileStopsAfterTwoDirectoryLevels(t *testing.T) {
	projectDir := t.TempDir()
	deepSettingsPath := filepath.Join(projectDir, "one", "two", "three", "settings.yaml")
	if err := os.MkdirAll(filepath.Dir(deepSettingsPath), 0750); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(deepSettingsPath, []byte(""), 0600); err != nil {
		t.Fatal(err)
	}

	if _, err := findProjectSettingsFile(projectDir); err == nil {
		t.Fatal("expected YAML files deeper than two levels to be ignored")
	}
}

func TestLoadProjectFolderMigratesSettingsToMetadataDirectory(t *testing.T) {
	projectDir := t.TempDir()
	inputDir := filepath.Join(projectDir, "input")
	if err := os.MkdirAll(inputDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := SavePngImage(filepath.Join(inputDir, "image.png"), image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}

	settings := &ProjectSettings{
		SettingsFile:   filepath.Join(projectDir, "legacy.yaml"),
		InputDirectory: inputDir,
		Superpixel:     SuperpixelSettings{SpAlgorithmId: "slic"},
	}
	if err := SaveProjectSettings(settings); err != nil {
		t.Fatal(err)
	}

	loaded := LoadProjectSettings(projectDir)
	if loaded.Err != nil {
		t.Fatal(loaded.Err)
	}
	wantSettingsPath := filepath.Join(projectDir, ProjectMetadataDirectoryName, DefaultConfigFileName)
	if loaded.SettingsFile != wantSettingsPath {
		t.Fatalf("settings path = %q, want %q", loaded.SettingsFile, wantSettingsPath)
	}
	if _, err := os.Stat(wantSettingsPath); err != nil {
		t.Fatalf("metadata settings file was not created: %v", err)
	}
}

func TestLoadSavedImageStateIgnoresLegacyRootMask(t *testing.T) {
	outputDir := t.TempDir()
	imagePath := filepath.Join(outputDir, "input.png")
	state := &AppState{
		Settings:   &ProjectSettings{SettingsFile: filepath.Join(outputDir, ProjectMetadataDirectoryName, DefaultConfigFileName)},
		ImageFiles: []string{imagePath},
	}
	if err := SavePngImage(state.maskPathForIndex(0), image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}

	_, _, found, err := state.loadSavedImageState(0)
	if err != nil {
		t.Fatal(err)
	}
	if found {
		t.Fatal("legacy root mask should not be restored without .meta selection and JSON files")
	}
}

func TestSaveAndRestoreImageMetadataAndSelection(t *testing.T) {
	outputDir := t.TempDir()
	inputDir := filepath.Join(outputDir, "input")
	if err := os.MkdirAll(inputDir, 0750); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(inputDir, "image.png")
	if err := SavePngImage(imagePath, image.NewNRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}
	secondImagePath := filepath.Join(inputDir, "second.png")
	if err := SavePngImage(secondImagePath, image.NewNRGBA(image.Rect(0, 0, 16, 16))); err != nil {
		t.Fatal(err)
	}

	parameters, err := superpixel.GetDefaultParameterValues("slic")
	if err != nil {
		t.Fatal(err)
	}
	projectSettings, err := NewProjectSettings(outputDir, inputDir, SuperpixelSettings{
		SpAlgorithmId:         "slic",
		SpAlgorithmParameters: parameters,
	})
	if err != nil {
		t.Fatal(err)
	}

	saveState := &AppState{
		Settings:      projectSettings,
		ImageFiles:    []string{imagePath},
		ActiveImage:   image.NewNRGBA(image.Rect(0, 0, 16, 16)),
		SelectionMask: superpixel_map.NewMask(16, 16),
		ActiveSp:      superpixel.NewSpResult(superpixel_map.NewLabelMap(16, 16)),
		UiState: UiState{
			ShrinkGrowAmount:        widget.Float{Value: 1},
			SmoothAmount:            widget.Float{Value: 0.25},
			SmoothSelectionOutlines: true,
		},
	}
	saveState.SelectionMask.SetValueAt(8, 8, true)
	saveState.Settings.Superpixel.SpAlgorithmParameters["count"] = 2
	saveState.Settings.ShrinkGrowAmount = 1
	saveState.Settings.SmoothAmount = 0.25
	saveState.SaveCurrentMask(true, false)

	finalPath := saveState.maskPathForIndex(0)
	selectionPath := saveState.selectionMaskPathForIndex(0)
	metadataPath := saveState.imageMetadataPathForIndex(0)
	finalMask := superpixel_map.NewMaskFromImage(LoadImage(finalPath))
	savedSelection := superpixel_map.NewMaskFromImage(LoadImage(selectionPath))
	if finalMask.IsEmpty() || savedSelection.IsEmpty() {
		t.Fatal("expected both processed and unprocessed masks to be saved")
	}
	if countMaskPixels(finalMask) <= countMaskPixels(savedSelection) {
		t.Fatal("expected processed mask to include the configured grow operation")
	}
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var savedMetadata ImageMetadata
	if err := json.Unmarshal(metadataBytes, &savedMetadata); err != nil {
		t.Fatal(err)
	}
	if savedMetadata.Superpixel.SpAlgorithmParameters["count"] != 2 ||
		savedMetadata.ShrinkGrowAmount != 1 ||
		savedMetadata.SmoothAmount != 0.25 {
		t.Fatalf("unexpected saved metadata: %+v", savedMetadata)
	}
	currentUser, err := user.Current()
	if err != nil {
		t.Fatal(err)
	}
	if savedMetadata.Annotator != currentUser.Username {
		t.Fatalf("annotator = %q, want current OS username %q", savedMetadata.Annotator, currentUser.Username)
	}

	loadedProject := LoadProjectSettings(outputDir)
	if loadedProject.Err != nil {
		t.Fatal(loadedProject.Err)
	}
	loadedState := &AppState{
		Settings:                cloneProjectSettings(loadedProject),
		ProjectSettingsDefaults: cloneProjectSettings(loadedProject),
		ImageFiles:              []string{imagePath, secondImagePath},
		UiState: UiState{
			ShrinkGrowAmount: widget.Float{Value: 0.5},
			SmoothAmount:     widget.Float{Value: 0},
		},
	}
	if err := loadedState.loadImageAtIndex(0); err != nil {
		t.Fatal(err)
	}
	if countMaskPixels(loadedState.SelectionMask) != 1 || !loadedState.SelectionMask.PositiveAt(8, 8) {
		t.Fatal("restored selection did not match the saved unprocessed selection")
	}
	if loadedState.Settings.Superpixel.SpAlgorithmParameters["count"] != 2 ||
		loadedState.Settings.ShrinkGrowAmount != 1 ||
		loadedState.Settings.SmoothAmount != 0.25 {
		t.Fatalf("image settings were not restored: %+v", loadedState.Settings)
	}
	loadedState.SelectionMask.SetValueAt(0, 0, true)
	if err := loadedState.RevertToSavedMask(); err != nil {
		t.Fatal(err)
	}
	if countMaskPixels(loadedState.SelectionMask) != 1 || !loadedState.SelectionMask.PositiveAt(8, 8) {
		t.Fatal("reverting did not restore the unprocessed mask from disk")
	}
	loadedState.Settings.Superpixel.SpAlgorithmParameters["count"] = loadedProject.Superpixel.SpAlgorithmParameters["count"]
	loadedState.Settings.ShrinkGrowAmount = loadedProject.ShrinkGrowAmount
	loadedState.Settings.SmoothAmount = loadedProject.SmoothAmount
	loadedState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(loadedProject.ShrinkGrowAmount)
	loadedState.SmoothAmount.Value = loadedProject.SmoothAmount
	if err := loadedState.RevertToSavedMask(); err != nil {
		t.Fatal(err)
	}
	if loadedState.Settings.Superpixel.SpAlgorithmParameters["count"] != 2 ||
		loadedState.Settings.ShrinkGrowAmount != 1 ||
		loadedState.Settings.SmoothAmount != 0.25 {
		t.Fatal("reverting did not restore saved image settings")
	}
	if err := loadedState.loadImageAtIndex(1); err != nil {
		t.Fatal(err)
	}
	if !loadedState.SelectionMask.IsEmpty() {
		t.Fatal("image without metadata should start with a blank selection")
	}
	if loadedState.Settings.Superpixel.SpAlgorithmParameters["count"] != loadedProject.Superpixel.SpAlgorithmParameters["count"] ||
		loadedState.Settings.ShrinkGrowAmount != loadedProject.ShrinkGrowAmount ||
		loadedState.Settings.SmoothAmount != loadedProject.SmoothAmount {
		t.Fatalf("image without metadata did not use project defaults: %+v", loadedState.Settings)
	}
}

func countMaskPixels(mask *superpixel_map.Mask) int {
	count := 0
	for _, selected := range mask.Mask {
		if selected {
			count++
		}
	}
	return count
}
