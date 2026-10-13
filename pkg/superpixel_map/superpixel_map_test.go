package superpixel_map

import (
	"image"
	"image/color"
	"reflect"
	"testing"
)

func TestNewLabelMapFromImage_PreservesGray16LabelsAndBounds(t *testing.T) {
	img := image.NewGray16(image.Rect(5, 7, 8, 8))
	img.SetGray16(5, 7, color.Gray16{Y: 0})
	img.SetGray16(6, 7, color.Gray16{Y: 15})
	img.SetGray16(7, 7, color.Gray16{Y: 65535})

	labelMap := NewLabelMapFromImage(img)

	if labelMap.Width != 3 || labelMap.Height != 1 {
		t.Fatalf("dimensions = %dx%d, want 3x1", labelMap.Width, labelMap.Height)
	}
	if want := []uint32{0, 15, 65535}; !reflect.DeepEqual(labelMap.Labels, want) {
		t.Fatalf("labels = %v, want %v", labelMap.Labels, want)
	}
}

func TestNewMaskFromImage_UsesImageBounds(t *testing.T) {
	img := image.NewGray16(image.Rect(5, 7, 7, 8))
	img.SetGray16(5, 7, color.Gray16{Y: 0})
	img.SetGray16(6, 7, color.Gray16{Y: 1})

	mask := NewMaskFromImage(img)

	if mask.Width != 2 || mask.Height != 1 {
		t.Fatalf("dimensions = %dx%d, want 2x1", mask.Width, mask.Height)
	}
	if want := []bool{false, true}; !reflect.DeepEqual(mask.Mask, want) {
		t.Fatalf("mask = %v, want %v", mask.Mask, want)
	}
}
