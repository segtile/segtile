package util

import (
	"fmt"
	"image"
	"image/draw"
	"image/jpeg"
	"image/png"
	"math"
	"os"
	"path/filepath"

	"golang.org/x/image/tiff"
)

func LoadImage(path string) image.Image {
	f, err := os.Open(filepath.Clean(path))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	src, _, err := image.Decode(f)
	if err != nil {
		panic(err)
	}
	return src
}

func ToNRGBA(src image.Image) *image.NRGBA {
	if n, ok := src.(*image.NRGBA); ok {
		return n
	}
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func ToGray16(src image.Image) *image.Gray16 {
	if g, ok := src.(*image.Gray16); ok {
		return g
	}
	b := src.Bounds()
	dst := image.NewGray16(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

func SaveImage(path string, img image.Image) {
	f, err := os.Create(filepath.Clean(path))
	if err != nil {
		panic(err)
	}
	defer f.Close()
	ext := filepath.Ext(path)
	switch ext {
	case ".png":
		err = png.Encode(f, img)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(f, img, &jpeg.Options{Quality: 99})
	case ".tiff", ".tif":
		err = tiff.Encode(f, img, nil)
	default:
		panic("Unsupported file format: " + ext)
	}
	if err != nil {
		panic(err)
	}
}

func CopyImage(src image.Image) *image.NRGBA {
	b := src.Bounds()
	dst := image.NewNRGBA(b)
	draw.Draw(dst, b, src, b.Min, draw.Src)
	return dst
}

// Determines the relative portion of boundary pixels in maskOriginal that are within epsilon distance of any boundary pixel in maskApproximated.
func AverageBoundaryDistance(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
	if maskOriginal.Bounds() != maskApproximated.Bounds() {
		return 0, fmt.Errorf("Masks must have the same dimensions")
	}

	bounds := maskOriginal.Bounds()

	maxDistance2 := float64(bounds.Dx()*bounds.Dx() + bounds.Dy()*bounds.Dy()) // Max possible distance squared

	// Collect boundary pixel positions for both masks
	type point struct{ x, y int }
	var boundaries1, boundaries2 []point
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if maskOriginal.Gray16At(x, y).Y > 0 && (maskOriginal.Gray16At(x-1, y).Y == 0 || maskOriginal.Gray16At(x+1, y).Y == 0 || maskOriginal.Gray16At(x, y-1).Y == 0 || maskOriginal.Gray16At(x, y+1).Y == 0) {
				boundaries1 = append(boundaries1, point{x, y})
			}
			if maskApproximated.Gray16At(x, y).Y > 0 && (maskApproximated.Gray16At(x-1, y).Y == 0 || maskApproximated.Gray16At(x+1, y).Y == 0 || maskApproximated.Gray16At(x, y-1).Y == 0 || maskApproximated.Gray16At(x, y+1).Y == 0) {
				boundaries2 = append(boundaries2, point{x, y})
			}
		}
	}

	if len(boundaries1) == 0 || len(boundaries2) == 0 {
		return 0, nil
	}

	// Check each boundary pixel in mask1 against all boundary pixels in mask2
	bestDistancesSum := 0.0
	for _, p := range boundaries1 {
		bestDistance2 := maxDistance2 // Max possible distance squared
		for _, q := range boundaries2 {
			deltaX := float64(p.x - q.x)
			deltaY := float64(p.y - q.y)
			distance2 := deltaX*deltaX + deltaY*deltaY
			if distance2 < bestDistance2 {
				bestDistance2 = distance2
			}
		}
		bestDistancesSum += math.Sqrt(bestDistance2)
	}

	return bestDistancesSum / float64(len(boundaries1)), nil
}

func DiceCoefficient(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
	if maskOriginal.Bounds() != maskApproximated.Bounds() {
		return 0, fmt.Errorf("Masks must have the same dimensions")
	}

	var intersection, union int
	for y := 0; y < maskOriginal.Bounds().Dy(); y++ {
		for x := 0; x < maskOriginal.Bounds().Dx(); x++ {
			p1 := maskOriginal.Gray16At(x, y)
			p2 := maskApproximated.Gray16At(x, y)
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

func LabelMapNeighborhoods(labelMap *image.Gray16) map[uint16][]uint16 {

	bounds := labelMap.Bounds()
	neighbors := make(map[uint16]map[uint16]struct{})

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			currentLabel := labelMap.Gray16At(x, y).Y
			if currentLabel == 0 {
				continue
			}
			if _, exists := neighbors[currentLabel]; !exists {
				neighbors[currentLabel] = make(map[uint16]struct{})
			}

			// Check 4-neighborhoods
			for _, offset := range []struct{ dx, dy int }{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+offset.dx, y+offset.dy
				if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
					neighborLabel := labelMap.Gray16At(nx, ny).Y
					if neighborLabel != 0 && neighborLabel != currentLabel {
						neighbors[currentLabel][neighborLabel] = struct{}{}
					}
				}
			}
		}
	}

	// Convert neighbor sets to slices
	result := make(map[uint16][]uint16)
	for label, neighborSet := range neighbors {
		for neighbor := range neighborSet {
			result[label] = append(result[label], neighbor)
		}
	}

	return result
}

// When clicking on an unselected superpixel, add all connected and still unselected superpixels to the selection
func SuperpixelFloodFillSelection(labelMap *image.Gray16, x, y int, selectionIndex []bool) (newSelectionIndex []bool) {
	targetLabel := labelMap.Gray16At(x, y).Y
	if targetLabel == 0 || selectionIndex[targetLabel] {
		return selectionIndex
	}
	neighbors := LabelMapNeighborhoods(labelMap)
	queue := []uint16{targetLabel}
	newSelectionIndex = make([]bool, len(selectionIndex))
	copy(newSelectionIndex, selectionIndex)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if !newSelectionIndex[current] {
			newSelectionIndex[current] = true
			queue = append(queue, neighbors[current]...)
		}
	}
	return newSelectionIndex
}
