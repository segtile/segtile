package evaluation

import (
	"image"
	"testing"

	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

func TestDice(t *testing.T) {
	newMask := func(pixels ...image.Point) *superpixel_map.Mask {
		m := superpixel_map.NewMask(4, 4)
		for _, p := range pixels {
			m.SetValueAt(p.X, p.Y, true)
		}
		return m
	}

	for _, tc := range []struct {
		name string
		a, b *superpixel_map.Mask
		want float64
	}{
		{name: "identical", a: newMask(image.Pt(0, 0), image.Pt(1, 1)), b: newMask(image.Pt(0, 0), image.Pt(1, 1)), want: 1.0},
		{name: "disjoint", a: newMask(image.Pt(0, 0)), b: newMask(image.Pt(1, 1)), want: 0.0},
		{name: "half overlap", a: newMask(image.Pt(0, 0), image.Pt(1, 1)), b: newMask(image.Pt(1, 1), image.Pt(2, 2)), want: 0.5},
		{name: "nil selection", a: nil, b: newMask(image.Pt(0, 0)), want: 0.0},
		{name: "both empty", a: newMask(), b: newMask(), want: 0.0},
		{name: "size mismatch", a: superpixel_map.NewMask(2, 2), b: newMask(image.Pt(0, 0)), want: 0.0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if got := Dice(tc.a, tc.b); got != tc.want {
				t.Fatalf("Dice() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestModeForIndexAlternates(t *testing.T) {
	for i, want := range []Mode{ModeSuperpixel, ModeManual, ModeSuperpixel, ModeManual} {
		if got := ModeForIndex(i); got != want {
			t.Fatalf("ModeForIndex(%d) = %v, want %v", i, got, want)
		}
	}
}

func TestBuildTintedOverlays(t *testing.T) {
	mask := superpixel_map.NewMask(4, 4)
	mask.SetValueAt(1, 1, true)

	red, green := buildTintedOverlays(mask)

	// The overlay follows the mask exactly (no blur) at ~50% opacity.
	setOffset := (1*4 + 1) * 4
	clearOffset := 0
	if red.Pix[setOffset] != 255 || red.Pix[setOffset+3] != 127 {
		t.Fatalf("unexpected red overlay at set pixel: %v", red.Pix[setOffset:setOffset+4])
	}
	if green.Pix[setOffset+1] != 220 || green.Pix[setOffset+3] != 127 {
		t.Fatalf("unexpected green overlay at set pixel: %v", green.Pix[setOffset:setOffset+4])
	}
	if red.Pix[clearOffset+3] != 0 || green.Pix[clearOffset+3] != 0 {
		t.Fatal("expected cleared pixels to be fully transparent")
	}
}

func TestBeginUpdateEndImage(t *testing.T) {
	state := NewState(0.95)
	gt := superpixel_map.NewMask(4, 4)
	gt.SetValueAt(1, 1, true)
	gt.SetValueAt(2, 2, true)

	img := image.NewNRGBA(image.Rect(0, 0, 4, 4))
	state.BeginImage(1, "img.png", img, gt)

	if state.CurrentMode() != ModeManual {
		t.Fatalf("index 1 must be manual-only, got %v", state.CurrentMode())
	}
	if state.CurrentDice() != 0 || state.AboveThreshold() {
		t.Fatal("dice must start at 0")
	}

	// Perfect annotation reaches dice 1 and passes the threshold.
	perfect := gt.Clone()
	state.UpdateDice(perfect)
	if state.CurrentDice() != 1 || !state.AboveThreshold() {
		t.Fatalf("dice = %v, want 1 and above threshold", state.CurrentDice())
	}

	snap := state.Snapshot()
	if !snap.HasImage || snap.Dice != 1 || snap.Mode != ModeManual {
		t.Fatalf("unexpected snapshot: %+v", snap)
	}

	state.EndImage()
	if len(state.results) != 1 || !state.results[0].Passed || state.results[0].Mode != ModeManual {
		t.Fatalf("unexpected results: %+v", state.results)
	}

	// Ending twice must not record a second result.
	state.EndImage()
	if len(state.results) != 1 {
		t.Fatal("EndImage must be idempotent")
	}
}
