package rss

import (
	"image"
	"math"
	"testing"
)

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
