package rendering

import (
	"testing"

	"github.com/struffel/annotation-tool/internal/io_utils"
)

func BenchmarkCalculateSelectionOutlineFromSlice(b *testing.B) {
	labelmap := io_utils.ToGray16(io_utils.LoadImage("./testdata/labelmap.png"))

	b.Run("CalculateOutline", func(b *testing.B) {
		for b.Loop() {
			CalculateOutline(labelmap, 5)
		}
	})
}
