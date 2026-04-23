package util

import (
	"fmt"
	"image"
)

func DiceCoefficient(mask1, mask2 *image.Gray16) (float64, error) {
	if mask1.Bounds() != mask2.Bounds() {
		return 0, fmt.Errorf("Masks must have the same dimensions")
	}

	var intersection, union int
	for y := 0; y < mask1.Bounds().Dy(); y++ {
		for x := 0; x < mask1.Bounds().Dx(); x++ {
			p1 := mask1.Gray16At(x, y)
			p2 := mask2.Gray16At(x, y)
			if p1.Y > 0 && p2.Y > 0 {
				intersection++
			}
			if p1.Y > 0 {
				union++
			}
			if p2.Y > 0 {
				union++
			}
		}
	}

	if union == 0 {
		return 0, nil
	}

	return float64(2*intersection) / float64(union), nil
}
