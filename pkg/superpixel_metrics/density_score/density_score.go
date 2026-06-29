package density_score

import (
	"fmt"
	"math"
	"math/rand/v2"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpMetricDensityScore struct {
	superpixel_metrics.SpMetricsData
	maxIslands int
}

func (e *SpMetricDensityScore) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricDensityScore) SetParameter(parameter superpixel_metrics.SpMetricParameter, value float64) error {
	switch parameter {
	case MaxIslands:
		e.maxIslands = int(value)
		return nil
	default:
		return fmt.Errorf("Unknown parameter for DensityScore metric: %s", parameter)
	}
}

func (e *SpMetricDensityScore) GetParameter(parameter superpixel_metrics.SpMetricParameter) (float64, error) {
	switch parameter {
	case MaxIslands:
		return float64(e.maxIslands), nil
	default:
		return 0, fmt.Errorf("Unknown parameter for DensityScore metric: %s", parameter)
	}
}

func (e *SpMetricDensityScore) GetSupportedParameters() []superpixel_metrics.SpMetricParameter {
	return []superpixel_metrics.SpMetricParameter{
		MaxIslands,
	}
}

func (e *SpMetricDensityScore) GetScore() (densityCalibrationScore float64, err error) {

	groundTruthIslands, groundTruthIslandCount := superpixel_utils.FindIslands(e.GetGroundTruthSegmentation())

	// Prepare metrics
	densityCalibrationScore = 0.0

	// Build a list of all island labels
	islandLabels := make([]uint16, groundTruthIslandCount)
	for i := uint16(0); i < groundTruthIslandCount; i++ {
		islandLabels[i] = i + 1
	}

	// Shuffle the island labels and pick up to N for evaluation
	rand.Shuffle(len(islandLabels), func(i, j int) {
		islandLabels[i], islandLabels[j] = islandLabels[j], islandLabels[i]
	})
	if len(islandLabels) > e.maxIslands {
		islandLabels = islandLabels[:e.maxIslands]
	}

	groundTruthMaskArea := e.GetGroundTruthSegmentation().Bounds().Dx() * e.GetGroundTruthSegmentation().Bounds().Dy()

	for _, islandLabel := range islandLabels {

		currentGroundTruthIsland := superpixel_utils.SelectIsland(groundTruthIslands, islandLabel)
		currentGroundTruthIslandArea := superpixel_metrics.Area(currentGroundTruthIsland)

		selectionFromCurrentIsland := e.GetSpResult().SelectFromMask(currentGroundTruthIsland, 0.5)
		currentApproximatedMask := e.GetSpResult().SelectionToMask(selectionFromCurrentIsland)
		currentApproximatedMaskArea := superpixel_metrics.Area(currentApproximatedMask)

		_, currentSelectedBorderingSpCount := superpixel_utils.CountBorderingSuperpixels(e.GetSpResult().LabelMap, selectionFromCurrentIsland)

		currentBorderingSpSelection := superpixel_utils.SelectBorderingSuperpixels(e.GetSpResult().LabelMap, selectionFromCurrentIsland)
		currentBorderingSpMask := e.GetSpResult().SelectionToMask(currentBorderingSpSelection)
		currentBorderingSpMaskArea := superpixel_metrics.Area(currentBorderingSpMask)

		// Density Calibration Score
		if currentSelectedBorderingSpCount > 0 {
			borderPortionOfSelection := float64(currentBorderingSpMaskArea) / float64(currentApproximatedMaskArea)
			desiredBorderPortionOfSelection := math.Pow(1-(float64(currentGroundTruthIslandArea)/float64(groundTruthMaskArea)), 2)
			densityCalibrationScore += math.Min(borderPortionOfSelection, desiredBorderPortionOfSelection) / math.Max(borderPortionOfSelection, desiredBorderPortionOfSelection)
		}

	}

	densityCalibrationScore /= float64(len(islandLabels))

	return densityCalibrationScore, nil
}
