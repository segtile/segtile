package superpixel

import (
	"image"
	"image/color"
	"image/png"
	"math/rand"
	"os"
	"testing"

	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

func loadTestData(t testing.TB, path string) *superpixel_map.LabelMap {
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

	labelMap, ok := src.(*image.Gray16)
	if !ok {
		t.Fatalf("%s is not a 16-bit grayscale PNG (got %T)", path, src)
	}
	return superpixel_map.NewLabelMapFromImage(labelMap)
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
	connected := loadTestData(t, "testdata/grid_4x4_connected.png")
	disconnected := loadTestData(t, "testdata/grid_4x4_disconnected.png")

	connectedResult := SpResult{LabelMap: connected}
	connectedResult.Optimize()
	disconnectedResult := SpResult{LabelMap: disconnected}

	disconnectedResult.Optimize()

	// At this point, the two label maps should be identical

	bounds := connectedResult.LabelMap.Bounds()
	if bounds != disconnectedResult.LabelMap.Bounds() {
		t.Fatalf("bounds mismatch: connected=%v disconnected=%v", bounds, disconnectedResult.LabelMap.Bounds())
	}

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			want := connectedResult.LabelMap.ValueAt(x, y)
			got := disconnectedResult.LabelMap.ValueAt(x, y)
			if want != got {
				t.Errorf("label mismatch at (%d,%d): connected=%d disconnected(post-enforcement)=%d", x, y, want, got)
			}
		}
	}
}

func TestEnforceConnectivity_MoreThan65535Components(t *testing.T) {
	const width, height = 512, 512
	labelMap := superpixel_map.NewLabelMap(width, height)
	for y := 0; y < height; y += 2 {
		for x := 0; x < width; x += 2 {
			labelMap.SetLabelAt(x, y, 1)
		}
	}

	result := NewSpResult(labelMap)
	result.Optimize()

	if result.SpCount != 2 {
		t.Fatalf("expected 2 superpixel labels after connectivity enforcement, got %d", result.SpCount)
	}
}

func TestMakeSpResult_RemapsDiscontinuousLabels(t *testing.T) {
	continuous := loadTestData(t, "testdata/grid_4x4_continuous.png")
	discontinuous := loadTestData(t, "testdata/grid_4x4_discontinuous.png")

	continuousResult := NewSpResult(continuous)
	discontinuousResult := NewSpResult(discontinuous)
	continuousResult.Optimize()
	discontinuousResult.Optimize()

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
			want := continuousResult.LabelMap.ValueAt(x, y)
			got := discontinuousResult.LabelMap.ValueAt(x, y)
			if want != got {
				t.Errorf("label mismatch at (%d,%d): continuous=%d discontinuous(post-cleanup)=%d", x, y, want, got)
			}
		}
	}
}

func TestSetActualSpCount(t *testing.T) {
	img := loadTestData(t, "testdata/bad_labelmap_32colors.png")
	spResult := NewSpResult(img)
	spResult.Optimize()
	if spResult.SpCount != 32 {
		t.Errorf("Expected SpCount to be 32 but got %d", spResult.SpCount)
	}
}

func BenchmarkOptimize(b *testing.B) {
	labelMap := loadTestData(b, "testdata/bad_labelmap_32colors.png")
	originalResult := &SpResult{
		LabelMap: labelMap,
	}
	originalResult.Optimize()
	b.ResetTimer()
	for b.Loop() {
		spResult := originalResult.Clone()
		spResult.Optimize()
	}

}
