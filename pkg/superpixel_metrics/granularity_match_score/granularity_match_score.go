package granularity_match_score

import (
	"fmt"
	"image"
	"math"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"
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
		"granularity_match_score",
		"Granularity Match Score",
		"Computes the granularity match score of the superpixel segmentation.",
		1.0,
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error) {
	return Run(result, groundTruth)
}

func Run(result superpixel.SpResult, groundTruth *superpixel_map.Mask) (densityCalibrationScore float64, err error) {

	// Check that input images match
	if result.LabelMap.Bounds() != groundTruth.Bounds() {
		return 0, fmt.Errorf("Ground truth and superpixel label map must have the same dimensions")
	}

	bounds := groundTruth.Bounds()
	pixelCount := bounds.Dx() * bounds.Dy()

	// Find islands in GT
	gtComponentIdMap, gtComponentValueIndex, gtComponentSize := superpixel_utils.FindConnectedComponents(groundTruth)

	gtComponentCount := len(gtComponentSize)
	gtForegroundComponentCount := 0
	for _, value := range gtComponentValueIndex {
		if value > 0 {
			gtForegroundComponentCount++
		}
	}
	if gtForegroundComponentCount == 0 {
		return 0, nil
	}

	// Count the number of pixels in each island
	pixelCountPerIsland := make([]int, gtComponentCount+1)

	// Count the number of pixels per superpixel
	pixelCountPerSuperpixel := make([]int, result.SpCount)

	// For each superpixel, keep track of how many pixels overlap with each island
	pixelCountPerSuperpixelPerIsland := make([]map[uint32]int, result.SpCount)

	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			islandIndex := gtComponentIdMap.ValueAt(x, y)
			islandValue := gtComponentValueIndex[islandIndex-1]
			spLabel := result.LabelMap.ValueAt(x, y)

			pixelCountPerSuperpixel[spLabel]++
			pixelCountPerIsland[islandIndex]++

			if islandValue > 0 {
				if pixelCountPerSuperpixelPerIsland[spLabel] == nil {
					pixelCountPerSuperpixelPerIsland[spLabel] = make(map[uint32]int)
				}
				pixelCountPerSuperpixelPerIsland[spLabel][islandIndex]++
			}
		}
	}

	// Determine which island (if any) "owns" each superpixel. Because islands are disjoint, a superpixel's overlap fractions across islands sum to
	// at most 1, so at most one island can reach the 50% threshold.
	const selectionThreshold = 0.5
	owningIsland := make([]uint32, result.SpCount)
	for spLabel, overlaps := range pixelCountPerSuperpixelPerIsland {
		for islandLabel, overlapCount := range overlaps {
			if float64(overlapCount)/float64(pixelCountPerSuperpixel[spLabel]) >= selectionThreshold {
				owningIsland[spLabel] = islandLabel
				break
			}
		}
	}

	// Classify every boundary pixel of the label map exactly once, attributing it to whichever
	// island owns its superpixel (if any), and tally inside/outside boundary counts per island.
	boundariesInside := make([]int, gtComponentCount+1)
	boundariesOutside := make([]int, gtComponentCount+1)

	for _, boundaryPoint := range superpixel_utils.BoundaryPoints(result.LabelMap, false) {
		owner := owningIsland[result.LabelMap.ValueAt(boundaryPoint.X, boundaryPoint.Y)]
		if owner == 0 {
			continue
		}

		// Does it have a neighbor pixel not in the same island's selection?
		hasNeighborOutsideSelection := false
		for _, neighbor := range superpixel_utils.PixelNeighbours(boundaryPoint, result.LabelMap.Bounds(), false, false) {
			neighborSpLabel := result.LabelMap.ValueAt(neighbor.X, neighbor.Y)
			if owningIsland[neighborSpLabel] != owner {
				hasNeighborOutsideSelection = true
				break
			}
		}

		if hasNeighborOutsideSelection {
			boundariesOutside[owner]++
		} else {
			boundariesInside[owner]++
		}
	}

	// Aggregate the per-island scores
	cumulativeDensityScore := 0.0
	for i := uint32(1); i <= uint32(gtComponentCount); i++ {
		if gtComponentValueIndex[i-1] == 0 {
			continue
		}

		inside := boundariesInside[i]
		outside := boundariesOutside[i]
		if inside+outside == 0 {
			continue
		}

		desiredOutsideRatio := 1 - math.Pow(float64(pixelCountPerIsland[i])/float64(pixelCount), 0.25)
		outsideRatio := float64(outside) / float64(inside+outside)
		cumulativeDensityScore += math.Min(outsideRatio, desiredOutsideRatio) / math.Max(outsideRatio, desiredOutsideRatio)
	}

	densityCalibrationScore = cumulativeDensityScore / float64(gtForegroundComponentCount)

	return densityCalibrationScore, nil
}
