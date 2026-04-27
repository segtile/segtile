package util

import (
	"image"
	"math"
)

// This function tries to fill the given area with a regular grid of points.
func RegularGridPoints(width int, height int, targetNumPoints int) (points []image.Point, gridInterval int) {

	// Build a square grid, trying to hit the target number of points
	numPixels := width * height
	gridInterval = int(math.Sqrt(float64(numPixels)/float64(targetNumPoints)) + 0.5)

	columns := int(0.5 + float64(width)/float64(gridInterval))
	rows := int(0.5 + float64(height)/float64(gridInterval))

	// If the grid overflows the image, remove a column/row
	if gridInterval*columns > width {
		columns--
	}
	if gridInterval*rows > height {
		rows--
	}

	// Recompute the undershoot after adjustments (always <= 0: grid is shorter than image)
	// and offset the grid to center it within the image.
	xOffset := (width - gridInterval*columns) / 2
	yOffset := (height - gridInterval*rows) / 2

	// Calculate the final positions
	points = make([]image.Point, columns*rows)
	for iX := 0; iX < columns; iX++ {
		for iY := 0; iY < rows; iY++ {
			points[iX*rows+iY] = image.Point{
				X: iX*gridInterval + xOffset,
				Y: iY*gridInterval + yOffset,
			}
		}
	}

	return points, gridInterval

}
