package superpixel_utils

import (
	"image"
)

// Returns the number of pixels in the mask that are not 0
func Area(mask *image.Gray16) int {
	bounds := mask.Bounds()

	covered := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.Gray16At(x, y).Y > 0 {
				covered++
			}
		}
	}

	return covered
}

func AreaPerValue(labelMap *image.Gray16) map[uint16]int {
	bounds := labelMap.Bounds()
	areaMap := make(map[uint16]int)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			value := labelMap.Gray16At(x, y).Y
			if value > 0 {
				areaMap[value]++
			}
		}
	}

	return areaMap
}

// PolygonArea computes the area of a simple polygon using the shoelace formula.
func PolygonArea(polygon []image.Point) float64 {
	if len(polygon) < 3 {
		return 0
	}

	sum := 0
	for i := range polygon {
		j := (i + 1) % len(polygon)
		sum += polygon[i].X*polygon[j].Y - polygon[j].X*polygon[i].Y
	}

	area := float64(sum) / 2
	if area < 0 {
		area = -area
	}
	return area
}
