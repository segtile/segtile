package superpixel

import (
	"image"
	"image/png"
	"os"
	"testing"
)

func loadGray16Testdata(t *testing.T, path string) *image.Gray16 {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("failed to open %s: %v", path, err)
	}
	defer f.Close()

	src, err := png.Decode(f)
	if err != nil {
		t.Fatalf("failed to decode %s: %v", path, err)
	}

	gray16, ok := src.(*image.Gray16)
	if !ok {
		t.Fatalf("%s is not a 16-bit grayscale PNG (got %T)", path, src)
	}
	return gray16
}

func TestMakeSpResult_RemapsDiscontinuousLabels(t *testing.T) {
	continuous := loadGray16Testdata(t, "testdata/grid_4x4_continuous.png")
	discontinuous := loadGray16Testdata(t, "testdata/grid_4x4_discontinuous.png")

	continuousResult := MakeSpResult(continuous)
	discontinuousResult := MakeSpResult(discontinuous)

	if continuousResult.SpCount != discontinuousResult.SpCount {
		t.Fatalf("SpCount mismatch: continuous=%d discontinuous=%d",
			continuousResult.SpCount, discontinuousResult.SpCount)
	}

	bounds := continuousResult.LabelMap.Bounds()
	if bounds != discontinuousResult.LabelMap.Bounds() {
		t.Fatalf("bounds mismatch: continuous=%v discontinuous=%v", bounds, discontinuousResult.LabelMap.Bounds())
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			want := continuousResult.LabelMap.Gray16At(x, y).Y
			got := discontinuousResult.LabelMap.Gray16At(x, y).Y
			if want != got {
				t.Errorf("label mismatch at (%d,%d): continuous=%d discontinuous(post-cleanup)=%d", x, y, want, got)
			}
		}
	}
}
