package superpixel_test

import (
	"image"
	_ "image/png"
	"os"
	"testing"

	"github.com/segtile/segtile/pkg/superpixel"
	_ "github.com/segtile/segtile/pkg/superpixel/rss"
	_ "github.com/segtile/segtile/pkg/superpixel/scale_adaptive"
	_ "github.com/segtile/segtile/pkg/superpixel/slic"

	//_ "github.com/segtile/segtile/pkg/superpixel/snic"
	_ "github.com/segtile/segtile/pkg/superpixel/vcell"
)

// BenchmarkSuperpixel runs one sub-benchmark per registered algorithm, so each gets its own stats.
func BenchmarkSuperpixel(b *testing.B) {
	reader, err := os.Open("testdata/grapes.png")
	if err != nil {
		b.Fatal(err)
	}
	defer reader.Close()
	img, _, err := image.Decode(reader)
	if err != nil {
		b.Fatal(err)
	}

	for _, id := range superpixel.ListAlgorithms() {
		params, err := superpixel.GetDefaultParameterValues(id)
		if err != nil {
			b.Fatal(err)
		}

		b.Run(id, func(b *testing.B) {
			for b.Loop() {
				if _, err := superpixel.Run(id, img, params); err != nil {
					b.Fatal(err)
				}
			}
		})
	}
}
