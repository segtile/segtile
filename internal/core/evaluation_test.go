package core

import (
	"image"
	"os"
	"path/filepath"
	"testing"

	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/evaluation"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

// TestEvaluationMode verifies the core guarantees of evaluation mode:
// annotations start empty even though masks exist on disk (they are ground
// truth), nothing is ever written, the dice score is tracked, and every second
// image is locked to manual fixes.
func TestEvaluationMode(t *testing.T) {
	outputDir := t.TempDir()
	inputDir := filepath.Join(outputDir, "input")
	if err := os.MkdirAll(inputDir, 0750); err != nil {
		t.Fatal(err)
	}

	firstPath := filepath.Join(inputDir, "a_first.png")
	secondPath := filepath.Join(inputDir, "b_second.png")
	for _, p := range []string{firstPath, secondPath} {
		if err := SavePngImage(p, image.NewNRGBA(image.Rect(0, 0, 16, 16))); err != nil {
			t.Fatal(err)
		}
	}

	parameters, err := superpixel.GetDefaultParameterValues("slic")
	if err != nil {
		t.Fatal(err)
	}
	settings, err := NewProjectSettings(outputDir, inputDir, &SuperpixelSettings{
		SpAlgorithmId:         "slic",
		SpAlgorithmParameters: parameters,
	})
	if err != nil {
		t.Fatal(err)
	}

	state := &AppState{
		ActiveSettings: settings,
		ImageFiles:     []string{firstPath, secondPath},
		Eval:           evaluation.NewState(0.95),
		UiState: UiState{
			// Neutral shrink/grow so the processed mask equals the raw mask.
			ShrinkGrowAmount:        widget.Float{Value: 0.5},
			SmoothSelectionOutlines: true,
			ManualFixBrushRadius:    4,
		},
	}

	// Ground truth masks live at the output root, where final masks are stored.
	gt := superpixel_map.NewMask(16, 16)
	gt.SetValueAt(3, 3, true)
	gt.SetValueAt(4, 4, true)
	for i := 0; i < 2; i++ {
		if err := SavePngImage(state.maskPathForIndex(i), gt.ToImageGray16()); err != nil {
			t.Fatal(err)
		}
	}

	if err := state.loadImageAtIndex(0); err != nil {
		t.Fatal(err)
	}

	// The selection starts empty even though a mask file exists on disk.
	if !state.SelectionMaskRaw.IsEmpty() {
		t.Fatal("evaluation mode must start with an empty selection")
	}
	// The ground truth must not be imported as saved state into .meta.
	if _, err := os.Stat(state.selectionMaskPathForIndex(0)); !os.IsNotExist(err) {
		t.Fatal("evaluation mode must not write .meta files")
	}
	if state.ActiveImageMetadata != nil {
		t.Fatal("evaluation mode must not expose saved image metadata")
	}
	// Even indices allow superpixels.
	if state.EvalManualOnly() {
		t.Fatal("index 0 must allow superpixel annotation")
	}

	// A perfect annotation reaches a dice score of 1 and passes the threshold.
	state.SelectionMaskRaw.SetValueAt(3, 3, true)
	state.SelectionMaskRaw.SetValueAt(4, 4, true)
	state.ReprocessSelection()
	if state.Eval.CurrentDice() != 1 {
		t.Fatalf("dice = %v, want 1", state.Eval.CurrentDice())
	}
	if !state.Eval.AboveThreshold() {
		t.Fatal("dice of 1 must be above the 0.95 threshold")
	}

	// Saving is a no-op in evaluation mode: no .meta, GT file untouched.
	state.SaveCurrentMask(true)
	if _, err := os.Stat(state.selectionMaskPathForIndex(0)); !os.IsNotExist(err) {
		t.Fatal("SaveCurrentMask must not write anything in evaluation mode")
	}
	if reloaded := superpixel_map.NewMaskFromImage(LoadImage(state.maskPathForIndex(0))); countMaskPixels(reloaded) != 2 {
		t.Fatal("ground truth mask on disk must stay untouched")
	}

	// Navigating to the second image locks manual fix mode and resets the score.
	if err := state.loadImageAtIndex(1); err != nil {
		t.Fatal(err)
	}
	if !state.EvalManualOnly() {
		t.Fatal("index 1 must be manual-only")
	}
	if !state.ManualFixMode {
		t.Fatal("manual fix mode must be forced on for manual-only images")
	}
	state.SetManualFixMode(false)
	if !state.ManualFixMode {
		t.Fatal("manual-only images must not be able to leave manual fix mode")
	}
	if state.Eval.CurrentDice() != 0 {
		t.Fatalf("dice must reset on navigation, got %v", state.Eval.CurrentDice())
	}
}
