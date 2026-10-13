package rendering

import (
	"testing"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
)

func BenchmarkCalculateSelectionOutlineFromSlice(b *testing.B) {
	labelmap := superpixel_map.NewLabelMapFromImage(io_utils.LoadImage("./testdata/labelmap.png"))

	b.Run("CalculateOutline", func(b *testing.B) {
		for b.Loop() {
			CalculateOutline(labelmap, 5)
		}
	})
}
