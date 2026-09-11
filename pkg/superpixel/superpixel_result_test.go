package superpixel

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"
)

func loadGray16Testdata(t testing.TB, path string) *image.Gray16 {
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

func saveGray16Testdata(path string, img *image.Gray16) {
	f, err := os.Create(path)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	if err := png.Encode(f, img); err != nil {
		panic(err)
	}
}

// generatePseudoRandomGray16 generates a pseudo-random 16-bit grayscale image of the specified width and height.
// The pixel values are unique and deterministic based on a fixed random seed.
func generatePseudoRandomGray16(t *testing.T, width, height int) *image.Gray16 {
	t.Helper()
	rng := rand.New(rand.NewSource(123))
	img := image.NewGray16(image.Rect(0, 0, width, height))
	alreadyUsed := make(map[uint16]bool)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var value uint16
			for {
				value = uint16(rng.Intn(65536))
				if !alreadyUsed[value] {
					alreadyUsed[value] = true
					break
				}
			}
			img.SetGray16(x, y, color.Gray16{Y: value})
		}
	}
	return img
}

func TestEnforceConnectivity_MergesDisjointComponents(t *testing.T) {
	connected := loadGray16Testdata(t, "testdata/grid_4x4_connected.png")
	disconnected := loadGray16Testdata(t, "testdata/grid_4x4_disconnected.png")

	connectedResult := SpResult{LabelMap: connected}
	disconnectedResult := SpResult{LabelMap: disconnected}

	disconnectedResult.EnforceConnectivity()

	// At this point, the two label maps should be identical

	bounds := connectedResult.LabelMap.Bounds()
	if bounds != disconnectedResult.LabelMap.Bounds() {
		t.Fatalf("bounds mismatch: connected=%v disconnected=%v", bounds, disconnectedResult.LabelMap.Bounds())
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			want := connectedResult.LabelMap.Gray16At(x, y).Y
			got := disconnectedResult.LabelMap.Gray16At(x, y).Y
			if want != got {
				t.Errorf("label mismatch at (%d,%d): connected=%d disconnected(post-enforcement)=%d", x, y, want, got)
			}
		}
	}
}

func TestMakeSpResult_RemapsRandomLabels(t *testing.T) {
	original := generatePseudoRandomGray16(t, 32, 32)
	originalResult := NewSpResult(original, 0)
	originalResult.EnforceContiguousIndexing()

	// At this point, originalResult.LabelMap should have labels ranging from 0 to 1023 for this 32x32 image.

	// Check which labels are actually used
	seenLabels := make([]bool, 65535)
	for i := 0; i < originalResult.LabelMap.Bounds().Dy(); i++ {
		for j := 0; j < originalResult.LabelMap.Bounds().Dx(); j++ {
			label := originalResult.LabelMap.Gray16At(j, i).Y
			seenLabels[label] = true
		}
	}

	// Throw errors for any labels that are expected to be present but are not, or the other way around.
	for label := uint16(0); label < 65535; label++ {
		if label < 1024 && !seenLabels[label] {
			t.Errorf("Expected label %d to be present in the original result, but it was not found.", label)
		}
		if label >= 1024 && seenLabels[label] {
			t.Errorf("Did not expect label %d to be present in the original result, but it was found.", label)
		}
	}

}

func TestMakeSpResult_RemapsDiscontinuousLabels(t *testing.T) {
	continuous := loadGray16Testdata(t, "testdata/grid_4x4_continuous.png")
	discontinuous := loadGray16Testdata(t, "testdata/grid_4x4_discontinuous.png")

	continuousResult := NewSpResult(continuous, 0)
	discontinuousResult := NewSpResult(discontinuous, 0)
	continuousResult.EnforceContiguousIndexing()
	discontinuousResult.EnforceContiguousIndexing()

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

func TestSetActualSpCount(t *testing.T) {
	img := loadGray16Testdata(t, "testdata/bad_labelmap_32colors.png")
	spResult := NewSpResult(img, 0)
	spResult.SetActualSpCount()
	if spResult.SpCount != 32 {
		t.Errorf("Expected SpCount to be 32 but got %d", spResult.SpCount)
	}
}

func BenchmarkEnforceConnectivity(b *testing.B) {
	img := loadGray16Testdata(b, "testdata/bad_labelmap_32colors.png")
	spResult := NewSpResult(img, 0)
	spResult.SetActualSpCount()
	b.ResetTimer()
	for b.Loop() {
		spResult.LabelMap = img
		spResult.EnforceConnectivity()
	}

}
