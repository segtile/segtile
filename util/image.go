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

	"github.com/struffel/annotation-tool/superpixel"
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

// BoundaryDistanceScore
func BoundaryDistanceScore(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
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
		bestDistancesSum += math.Exp(-math.Max(0, math.Sqrt(bestDistance2)-1))
	}

	return bestDistancesSum / float64(len(boundariesOriginal)), nil
}

func F1Score(maskOriginal, maskApproximated *image.Gray16) (float64, error) {
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
func BoundaryLength(image *image.Gray16) int {
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

func FillRatioOfOwnBoundingBox(mask *image.Gray16) float64 {
	bounds := mask.Bounds()
	minX, minY := bounds.Max.X, bounds.Max.Y
	maxX, maxY := bounds.Min.X, bounds.Min.Y
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.Gray16At(x, y).Y > 0 {
				if x < minX {
					minX = x
				}
				if x > maxX {
					maxX = x
				}
				if y < minY {
					minY = y
				}
				if y > maxY {
					maxY = y
				}
			}
		}
	}
	if minX > maxX || minY > maxY {
		return 0
	}
	bbArea := float64((maxX - minX + 1) * (maxY - minY + 1))
	covered := float64(Area(mask))
	return covered / bbArea
}

func FillRatioOfOwnBoundingBoxPerSuperpixel(labelMap *image.Gray16, spCount int) float64 {
	type boundingBox struct {
		minX, minY, maxX, maxY int
	}

	spPixelCount := make([]int, spCount)
	spBoundingBox := make([]boundingBox, spCount)
	bounds := labelMap.Bounds()
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			spLabel := labelMap.Gray16At(x, y).Y
			spPixelCount[spLabel]++
			if spPixelCount[spLabel] == 1 {
				spBoundingBox[spLabel] = boundingBox{x, y, x, y}
			} else {
				if x < spBoundingBox[spLabel].minX {
					spBoundingBox[spLabel].minX = x
				}
				if x > spBoundingBox[spLabel].maxX {
					spBoundingBox[spLabel].maxX = x
				}
				if y < spBoundingBox[spLabel].minY {
					spBoundingBox[spLabel].minY = y
				}
				if y > spBoundingBox[spLabel].maxY {
					spBoundingBox[spLabel].maxY = y
				}
			}
		}
	}

	var totalFillRatio float64
	for i := 0; i < spCount; i++ {
		if spPixelCount[i] > 0 {
			area := float64(spPixelCount[i])
			bbArea := float64((spBoundingBox[i].maxX - spBoundingBox[i].minX + 1) * (spBoundingBox[i].maxY - spBoundingBox[i].minY + 1))
			totalFillRatio += area / bbArea
		}
	}
	return totalFillRatio / float64(spCount)
}

// Computes for each superpixel which other superpixels are its neighbors (i.e. share a border with it)
func LabelMapNeighborhoods(labelMap *image.Gray16, spCount int) [][]uint16 {

	bounds := labelMap.Bounds()
	neighbors := make([][]uint16, spCount)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			currentLabel := labelMap.Gray16At(x, y).Y
			if currentLabel == 0 {
				continue
			}
			if neighbors[currentLabel] == nil {
				neighbors[currentLabel] = make([]uint16, 0)
			}

			// Check 4-neighborhoods
			for _, offset := range []struct{ dx, dy int }{{-1, 0}, {1, 0}, {0, -1}, {0, 1}} {
				nx, ny := x+offset.dx, y+offset.dy
				if nx >= bounds.Min.X && nx < bounds.Max.X && ny >= bounds.Min.Y && ny < bounds.Max.Y {
					neighborLabel := labelMap.Gray16At(nx, ny).Y
					if neighborLabel != 0 && neighborLabel != currentLabel {
						neighbors[currentLabel] = append(neighbors[currentLabel], neighborLabel)
					}
				}
			}
		}
	}
	return neighbors
}

