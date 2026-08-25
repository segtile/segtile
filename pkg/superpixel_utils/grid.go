package superpixel_utils

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

// HexagonalGridPoints generates a regular hexagonal lattice of points trying to hit the target number of points.
func HexagonalGridPoints(width int, height int, targetNumPoints int) (points []image.Point, gridInterval int) {
	numPixels := width * height
	if width <= 0 || height <= 0 || targetNumPoints < 1 {
		return nil, 0
	}

	// Target cell area: for a regular hexagon, area = (sqrt(3)/2) * S^2
	targetArea := float64(numPixels) / float64(targetNumPoints)
	spacing := math.Sqrt(targetArea * (2.0 / math.Sqrt(3.0)))
	gridInterval = int(spacing + 0.5)
	if gridInterval < 1 {
		gridInterval = 1
	}

	deltaX := float64(gridInterval)
	deltaY := deltaX * (math.Sqrt(3.0) / 2.0)

	columns := int(0.5 + float64(width)/deltaX)
	rows := int(0.5 + float64(height)/deltaY)

	if columns < 1 {
		columns = 1
	}
	if rows < 1 {
		rows = 1
	}

	if int(float64(columns)*deltaX) > width {
		columns--
		if columns < 1 {
			columns = 1
		}
	}
	if int(float64(rows)*deltaY) > height {
		rows--
		if rows < 1 {
			rows = 1
		}
	}

	xOffset := int((float64(width) - float64(columns)*deltaX) / 2.0)
	yOffset := int((float64(height) - float64(rows)*deltaY) / 2.0)

	points = make([]image.Point, 0, columns*rows)
	for r := 0; r < rows; r++ {
		rowY := int(float64(r)*deltaY+deltaY/2.0) + yOffset
		if rowY < 0 {
			rowY = 0
		}
		if rowY >= height {
			rowY = height - 1
		}
		for c := 0; c < columns; c++ {
			rowOffsetX := 0.0
			if r%2 == 1 {
				rowOffsetX = deltaX / 2.0
			}
			colX := int(float64(c)*deltaX+deltaX/2.0+rowOffsetX) + xOffset
			if colX < 0 {
				colX = 0
			}
			if colX >= width {
				colX = width - 1
			}
			points = append(points, image.Point{X: colX, Y: rowY})
		}
	}

	return points, gridInterval
}
