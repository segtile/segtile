package regularity_score_legacy

import (
	"image"
)

type boundingBox struct {
	minX, minY, maxX, maxY int
	initialized            bool
}

func (bb *boundingBox) addPoint(x, y int) {
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

// fillRatioOfOwnBoundingBoxPerIsland computes, in a single pass over islandMap, the fill ratio of
// every ground truth island (label 0 is background and is ignored). The returned slice is indexed
// by island label, so index 0 is always 0.
func fillRatioOfOwnBoundingBoxPerIsland(islandMap *image.Gray16, islandCount uint16) []float64 {
	islandPixelCount := make([]int, islandCount+1)
	islandBoundingBox := make([]boundingBox, islandCount+1)
	bounds := islandMap.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			islandLabel := islandMap.Gray16At(x, y).Y
			if islandLabel == 0 {
				continue
			}
			islandPixelCount[islandLabel]++
			islandBoundingBox[islandLabel].addPoint(x, y)
		}
	}

	fillRatios := make([]float64, islandCount+1)
	for i := uint16(1); i <= islandCount; i++ {
		if islandPixelCount[i] > 0 {
			fillRatios[i] = float64(islandPixelCount[i]) / float64(islandBoundingBox[i].area())
		}
	}
	return fillRatios
}

func fillRatioOfOwnBoundingBoxPerSuperpixel(labelMap *image.Gray16, spCount int) float64 {
	spPixelCount := make([]int, spCount)
	spBoundingBox := make([]boundingBox, spCount)
	bounds := labelMap.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			spLabel := labelMap.Gray16At(x, y).Y
			spPixelCount[spLabel]++
			spBoundingBox[spLabel].addPoint(x, y)
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
