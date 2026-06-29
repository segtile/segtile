package labeling_calibration_score

import (
	"fmt"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/weighted_f1_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

type SpMetricLabelingCalibrationScore struct {
	superpixel_metrics.SpMetricsData
	maxIslands int
}

func (e *SpMetricLabelingCalibrationScore) PerfectScore() float64 {
	return 1.0
}

func (e *SpMetricLabelingCalibrationScore) SetParameter(parameter superpixel_metrics.SpMetricParameter, value float64) error {
	switch parameter {
	case MaxIslands:
		e.maxIslands = int(value)
		return nil
	default:
		return fmt.Errorf("Unknown parameter for Labeling Calibration Score metric: %s", parameter)
	}
}

func (e *SpMetricLabelingCalibrationScore) GetParameter(parameter superpixel_metrics.SpMetricParameter) (float64, error) {
	switch parameter {
	case MaxIslands:
		return float64(e.maxIslands), nil
	default:
		return 0, fmt.Errorf("Unknown parameter for Labeling Calibration Score metric: %s", parameter)
	}
}

func (e *SpMetricLabelingCalibrationScore) GetSupportedParameters() []superpixel_metrics.SpMetricParameter {
	return []superpixel_metrics.SpMetricParameter{
		MaxIslands,
	}
}

func (e *SpMetricLabelingCalibrationScore) GetScore() (labelingCalibrationScore float64, err error) {

	spResult := e.GetSpResult()
	gtSegmentation := e.GetGroundTruthSegmentation()

	islandMap, islandCount := superpixel_utils.FindIslands(gtSegmentation)

	// Density Score
	densityScoreEvaluator := density_score.SpMetricDensityScore{}
	densityScoreEvaluator.SetSpResult(spResult)
	densityScoreEvaluator.SetGroundTruthSegmentation(gtSegmentation)
	densityScoreEvaluator.SetParameter(density_score.MaxIslands, float64(e.maxIslands))
	densityScore, _ := densityScoreEvaluator.GetScoreWithCachedIslands(islandMap, islandCount)

	// Weighted F1 Score
	weightedF1ScoreEvaluator := weighted_f1_score.SpMetricWeightedF1Score{}
	weightedF1ScoreEvaluator.SetSpResult(spResult)
	weightedF1ScoreEvaluator.SetGroundTruthSegmentation(gtSegmentation)
	weightedF1Score, _ := weightedF1ScoreEvaluator.GetScore()

	// Regularity Score
	regularityScoreEvaluator := regularity_score.SpMetricRegularityScore{}
	regularityScoreEvaluator.SetSpResult(spResult)
	regularityScoreEvaluator.SetGroundTruthSegmentation(gtSegmentation)
	regularityScoreEvaluator.SetParameter(regularity_score.MaxIslands, float64(e.maxIslands))
	regularityScore, _ := regularityScoreEvaluator.GetScoreWithCachedIslands(islandMap, islandCount)

	//fmt.Printf("Density Score: %f, Weighted F1 Score: %f, Regularity Score: %f\n", densityScore, weightedF1Score, regularityScore)

	return densityScore * weightedF1Score * regularityScore, nil
}
