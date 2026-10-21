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
		ActiveImage:      image.NewNRGBA(image.Rect(0, 0, 4, 4)),
		SelectionMaskRaw: superpixel_map.NewMask(4, 4),
		UiState:          UiState{ManualFixBrushRadius: 2},
	}

	state.ApplyManualFixAt(0, 0, true)

	selected := 0
	for _, value := range state.SelectionMaskRaw.Mask {
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

	settingsPath := "testdata/project/settings_ok.yaml"
	settings := LoadProjectSettings(settingsPath)

	if settings == nil {
		t.Fatal("Settings is nil")
	}
	if settings.Err != nil {
		t.Fatalf("%v", settings.Err)
	}
}

func TestLoadUnknownAlgorithmProjectSettings(t *testing.T) {

	settingsPath := "testdata/project/settings_bad_algorithm.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *superpixel.UnknownSpAlgorithmError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected UnknownSpAlgorithmError, got %v", settings.Err)
	}

}

func TestBadInputDirectoryProjectSettings(t *testing.T) {

	settingsPath := "testdata/project/settings_bad_input_directory.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *SettingsInputDirectoryError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected SettingsInputDirectoryError, got %v", settings.Err)
	}

}

func TestNonExistentSettingsYaml(t *testing.T) {

	settingsPath := "testdata/project/settings_does_not_exist.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *SettingsFileReadError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected SettingsFileReadError, got %v", settings.Err)
	}

}

func TestNoImagesInInputDirectory(t *testing.T) {

	settingsPath := "testdata/project/settings_no_images_in_directory.yaml"
	settings := LoadProjectSettings(settingsPath)

	var targetErr *SettingsNoImagesInInputDirectoryError
	if !errors.As(settings.Err, &targetErr) {
		t.Fatalf("Expected SettingsNoImagesInInputDirectoryError, got %v", settings.Err)
	}

}

func TestLoadProjectFromDirectoryUsesDefaultSettingsFile(t *testing.T) {
	projectDir := t.TempDir()
	inputDir := filepath.Join(projectDir, "input")
	if err := os.MkdirAll(inputDir, 0750); err != nil {
		t.Fatal(err)
	}
	if err := SavePngImage(filepath.Join(inputDir, "image.png"), image.NewNRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}

	settings, err := NewProjectSettings(projectDir, inputDir, &SuperpixelSettings{SpAlgorithmId: "slic"})
	if err != nil {
		t.Fatal(err)
	}
	wantSettingsPath := filepath.Join(projectDir, DefaultConfigFileName)
	if settings.SettingsFile != wantSettingsPath {
		t.Fatalf("settings path = %q, want %q", settings.SettingsFile, wantSettingsPath)
	}

	loaded := LoadProjectSettings(projectDir)
	if loaded.Err != nil {
		t.Fatal(loaded.Err)
	}
	if loaded.SettingsFile != wantSettingsPath {
		t.Fatalf("loaded settings path = %q, want %q", loaded.SettingsFile, wantSettingsPath)
	}
}

func TestLoadSavedImageStateImportsLegacyRootMask(t *testing.T) {
	outputDir := t.TempDir()
	imagePath := filepath.Join(outputDir, "input.png")
	parameters, err := superpixel.GetDefaultParameterValues("slic")
	if err != nil {
		t.Fatal(err)
	}
	state := &AppState{
		ActiveSettings: &ProjectSettings{
			SettingsFile: filepath.Join(outputDir, DefaultConfigFileName),
			Superpixel:   &SuperpixelSettings{SpAlgorithmId: "slic", SpAlgorithmParameters: parameters},
		},
		ImageFiles: []string{imagePath},
	}

	// A final mask exists in the output directory, but nothing in .meta.
	legacyMask := superpixel_map.NewMask(2, 2)
	legacyMask.SetValueAt(1, 1, true)
	if err := SavePngImage(state.maskPathForIndex(0), legacyMask.ToImageGray16()); err != nil {
		t.Fatal(err)
	}

	metadata, selection, found, err := state.loadSavedImageState(0)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected legacy root mask to be imported")
	}
	if !selection.PositiveAt(1, 1) || countMaskPixels(selection) != 1 {
		t.Fatal("imported selection does not match the legacy mask")
	}
	if metadata.SmoothAmount != 0 || metadata.ShrinkGrowAmount != 0 {
		t.Fatalf("imported metadata should have zero grow/shrink/smooth, got %+v", metadata)
	}
	if metadata.Superpixel.SpAlgorithmId != "slic" {
		t.Fatalf("imported metadata should keep current SP settings, got %+v", metadata.Superpixel)
	}

	// The import must have materialized the .meta files so future loads are regular.
	if _, err := os.Stat(state.selectionMaskPathForIndex(0)); err != nil {
		t.Fatalf("selection was not copied to .meta: %v", err)
	}
	if _, err := os.Stat(state.imageMetadataPathForIndex(0)); err != nil {
		t.Fatalf("metadata was not written to .meta: %v", err)
	}
}