func CountBorderingSuperpixels(labelMap *image.Gray16, selection []bool) (selectedSpCount int, selectedBorderingSpCount int) {

	neighbors := LabelMapNeighborhoods(labelMap, len(selection))

	for spLabel, selected := range selection {
		if selected {
			selectedSpCount++
			for _, neighbor := range neighbors[spLabel] {
				if !selection[neighbor] {
					selectedBorderingSpCount++
					break
				}
			}
		}

	}
	return selectedSpCount, selectedBorderingSpCount
}

// When clicking on an unselected superpixel, add all connected and still unselected superpixels to the selection
func SuperpixelFloodFillSelection(labelMap *image.Gray16, x, y int, selection []bool) (newSelection []bool) {
	targetLabel := labelMap.Gray16At(x, y).Y
	if targetLabel == 0 || selection[targetLabel] {
		return selection
	}
	neighbors := LabelMapNeighborhoods(labelMap, len(selection))
	queue := []uint16{targetLabel}
	newSelection = make([]bool, len(selection))
	copy(newSelection, selection)
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if !newSelection[current] {
			newSelection[current] = true
			queue = append(queue, neighbors[current]...)
		}
	}
	return newSelection
}

func HasNonZeroPixels(img *image.Gray16) bool {
	for y := img.Bounds().Min.Y; y < img.Bounds().Max.Y; y++ {
		for x := img.Bounds().Min.X; x < img.Bounds().Max.X; x++ {
			if img.Gray16At(x, y).Y > 0 {
				return true
			}
		}
	}
	return false
}

func SuperpixelEvaluation(groundTruthMask *image.Gray16, spResult *superpixel.SpResult) (labelingCalibrationScore float64,
	f1Score float64, densityCalibrationScore float64, regularityCalibrationScore float64, averageBoundaryDistance float64, selectedSpCount int, selectedBorderingSpCount int) {

	// Create Metrics
	threshold := 0.5
	selectionFromMask := spResult.SelectFromMask(groundTruthMask, threshold)
	approximatedMask := spResult.SelectionToMask(selectionFromMask)
	selectedSpCount, selectedBorderingSpCount = CountBorderingSuperpixels(spResult.LabelMap, selectionFromMask)

	// F1 Score
	f1Score, _ = F1Score(groundTruthMask, approximatedMask)

	// Density Calibration Score
	densityCalibrationScore = 0.0
	if selectedSpCount > 0 {
		borderPortionOfSelection := float64(selectedBorderingSpCount) / float64(selectedSpCount)
		desiredBorderPortionOfSelection := 1 - math.Pow(float64(selectedSpCount)/float64(spResult.SpCount), 0.5)
		densityCalibrationScore = math.Min(borderPortionOfSelection, desiredBorderPortionOfSelection) / math.Max(borderPortionOfSelection, desiredBorderPortionOfSelection)
	}

	// Regularity Calibration Score
	groundTruthFillRatio := FillRatioOfOwnBoundingBox(groundTruthMask)
	superpixelFillRatio := FillRatioOfOwnBoundingBoxPerSuperpixel(spResult.LabelMap, spResult.SpCount)
	regularityCalibrationScore = 0.0
	if groundTruthFillRatio > 0.0 || superpixelFillRatio > 0.0 {
		regularityCalibrationScore = math.Min(groundTruthFillRatio, superpixelFillRatio) / math.Max(groundTruthFillRatio, superpixelFillRatio)
	}

	// Final Labeling Calibration Score
	labelingCalibrationScore = f1Score * densityCalibrationScore * regularityCalibrationScore

	// Bonus: Average Boundary Distance
	averageBoundaryDistance, _ = AverageBoundaryDistance(groundTruthMask, approximatedMask)

	return labelingCalibrationScore, f1Score, densityCalibrationScore, regularityCalibrationScore, averageBoundaryDistance, selectedSpCount, selectedBorderingSpCount
}
