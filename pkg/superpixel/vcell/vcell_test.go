package vcell

import (
	"image"
	"image/color"
	"testing"

	"github.com/struffel/annotation-tool/pkg/superpixel"
)

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var c color.RGBA
			if x < width/2 && y < height/2 {
				c = color.RGBA{R: 255, G: 0, B: 0, A: 255}
			} else if x >= width/2 && y < height/2 {
				c = color.RGBA{R: 0, G: 255, B: 0, A: 255}
			} else if x < width/2 && y >= height/2 {
				c = color.RGBA{R: 0, G: 0, B: 255, A: 255}
			} else {
				c = color.RGBA{R: 255, G: 255, B: 0, A: 255}
			}
			img.Set(x, y, c)
		}
	}
	return img
}

func TestVCells_Registered(t *testing.T) {
	title, desc, params, err := superpixel.GetAlgorithmInfo("vcell")
	if err != nil {
		t.Fatalf("vcell algorithm not registered: %v", err)
	}
	if title != "VCells" {
		t.Errorf("expected title 'VCells', got '%s'", title)
	}
	if desc == "" {
		t.Errorf("expected non-empty description")
	}
	if len(params) == 0 {
		t.Errorf("expected registered parameters")
	}
}

func TestVCells_ParameterValidation(t *testing.T) {
	img := createTestImage(64, 64)

	// Nil image
	if _, err := Run(nil, 10, 10.0); err == nil {
		t.Errorf("expected error for nil image")
	}

	// Count < 2
	if _, err := Run(img, 1, 10.0); err == nil {
		t.Errorf("expected error for count < 2")
	}

	// Negative compactness
	if _, err := Run(img, 10, -1.0); err == nil {
		t.Errorf("expected error for negative compactness")
	}
}

func TestVCells_Run(t *testing.T) {
	img := createTestImage(64, 64)

	result, err := Run(img, 16, 10.0)
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if result.LabelMap == nil {
		t.Fatal("LabelMap is nil")
	}

	bounds := result.LabelMap.Bounds()
	if bounds.Dx() != 64 || bounds.Dy() != 64 {
		t.Errorf("unexpected bounds: %v", bounds)
	}

	if result.SpCount < 2 {
		t.Errorf("expected SpCount >= 2, got %d", result.SpCount)
	}

	// Check that all labels in LabelMap are in range [0, SpCount-1]
	seen := make(map[uint16]bool)
	for y := 0; y < 64; y++ {
		for x := 0; x < 64; x++ {
			val := result.LabelMap.Gray16At(x, y).Y
			if int(val) >= result.SpCount {
				t.Errorf("label %d out of range [0, %d)", val, result.SpCount)
			}
			seen[val] = true
		}
	}

	if len(seen) != result.SpCount {
		t.Errorf("actual unique label count %d != SpCount %d", len(seen), result.SpCount)
	}
}

func TestVCells_AutoLambda(t *testing.T) {
	img := createTestImage(64, 64)

	// compactness = 0 triggers automatic lambda calculation
	result, err := Run(img, 16, 0.0)
	if err != nil {
		t.Fatalf("Run with auto-lambda failed: %v", err)
	}

	if result.SpCount < 2 {
		t.Errorf("expected SpCount >= 2, got %d", result.SpCount)
	}
}
