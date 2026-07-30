package density_score

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
		"density_score",
		"Density Score",
		"Computes the density score of the superpixel segmentation.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth)
}

func Run(result superpixel.SpResult, groundTruth *image.Gray16) (densityCalibrationScore float64, err error) {

	// Check that input images match
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	bounds := groundTruth.Bounds()
	pixelCount := bounds.Dx() * bounds.Dy()

	// Find islands in GT
	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(groundTruth)

	// Count the number of pixels in each island
	pixelCountPerIsland := make([]int, groundTruthIslandCount+1)

	// Count the number of pixels per superpixel
	pixelCountPerSuperpixel := make([]int, result.SpCount)

	// For each superpixel, keep track of how many pixels overlap with each island
	spIslandOverlap := make([]map[uint16]int, result.SpCount)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			islandIndex := groundTruthIslands.Gray16At(x, y).Y
			spLabel := result.LabelMap.Gray16At(x, y).Y

			pixelCountPerSuperpixel[spLabel]++
			pixelCountPerIsland[islandIndex]++

			if islandIndex > 0 {
				if spIslandOverlap[spLabel] == nil {
					spIslandOverlap[spLabel] = make(map[uint16]int)
				}
				spIslandOverlap[spLabel][islandIndex]++
			}
		}
	}

	// Determine which island (if any) "owns" each superpixel. Because islands are disjoint, a superpixel's overlap fractions across islands sum to
	// at most 1, so at most one island can reach the 50% threshold.
	const selectionThreshold = 0.5
	owningIsland := make([]uint16, result.SpCount)
	for spLabel, overlaps := range spIslandOverlap {
		for islandLabel, overlapCount := range overlaps {
			if float64(overlapCount)/float64(pixelCountPerSuperpixel[spLabel]) >= selectionThreshold {
				owningIsland[spLabel] = islandLabel
				break
			}
		}
	}

	// Classify every boundary pixel of the label map exactly once, attributing it to whichever
	// island owns its superpixel (if any), and tally inside/outside boundary counts per island.
	boundariesInside := make([]int, groundTruthIslandCount+1)
	boundariesOutside := make([]int, groundTruthIslandCount+1)

	for _, boundaryPoint := range superpixel_utils.BoundaryPoints(result.LabelMap, false) {
		owner := owningIsland[result.LabelMap.Gray16At(boundaryPoint.X, boundaryPoint.Y).Y]
		if owner == 0 {
			continue
		}

		// Does it have a neighbor pixel not in the same island's selection?
		hasNeighborOutsideSelection := false
		for _, neighbor := range superpixel_utils.Neighbors(boundaryPoint, result.LabelMap.Bounds(), true) {
			neighborSpLabel := result.LabelMap.Gray16At(neighbor.X, neighbor.Y).Y
			if owningIsland[neighborSpLabel] != owner {
				hasNeighborOutsideSelection = true
				break
			}
		}

		if hasNeighborOutsideSelection {
			boundariesOutside[owner] += 2
		} else {
			boundariesInside[owner]++
		}
	}

	// Aggregate the per-island scores
	cumulativeDensityScore := 0.0
	for i := uint16(1); i <= groundTruthIslandCount; i++ {
		inside := boundariesInside[i]
		outside := boundariesOutside[i]
		if inside+outside == 0 {
			continue
		}

		desiredOutsideRatio := 1 - math.Pow(float64(pixelCountPerIsland[i])/float64(pixelCount), 0.25)
		outsideRatio := float64(outside) / float64(inside+outside)
		cumulativeDensityScore += math.Min(outsideRatio, desiredOutsideRatio) / math.Max(outsideRatio, desiredOutsideRatio)
	}

	densityCalibrationScore = cumulativeDensityScore / float64(groundTruthIslandCount)

	return densityCalibrationScore, nil
}
