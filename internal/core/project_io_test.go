package core

import (
	"errors"
	"image"
	"math"
	"testing"

	"gioui.org/f32"
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
