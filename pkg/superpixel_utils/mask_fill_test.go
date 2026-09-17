package superpixel_utils

import (
	"image"
	"image/color"
	"testing"
)

// newMask creates a width x height mask with the given pixels set to selected (65535).
func newMask(width, height int, selected ...image.Point) *image.Gray16 {
	mask := image.NewGray16(image.Rect(0, 0, width, height))
	for _, p := range selected {
		mask.SetGray16(p.X, p.Y, color.Gray16{Y: 65535})
	}
	return mask
}

// selectedColumn selects a full vertical line of pixels.
func selectedColumn(mask *image.Gray16, x int) {
	for y := 0; y < mask.Bounds().Dy(); y++ {
		mask.SetGray16(x, y, color.Gray16{Y: 65535})
	}
}

func countSelected(t *testing.T, mask *image.Gray16) int {
	t.Helper()
	count := 0
	for y := mask.Bounds().Min.Y; y < mask.Bounds().Max.Y; y++ {
		for x := mask.Bounds().Min.X; x < mask.Bounds().Max.X; x++ {
			if mask.Gray16At(x, y).Y != 0 {
				count++
			}
		}
	}
	return count
}

func TestFloodFillMaskFromPoint_FillsUpToBarrier(t *testing.T) {
	// 6x5 mask with a full vertical wall at x=3; filling from the left must not cross it.
	mask := newMask(6, 5)
	selectedColumn(mask, 3)

	if !FloodFillMaskFromPoint(mask, 0, 0) {
		t.Fatal("expected flood fill to change the mask")
	}

	if got := countSelected(t, mask); got != 3*5+5 { // wall (5) + left region (3*5)
		t.Fatalf("expected %d selected pixels, got %d", 3*5+5, got)
	}
	for x := 4; x < 6; x++ {
		for y := 0; y < 5; y++ {
			if mask.Gray16At(x, y).Y != 0 {
				t.Fatalf("flood fill crossed the barrier at (%d, %d)", x, y)
			}
		}
	}
}

func TestFloodFillMaskFromPoint_NoOpOnSelectedPixel(t *testing.T) {
	mask := newMask(4, 4, image.Pt(1, 1))

	if FloodFillMaskFromPoint(mask, 1, 1) {
		t.Fatal("expected no change when starting on a selected pixel")
	}
	if got := countSelected(t, mask); got != 1 {
		t.Fatalf("expected 1 selected pixel, got %d", got)
	}
}

func TestFloodFillMaskFromPoint_OutOfBounds(t *testing.T) {
	mask := newMask(4, 4)
	if FloodFillMaskFromPoint(mask, -1, 0) || FloodFillMaskFromPoint(mask, 4, 0) {
		t.Fatal("expected no change for out-of-bounds start points")
	}
}

func TestFillEnclosedMaskRegions_FillsHole(t *testing.T) {
	// 7x7 mask with a ring (perimeter of the inner 5x5) selected, enclosing a 3x3 hole.
	mask := newMask(7, 7)
	for i := 1; i <= 5; i++ {
		mask.SetGray16(i, 1, color.Gray16{Y: 65535})
		mask.SetGray16(i, 5, color.Gray16{Y: 65535})
		mask.SetGray16(1, i, color.Gray16{Y: 65535})
		mask.SetGray16(5, i, color.Gray16{Y: 65535})
	}

	if !FillEnclosedMaskRegions(mask) {
		t.Fatal("expected hole fill to change the mask")
	}

	// Ring (16) + hole (9) selected; the 24 pixels outside the ring stay unselected.
	if got := countSelected(t, mask); got != 25 {
		t.Fatalf("expected 25 selected pixels, got %d", got)
	}
	if mask.Gray16At(3, 3).Y != 65535 {
		t.Fatal("expected enclosed center pixel to be selected")
	}
	if mask.Gray16At(0, 0).Y != 0 {
		t.Fatal("expected border-connected pixel to stay unselected")
	}
}

func TestFillEnclosedMaskRegions_NoEnclosedRegion(t *testing.T) {
	mask := newMask(4, 4, image.Pt(1, 1), image.Pt(2, 2))

	if FillEnclosedMaskRegions(mask) {
		t.Fatal("expected no change when no enclosed region exists")
	}
	if got := countSelected(t, mask); got != 2 {
		t.Fatalf("expected 2 selected pixels, got %d", got)
	}
}

// largeMask builds a width x height mask with a checkerboard of selected pixels on the
// border rows/columns and a large enclosed empty area in the middle.
func largeMask(width, height int) *image.Gray16 {
	mask := image.NewGray16(image.Rect(0, 0, width, height))
	// Wall around the middle half, leaving an enclosed hole.
	for i := width / 4; i < 3*width/4; i++ {
		mask.SetGray16(i, height/4, color.Gray16{Y: 65535})
		mask.SetGray16(i, 3*height/4, color.Gray16{Y: 65535})
	}
	for j := height / 4; j < 3*height/4; j++ {
		mask.SetGray16(width/4, j, color.Gray16{Y: 65535})
		mask.SetGray16(3*width/4, j, color.Gray16{Y: 65535})
	}
	return mask
}

func BenchmarkFloodFillMaskFromPoint(b *testing.B) {
	for b.Loop() {
		mask := image.NewGray16(image.Rect(0, 0, 2048, 2048))
		FloodFillMaskFromPoint(mask, 1024, 1024)
	}
}

func BenchmarkFillEnclosedMaskRegions(b *testing.B) {
	for b.Loop() {
		mask := largeMask(2048, 2048)
		FillEnclosedMaskRegions(mask)
	}
}
