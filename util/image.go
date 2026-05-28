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

// AverageBoundaryDistance computes the average distance from each boundary pixel in maskOriginal to the nearest boundary pixel in maskApproximated.
func AverageBoundaryDistance(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
	if maskOriginal.Bounds() != maskApproximated.Bounds() {
		return 0, fmt.Errorf("Masks must have the same dimensions")
	}

	bounds := maskOriginal.Bounds()

	maxDistance2 := float64(bounds.Dx()*bounds.Dx() + bounds.Dy()*bounds.Dy()) // Max possible distance squared

	// Collect boundary pixel positions for both masks
	type point struct{ x, y int }
	var boundariesOriginal, boundariesApproximated []point
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if maskOriginal.Gray16At(x, y).Y > 0 && (maskOriginal.Gray16At(x-1, y).Y == 0 || maskOriginal.Gray16At(x+1, y).Y == 0 || maskOriginal.Gray16At(x, y-1).Y == 0 || maskOriginal.Gray16At(x, y+1).Y == 0) {
				boundariesOriginal = append(boundariesOriginal, point{x, y})
			}
			if maskApproximated.Gray16At(x, y).Y > 0 && (maskApproximated.Gray16At(x-1, y).Y == 0 || maskApproximated.Gray16At(x+1, y).Y == 0 || maskApproximated.Gray16At(x, y-1).Y == 0 || maskApproximated.Gray16At(x, y+1).Y == 0) {
				boundariesApproximated = append(boundariesApproximated, point{x, y})
			}
		}
	}

	if len(boundariesOriginal) == 0 || len(boundariesApproximated) == 0 {
		return 0, nil
	}

	// Check each boundary pixel in maskOriginal against all boundary pixels in maskApproximated
	bestDistancesSum := 0.0
	for _, p := range boundariesOriginal {
		bestDistance2 := maxDistance2 // Max possible distance squared
		for _, q := range boundariesApproximated {
			deltaX := float64(p.x - q.x)
			deltaY := float64(p.y - q.y)
			distance2 := deltaX*deltaX + deltaY*deltaY
			if distance2 < bestDistance2 {
				bestDistance2 = distance2
			}
		}
		bestDistancesSum += math.Sqrt(bestDistance2)
	}

	return bestDistancesSum / float64(len(boundariesOriginal)), nil
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

func JaccardIndex(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
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
			if p1.Y > 0 || p2.Y > 0 {
				union++
			}
		}
	}

	if union == 0 {
		return 0, nil
	}

	return float64(intersection) / float64(union), nil
}

// Counts the length of all edges in the image without counting the same edge twice
// by only accounting for the brighter value
func TotalEdgeLength(image *image.Gray16) int {
	bounds := image.Bounds()
	count := 0
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			value := image.Gray16At(x, y).Y
			if image.Gray16At(x-1, y).Y < value || image.Gray16At(x+1, y).Y < value || image.Gray16At(x, y-1).Y < value || image.Gray16At(x, y+1).Y < value {
				count++
			}
		}
	}
	return count
}

// Returns the number of pixels in the mask that are covered
func CoveredArea(mask *image.Gray16) int {
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

func Compactness(mask *image.Gray16) float64 {
	area := float64(CoveredArea(mask))
	edgeLength := float64(TotalEdgeLength(mask))
	if area == 0 {
		return 0
	}
	return (edgeLength * edgeLength) / (4 * math.Pi * area)
}

// Returns the average compactness of all superpixels in the label map
func CompactnessPerSuperpixel(labelMap *image.Gray16, spCount int) float64 {
	spPixelCount := make([]int, spCount)
	spEdgeLength := make([]int, spCount)
	bounds := labelMap.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			spLabel := labelMap.Gray16At(x, y).Y
			spPixelCount[spLabel]++
			value := labelMap.Gray16At(x, y).Y
			if labelMap.Gray16At(x-1, y).Y < value || labelMap.Gray16At(x+1, y).Y < value || labelMap.Gray16At(x, y-1).Y < value || labelMap.Gray16At(x, y+1).Y < value {
				spEdgeLength[spLabel]++
			}
		}
	}

	var totalCompactness float64
	for i := 0; i < spCount; i++ {
		if spPixelCount[i] > 0 {
			area := float64(spPixelCount[i])
			edgeLength := float64(spEdgeLength[i])
			totalCompactness += (edgeLength * edgeLength) / (4 * math.Pi * area)
		}
	}
	return totalCompactness / float64(spCount)
}
