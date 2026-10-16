package superpixel_utils

import (
	"image"
	"testing"
)

func TestGrowShrinkMutatesMaskInPlace(t *testing.T) {
	mask := newMask(5, 5, image.Pt(2, 2))

	GrowShrink(mask, 1)

	if got := countSelected(t, mask); got != 5 {
		t.Fatalf("expected 5 selected pixels after dilation, got %d", got)
	}
	for _, p := range []image.Point{
		image.Pt(2, 2),
		image.Pt(1, 2),
		image.Pt(3, 2),
		image.Pt(2, 1),
		image.Pt(2, 3),
	} {
		if !mask.PositiveAt(p.X, p.Y) {
			t.Errorf("expected pixel (%d, %d) to be selected", p.X, p.Y)
		}
	}
	if mask.PositiveAt(1, 1) {
		t.Error("expected diagonal pixel (1, 1) to remain unselected")
	}
}

func TestSmoothMaskMutatesMaskInPlace(t *testing.T) {
	mask := newMask(3, 3)
	for y := 0; y < mask.Height; y++ {
		for x := 0; x < mask.Width; x++ {
			mask.SetValueAt(x, y, true)
		}
	}
	mask.SetValueAt(1, 1, false)

	SmoothMask(mask, 1)

	if got := countSelected(t, mask); got != 9 {
		t.Fatalf("expected all 9 pixels to be selected after smoothing, got %d", got)
	}
}

func TestSmoothMaskNonPositiveRadiusLeavesMaskUnchanged(t *testing.T) {
	mask := newMask(3, 3, image.Pt(1, 1))
	before := mask.Clone()

	SmoothMask(mask, 0)

	if !mask.Equals(before) {
		t.Fatal("expected zero-radius smoothing to leave the mask unchanged")
	}
}
