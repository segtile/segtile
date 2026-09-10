package rendering

import (
	"testing"

	"github.com/struffel/annotation-tool/internal/io_utils"
)

func BenchmarkCalculateSelectionOutlineFromSlice(b *testing.B) {
	labelmap := io_utils.ToGray16(io_utils.LoadImage("./testdata/labelmap.png"))
	selection := make([]bool, 16)
	selection[1] = true
	selection[12] = true
	selection[5] = true

	b.Run("CalculateSelectionOutlineFromSlice", func(b *testing.B) {
		for b.Loop() {
			CalculateSelectionOutlineFromSlice(labelmap, selection)
		}
	})
}