func TestSetSpAlgorithmPersistsAsProjectDefault(t *testing.T) {
	const dummyAlgorithmId = "test_dummy"
	superpixel.RegisterSpAlgorithm(dummyAlgorithmId, "Test Dummy", "Dummy algorithm for tests",
		func(img image.Image, params map[string]float64) (superpixel.SpResult, error) {
			bounds := img.Bounds()
			return superpixel.NewSpResult(superpixel_map.NewLabelMap(bounds.Dx(), bounds.Dy())), nil
		},
		[]superpixel.SpParameterInfo{{Id: "count", Kind: superpixel.SpParameterKindInt, Title: "Count", Default: 4}},
	)

	outputDir := t.TempDir()
	inputDir := filepath.Join(outputDir, "input")
	if err := os.MkdirAll(inputDir, 0750); err != nil {
		t.Fatal(err)
	}
	imagePath := filepath.Join(inputDir, "image.png")
	if err := SavePngImage(imagePath, image.NewNRGBA(image.Rect(0, 0, 8, 8))); err != nil {
		t.Fatal(err)
	}

	parameters, err := superpixel.GetDefaultParameterValues("slic")
	if err != nil {
		t.Fatal(err)
	}
	projectSettings, err := NewProjectSettings(outputDir, inputDir, &SuperpixelSettings{
		SpAlgorithmId:         "slic",
		SpAlgorithmParameters: parameters,
	})
	if err != nil {
		t.Fatal(err)
	}

	state := &AppState{
		ActiveSettings:   projectSettings,
		DefaultSettings:  projectSettings.Clone(),
		ImageFiles:       []string{imagePath},
		ActiveImage:      image.NewNRGBA(image.Rect(0, 0, 8, 8)),
		SelectionMaskRaw: superpixel_map.NewMask(8, 8),
	}

	if err := state.SetSpAlgorithm(dummyAlgorithmId); err != nil {
		t.Fatal(err)
	}

	// The change must apply to the current image and become the project default.
	if state.ActiveSettings.Superpixel.SpAlgorithmId != dummyAlgorithmId {
		t.Fatalf("active settings algorithm = %q, want %q", state.ActiveSettings.Superpixel.SpAlgorithmId, dummyAlgorithmId)
	}
	if state.DefaultSettings.Superpixel.SpAlgorithmId != dummyAlgorithmId {
		t.Fatalf("project default algorithm = %q, want %q", state.DefaultSettings.Superpixel.SpAlgorithmId, dummyAlgorithmId)
	}

	// The change must be persisted to the settings file on disk.
	reloaded := LoadProjectSettings(projectSettings.SettingsFile)
	if reloaded.Err != nil {
		t.Fatal(reloaded.Err)
	}
	if reloaded.Superpixel.SpAlgorithmId != dummyAlgorithmId {
		t.Fatalf("persisted algorithm = %q, want %q", reloaded.Superpixel.SpAlgorithmId, dummyAlgorithmId)
	}
	if reloaded.Superpixel.SpAlgorithmParameters["count"] != 4 {
		t.Fatalf("persisted parameters = %v, want dummy defaults", reloaded.Superpixel.SpAlgorithmParameters)
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
	projectSettings, err := NewProjectSettings(outputDir, inputDir, &SuperpixelSettings{
		SpAlgorithmId:         "slic",
		SpAlgorithmParameters: parameters,
	})
	if err != nil {
		t.Fatal(err)
	}

	saveState := &AppState{
		ActiveSettings:   projectSettings,
		ImageFiles:       []string{imagePath},
		ActiveImage:      image.NewNRGBA(image.Rect(0, 0, 16, 16)),
		SelectionMaskRaw: superpixel_map.NewMask(16, 16),
		ActiveSp:         superpixel.NewSpResult(superpixel_map.NewLabelMap(16, 16)),
		UiState: UiState{
			ShrinkGrowAmount:        widget.Float{Value: 1},
			SmoothAmount:            widget.Float{Value: 0.25},
			SmoothSelectionOutlines: true,
		},
	}
	saveState.SelectionMaskRaw.SetValueAt(8, 8, true)
	saveState.ActiveSettings.Superpixel.SpAlgorithmParameters["count"] = 2
	saveState.ActiveSettings.ShrinkGrowAmount = 1
	saveState.ActiveSettings.SmoothAmount = 0.25
	saveState.SaveCurrentMask(true)

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
		ActiveSettings:  loadedProject.Clone(),
		DefaultSettings: loadedProject.Clone(),
		ImageFiles:      []string{imagePath, secondImagePath},
		UiState: UiState{
			ShrinkGrowAmount: widget.Float{Value: 0.5},
			SmoothAmount:     widget.Float{Value: 0},
		},
	}
	if err := loadedState.loadImageAtIndex(0); err != nil {
		t.Fatal(err)
	}
	if countMaskPixels(loadedState.SelectionMaskRaw) != 1 || !loadedState.SelectionMaskRaw.PositiveAt(8, 8) {
		t.Fatal("restored selection did not match the saved unprocessed selection")
	}
	if loadedState.ActiveSettings.Superpixel.SpAlgorithmParameters["count"] != 2 ||
		loadedState.ActiveSettings.ShrinkGrowAmount != 1 ||
		loadedState.ActiveSettings.SmoothAmount != 0.25 {
		t.Fatalf("image settings were not restored: %+v", loadedState.ActiveSettings)
	}
	if loadedState.SelectionMaskDiffersVersionOnDisk {
		t.Fatal("restored selection should not be marked as modified")
	}
	loadedState.SelectionMaskRaw.SetValueAt(0, 0, true)
	loadedState.SelectionMaskDiffersVersionOnDisk = true
	if err := loadedState.RevertToSavedMask(); err != nil {
		t.Fatal(err)
	}
	if countMaskPixels(loadedState.SelectionMaskRaw) != 1 || !loadedState.SelectionMaskRaw.PositiveAt(8, 8) {
		t.Fatal("reverting did not restore the unprocessed mask from disk")
	}
	if loadedState.SelectionMaskDiffersVersionOnDisk {
		t.Fatal("reverting should clear the modified flag")
	}
	loadedState.ActiveSettings.Superpixel.SpAlgorithmParameters["count"] = loadedProject.Superpixel.SpAlgorithmParameters["count"]
	loadedState.ActiveSettings.ShrinkGrowAmount = loadedProject.ShrinkGrowAmount
	loadedState.ActiveSettings.SmoothAmount = loadedProject.SmoothAmount
	loadedState.ShrinkGrowAmount.Value = SignedShrinkGrowToSlider(loadedProject.ShrinkGrowAmount)
	loadedState.SmoothAmount.Value = loadedProject.SmoothAmount
	loadedState.SelectionMaskDiffersVersionOnDisk = true
	if err := loadedState.RevertToSavedMask(); err != nil {
		t.Fatal(err)
	}
	if loadedState.ActiveSettings.Superpixel.SpAlgorithmParameters["count"] != 2 ||
		loadedState.ActiveSettings.ShrinkGrowAmount != 1 ||
		loadedState.ActiveSettings.SmoothAmount != 0.25 {
		t.Fatal("reverting did not restore saved image settings")
	}

	// Navigating away from an unmodified restored image must not write anything.
	if err := loadedState.loadImageAtIndex(1); err != nil {
		t.Fatal(err)
	}
	if !loadedState.SelectionMaskRaw.IsEmpty() {
		t.Fatal("image without metadata should start with a blank selection")
	}
	if loadedState.SelectionMaskDiffersVersionOnDisk {
		t.Fatal("blank selection should not be marked as modified")
	}
	if loadedState.ActiveSettings.Superpixel.SpAlgorithmParameters["count"] != loadedProject.Superpixel.SpAlgorithmParameters["count"] ||
		loadedState.ActiveSettings.ShrinkGrowAmount != loadedProject.ShrinkGrowAmount ||
		loadedState.ActiveSettings.SmoothAmount != loadedProject.SmoothAmount {
		t.Fatalf("image without metadata did not use project defaults: %+v", loadedState.ActiveSettings)
	}

	// A user change marks the selection as modified and navigation then saves it.
	loadedState.SetSpSelected(0, true)
	if !loadedState.SelectionMaskDiffersVersionOnDisk {
		t.Fatal("changing the selection should mark it as modified")
	}
	if err := loadedState.loadImageAtIndex(0); err != nil {
		t.Fatal(err)
	}
	if _, _, found, err := loadedState.loadSavedImageState(1); err != nil || !found {
		t.Fatalf("modified image should have been saved on navigation (found=%v, err=%v)", found, err)
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
