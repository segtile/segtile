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

func FillRatioOfBoundingBox(mask *image.Gray16) float64 {
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

// Calculate the convex hull of a binary mask (anything >0) and return it as a new binary mask
func ConvexHull(img *image.Gray16) *image.Gray16 {
	bounds := img.Bounds()
	result := image.NewGray16(bounds)

	// Collect boundary pixels as input points for the hull
	type point struct{ x, y int }
	var points []point
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if img.Gray16At(x, y).Y > 0 &&
				(img.Gray16At(x-1, y).Y == 0 || img.Gray16At(x+1, y).Y == 0 ||
					img.Gray16At(x, y-1).Y == 0 || img.Gray16At(x, y+1).Y == 0) {
				points = append(points, point{x, y})
			}
		}
	}

	if len(points) == 0 {
		return result
	}

	// Gift wrapping (Jarvis March): find the leftmost point as start
	n := len(points)
	startIdx := 0
	for i, p := range points {
		if p.x < points[startIdx].x || (p.x == points[startIdx].x && p.y < points[startIdx].y) {
			startIdx = i
		}
	}

	var hull []point
	currentIdx := startIdx
	for {
		hull = append(hull, points[currentIdx])
		nextIdx := (currentIdx + 1) % n
		for i := 0; i < n; i++ {
			if i == currentIdx {
				continue
			}
			cx, cy := points[currentIdx].x, points[currentIdx].y
			nx, ny := points[nextIdx].x, points[nextIdx].y
			px, py := points[i].x, points[i].y
			cross := (nx-cx)*(py-cy) - (ny-cy)*(px-cx)
			if cross < 0 {
				nextIdx = i
			} else if cross == 0 {
				// Collinear: keep the farther point
				d1 := (nx-cx)*(nx-cx) + (ny-cy)*(ny-cy)
				d2 := (px-cx)*(px-cx) + (py-cy)*(py-cy)
				if d2 > d1 {
					nextIdx = i
				}
			}
		}
		currentIdx = nextIdx
		if currentIdx == startIdx {
			break
		}
	}

	// Scanline fill the convex hull polygon
	hn := len(hull)
	minY, maxY := hull[0].y, hull[0].y
	for _, p := range hull {
		if p.y < minY {
			minY = p.y
		}
		if p.y > maxY {
			maxY = p.y
		}
	}

	scanMin := make([]int, maxY-minY+1)
	scanMax := make([]int, maxY-minY+1)
	for i := range scanMin {
		scanMin[i] = bounds.Max.X + 1
		scanMax[i] = bounds.Min.X - 1
	}

	// Rasterize each hull edge to determine per-scanline x extents
	for i := 0; i < hn; i++ {
		p1, p2 := hull[i], hull[(i+1)%hn]
		x0, y0, x1, y1 := p1.x, p1.y, p2.x, p2.y
		if y0 > y1 {
			x0, y0, x1, y1 = x1, y1, x0, y0
		}
		if y0 == y1 {
			xLo, xHi := x0, x1
			if xLo > xHi {
				xLo, xHi = xHi, xLo
			}
			row := y0 - minY
			if xLo < scanMin[row] {
				scanMin[row] = xLo
			}
			if xHi > scanMax[row] {
				scanMax[row] = xHi
			}
			continue
		}
		for y := y0; y <= y1; y++ {
			x := x0 + (y-y0)*(x1-x0)/(y1-y0)
			row := y - minY
			if x < scanMin[row] {
				scanMin[row] = x
			}
			if x > scanMax[row] {
				scanMax[row] = x
			}
		}
	}

	for rowIdx, xMin := range scanMin {
		xMax := scanMax[rowIdx]
		y := minY + rowIdx
		for x := xMin; x <= xMax; x++ {
			off := (y-bounds.Min.Y)*result.Stride + (x-bounds.Min.X)*2
			result.Pix[off] = 0xff
			result.Pix[off+1] = 0xff
		}
	}

	return result
}
