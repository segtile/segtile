package labeling_calibration_bayesian

import (
	"fmt"
	"log"

	"github.com/d4l3k/go-bayesopt"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel/slic"
	"github.com/struffel/annotation-tool/pkg/superpixel_factory"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/weighted_f1_score"
)

type SpOptimizerLabelingCalibration struct {
	superpixel_optimizer.SpOptimizerData
	score float64
}

func (o *SpOptimizerLabelingCalibration) Run() error {
	if len(o.AnnotationPairs) == 0 {
		return fmt.Errorf("No image and ground truth paths provided for optimization.")
	}

	// Prepare the parameters for the optimizer.
	bayesParameters := []bayesopt.Param{}

	/*for _, param := range spImpl.GetSupportedParameters() {
		bayesParameters = append(bayesParameters, bayesopt.UniformParam{
			Name: string(param),
			Min:  param.Min(),
			Max:  param.Max(),
		})
	}*/

	// Temporary: Manually set parameters
	bayesParameters = append(bayesParameters, bayesopt.UniformParam{
		Name: string(slic.Count),
		Min:  slic.Count.Min(),
		Max:  slic.Count.Max(),
	})
	bayesParameters = append(bayesParameters, bayesopt.UniformParam{
		Name: string(slic.Compactness),
		Min:  slic.Compactness.Min(),
		Max:  slic.Compactness.Max(),
	})

	// Create the optimizer with the parameters.
	optimizer := bayesopt.New(
		bayesParameters,
		bayesopt.WithMinimize(false),
		bayesopt.WithRandomRounds(16),
		bayesopt.WithRounds(32),
	)

	parameterMap, score, err := optimizer.Optimize(func(params map[bayesopt.Param]float64) float64 {
		result := evaluate(o.SpImplementation, o.AnnotationPairs, params)
		log.Printf("Round %d: score=%.4f, params=%v", optimizer.Rounds(), result, params)
		return result
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("Optimization complete: best score=%.4f, params=%v", score, parameterMap)
	return nil
}

func evaluate(impl superpixel.SpImplementation, annotationPairs []superpixel_optimizer.SpOptimizerImagePair, params map[bayesopt.Param]float64) float64 {
	if len(annotationPairs) == 0 {
		return 0
	}

	implName, err := superpixel_factory.ImplementationName(impl)
	if err != nil {
		return 0
	}

	totalScore := 0.0
	for _, sample := range annotationPairs {
		currentImpl, err := superpixel_factory.NewSuperpixelImplementationByName(implName)
		if err != nil {
			continue
		}
		currentImg := io_utils.LoadImage(sample.ImagePath)
		currentGt := io_utils.ToGray16(io_utils.LoadImage(sample.GroundTruthPath))
		currentImpl.SetImage(currentImg)
		currentImpl.SetRecommendedParameters()

		// Apply the candidate parameters
		for param, value := range params {
			currentImpl.SetParameterByName(param.GetName(), value)
		}

		if err := currentImpl.Run(); err != nil {
			continue
		}

		// Density Score
		densityScoreEvaluator := density_score.SpMetricDensityScore{}
		densityScoreEvaluator.SetSpResult(currentImpl.GetResult())
		densityScoreEvaluator.SetGroundTruthSegmentation(currentGt)
		densityScoreEvaluator.SetParameter(density_score.MaxIslands, 5)
		densityScore, _ := densityScoreEvaluator.GetScore()

		// Weighted F1 Score
		weightedF1ScoreEvaluator := weighted_f1_score.SpMetricWeightedF1Score{}
		weightedF1ScoreEvaluator.SetSpResult(currentImpl.GetResult())
		weightedF1ScoreEvaluator.SetGroundTruthSegmentation(currentGt)
		weightedF1Score, _ := weightedF1ScoreEvaluator.GetScore()

		// Regularity Score
		regularityScoreEvaluator := regularity_score.SpMetricRegularityScore{}
		regularityScoreEvaluator.SetSpResult(currentImpl.GetResult())
		regularityScoreEvaluator.SetGroundTruthSegmentation(currentGt)
		regularityScore, _ := regularityScoreEvaluator.GetScore()

		score := densityScore * weightedF1Score * regularityScore

		//Sum up
		totalScore += score
	}

	return totalScore / float64(len(annotationPairs))
}
