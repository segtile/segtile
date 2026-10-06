package rss

import (
	"image"
	"image/color"
	"math"
	"testing"
)

func TestRunProducesConnectedSuperpixelsAtSeedCount(t *testing.T) {
	img := image.NewNRGBA(image.Rect(0, 0, 32, 24))
	for y := 0; y < img.Bounds().Dy(); y++ {
		for x := 0; x < img.Bounds().Dx(); x++ {
			img.SetNRGBA(x, y, color.NRGBA{R: 96, G: 128, B: 160, A: 255})
		}
	}

	result, err := Run(img, 12, 3)
	if err != nil {
		t.Fatalf("Run returned an error: %v", err)
	}
	if result.SpCount != 12 {
		t.Fatalf("expected 12 superpixels, got %d", result.SpCount)
	}
	if !result.ConnectivityEnforced() {
		t.Fatal("expected the result to enforce connectivity")
	}
	if got := result.LabelMap.Bounds(); got != img.Bounds() {
		t.Fatalf("label map bounds = %v, want %v", got, img.Bounds())
	}
}

func TestRunRejectsInvalidInputs(t *testing.T) {
	tests := []struct {
		name        string
		img         image.Image
		count       int
		compactness float64
	}{
		{name: "nil image", count: 2},
		{name: "small count", img: image.NewNRGBA(image.Rect(0, 0, 4, 4)), count: 1},
		{name: "count exceeds pixels", img: image.NewNRGBA(image.Rect(0, 0, 2, 2)), count: 5},
		{name: "negative compactness", img: image.NewNRGBA(image.Rect(0, 0, 4, 4)), count: 2, compactness: -1},
		{name: "non-finite compactness", img: image.NewNRGBA(image.Rect(0, 0, 4, 4)), count: 2, compactness: math.NaN()},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := Run(test.img, test.count, test.compactness); err == nil {
				t.Fatal("expected an error")
			}
		})
	}
}
