package density_score_2

import (
	"fmt"
	"image"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

func init() {
	parameterInfo := []superpixel_metrics.SpMetricParameterInfo{
		{
			Id:          "max_islands",
			Kind:        superpixel_metrics.SpMetricParameterKindInt,
			Title:       "Max Islands",
			Description: "Maximum number of islands to consider for the density score calculation. If set to 0, all islands will be considered.",
			Default:     5,
			Min:         0,
			Max:         math.MaxInt,
		},
	}
	superpixel_metrics.RegisterSpMetric(
		"density_score_2",
		"Density Score",
		"Computes the density score of the superpixel segmentation.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth, uint16(params["max_islands"]))
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16, maxIslands uint16) (densityCalibrationScore float64, err error) {

	// Check that images are valid
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	// Find islands in GT
	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruth)

	// Prepare metrics
	cumulativeDensityScore := 0.0
	pixelCount := groundTruth.Bounds().Dx() * groundTruth.Bounds().Dy()

	// Choose the number of islands to evaluate
	if maxIslands <= 0 || maxIslands > groundTruthIslandCount {
		maxIslands = groundTruthIslandCount
	}

	// Return early
	if maxIslands == 0 {
		return 0.0, nil
	}

	// Evaluate islands
	for i := uint16(1); i <= maxIslands; i++ {

		// Make a selection of the current island and get its area
		currentGroundTruthIsland := superpixel_utils.SelectIsland(groundTruthIslands, i)
		currentGroundTruthIslandArea := superpixel_utils.Area(currentGroundTruthIsland)

		// Get the desired ratio for the current island
		desiredOutsideRatio := 1 - math.Pow(float64(currentGroundTruthIslandArea)/float64(pixelCount), 0.25)

		// Count inside and outside boundaries for the current island approximation
		labelMapBoundaries := superpixel_utils.BoundaryPoints(result.LabelMap, false)
		selectionFromCurrentIsland := result.SelectFromMask(currentGroundTruthIsland, 0.5)
		currentApproximatedMask := result.SelectionToMask(selectionFromCurrentIsland)
		boundariesInside := 0
		boundariesOutside := 0

		for _, boundaryPoint := range labelMapBoundaries {

			// Is the boundary in the approximated selection?
			// (Skip if not)
			if currentApproximatedMask.Gray16At(boundaryPoint.X, boundaryPoint.Y).Y == 0 {
				continue
			}

			// Does it have a neighbor pixel not in the selection?
			hasNeighborOutsideSelection := false
			for _, neighbor := range superpixel_utils.Neighbors(boundaryPoint, result.LabelMap.Bounds(), true) {
				if currentApproximatedMask.Gray16At(neighbor.X, neighbor.Y).Y == 0 {
					hasNeighborOutsideSelection = true
					break
				}
			}

			if hasNeighborOutsideSelection {
				boundariesOutside += 2
			} else {
				boundariesInside++
			}
		}

		if boundariesInside+boundariesOutside == 0 {
			continue
		}

		outsideRatio := float64(boundariesOutside) / float64(boundariesInside+boundariesOutside)
		cumulativeDensityScore += math.Min(outsideRatio, desiredOutsideRatio) / math.Max(outsideRatio, desiredOutsideRatio)
	}

	densityCalibrationScore = cumulativeDensityScore / float64(maxIslands)

	return densityCalibrationScore, nil
}
