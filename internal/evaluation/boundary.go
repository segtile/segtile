package evaluation

import (
	"fmt"
	"image"
	"image/color"
	"math"
)

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

func DistanceToNextBoundaryPixel(mask *image.Gray16) *image.Gray16 {
	bounds := mask.Bounds()
	w := bounds.Dx()
	h := bounds.Dy()

	const inf = int(^uint(0) >> 1)
	dist := make([]int, w*h)
	for i := range dist {
		dist[i] = inf
	}

	type point struct{ x, y int }
	queue := make([]point, 0, w*h)

	// Seed the BFS with all non-zero pixels that have at least one zero neighbour (boundary pixels).
	neighbors := [4][2]int{{1, 0}, {-1, 0}, {0, 1}, {0, -1}}
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.Gray16At(x, y).Y == 0 {
				continue
			}
			isBoundary := false
			for _, d := range neighbors {
				nx, ny := x+d[0], y+d[1]
				if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y || mask.Gray16At(nx, ny).Y == 0 {
					isBoundary = true
					break
				}
			}
			if isBoundary {
				idx := (y-bounds.Min.Y)*w + (x - bounds.Min.X)
				dist[idx] = 0
				queue = append(queue, point{x, y})
			}
		}
	}

	// BFS propagation — each step increments distance by 1 (Chebyshev-like; use squared for Euclidean approximation).
	// For exact squared-Euclidean use a two-pass approach; BFS gives L∞ (Chebyshev) distance which is a fast, tight approximation.
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		curIdx := (cur.y-bounds.Min.Y)*w + (cur.x - bounds.Min.X)
		curDist := dist[curIdx]
		for _, d := range neighbors {
			nx, ny := cur.x+d[0], cur.y+d[1]
			if nx < bounds.Min.X || nx >= bounds.Max.X || ny < bounds.Min.Y || ny >= bounds.Max.Y {
				continue
			}
			if mask.Gray16At(nx, ny).Y == 0 {
				continue
			}
			nIdx := (ny-bounds.Min.Y)*w + (nx - bounds.Min.X)
			if nd := curDist + 1; nd < dist[nIdx] {
				dist[nIdx] = nd
				queue = append(queue, point{nx, ny})
			}
		}
	}

	// Write results to output image.
	distanceMap := image.NewGray16(bounds)
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			if mask.Gray16At(x, y).Y > 0 {
				idx := (y-bounds.Min.Y)*w + (x - bounds.Min.X)
				d := dist[idx]
				if d > 65535 {
					d = 65535
				}
				distanceMap.SetGray16(x, y, color.Gray16{Y: uint16(d)})
			}
		}
	}

	return distanceMap
}
