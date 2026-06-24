package labeling_calibration

import (
	"errors"

	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/f1_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_optimization"
)

type SpOptimizerLabelingCalibration struct {
	superpixel_optimization.SpOptimizerData
	score                      float64
	densityCalibrationScore    float64
	f1Score                    float64
	regularityCalibrationScore float64
}

func (o *SpOptimizerLabelingCalibration) Run() error {
	if len(o.Paths) == 0 {
		return errors.New("No image and ground truth paths provided for optimization.")
	}

	// Density
	densityEvaluator := density_score.SpMetricDensityScore{}
	densityEvaluator.SetSpResult(o.Implementation.GetResult())
	o.densityCalibrationScore = densityEvaluator.GetScore()

	// F1 Score
	f1Evaluator := f1_score.SpMetricF1Score{}
	f1Evaluator.SetSpResult(o.Implementation.GetResult())
	o.f1Score = f1Evaluator.GetScore()

	// Regularity
	regularityEvaluator := regularity_score.SpMetricRegularityScore{}
	regularityEvaluator.SetSpResult(o.Implementation.GetResult())
	o.regularityCalibrationScore = regularityEvaluator.GetScore()

	o.score = o.densityCalibrationScore * o.f1Score * o.regularityCalibrationScore

	return nil
}
