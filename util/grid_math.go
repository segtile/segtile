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

	// Calculate the "over/undershoot" that the equal grid would produce
	xError := gridInterval*columns - width
	yError := gridInterval*rows - height

	// If the error is positive, we are clipping out of the image, so remove a column
	if xError > 0 {
		columns--
		xError = width - gridInterval*columns
	}

	if yError > 0 {
		rows--
		yError = height - gridInterval*rows
	}

	// Offset the grid
	xOffset := -xError / 2
	yOffset := -yError / 2

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
