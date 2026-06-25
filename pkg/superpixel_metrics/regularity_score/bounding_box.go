package regularity_score

import (
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
)

type boundingBox struct {
	minX, minY, maxX, maxY int
	initialized            bool
}

func (bb *boundingBox) includePoint(x, y int) {
	if !bb.initialized {
		bb.minX, bb.minY, bb.maxX, bb.maxY = x, y, x, y
		bb.initialized = true
		return
	}
	if x < bb.minX {
		bb.minX = x
	}
	if x > bb.maxX {
		bb.maxX = x
	}
	if y < bb.minY {
		bb.minY = y
	}
	if y > bb.maxY {
		bb.maxY = y
	}
}

func (bb boundingBox) area() int {
	if !bb.initialized {
		return 0
	}
	return (bb.maxX - bb.minX + 1) * (bb.maxY - bb.minY + 1)
}

func FillRatioOfOwnBoundingBox(mask *image.Gray16) float64 {
	var bb boundingBox
	bounds := mask.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.Gray16At(x, y).Y > 0 {
				bb.includePoint(x, y)
			}
		}
	}
	if !bb.initialized {
		return 0
	}
	return float64(superpixel_metrics.Area(mask)) / float64(bb.area())
}

func FillRatioOfOwnBoundingBoxPerSuperpixel(labelMap *image.Gray16, spCount int) float64 {
	spPixelCount := make([]int, spCount)
	spBoundingBox := make([]boundingBox, spCount)
	bounds := labelMap.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			spLabel := labelMap.Gray16At(x, y).Y
			spPixelCount[spLabel]++
			spBoundingBox[spLabel].includePoint(x, y)
		}
	}

	var totalFillRatio float64
	for i := 0; i < spCount; i++ {
		if spPixelCount[i] > 0 {
			totalFillRatio += float64(spPixelCount[i]) / float64(spBoundingBox[i].area())
		}
	}
	return totalFillRatio / float64(spCount)
}
