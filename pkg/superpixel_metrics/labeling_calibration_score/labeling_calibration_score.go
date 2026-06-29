package labeling_calibration_score

import (
	"fmt"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/weighted_f1_score"
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

func (e *SpMetricDensityScore) GetScore() (labelingCalibrationScore float64, err error) {

	spResult := e.GetSpResult()
	gtSegmentation := e.GetGroundTruthSegmentation()

	// Density Score
	densityScoreEvaluator := density_score.SpMetricDensityScore{}
	densityScoreEvaluator.SetSpResult(spResult)
	densityScoreEvaluator.SetGroundTruthSegmentation(gtSegmentation)
	densityScoreEvaluator.SetParameter(density_score.MaxIslands, float64(e.maxIslands))
	densityScore, _ := densityScoreEvaluator.GetScore()

	// Weighted F1 Score
	weightedF1ScoreEvaluator := weighted_f1_score.SpMetricWeightedF1Score{}
	weightedF1ScoreEvaluator.SetSpResult(spResult)
	weightedF1ScoreEvaluator.SetGroundTruthSegmentation(gtSegmentation)
	weightedF1Score, _ := weightedF1ScoreEvaluator.GetScore()

	// Regularity Score
	regularityScoreEvaluator := regularity_score.SpMetricRegularityScore{}
	regularityScoreEvaluator.SetSpResult(spResult)
	regularityScoreEvaluator.SetGroundTruthSegmentation(gtSegmentation)
	regularityScore, _ := regularityScoreEvaluator.GetScore()

	return densityScore * weightedF1Score * regularityScore, nil
}
