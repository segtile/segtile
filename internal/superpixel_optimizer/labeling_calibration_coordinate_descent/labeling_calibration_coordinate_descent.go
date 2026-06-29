package labeling_calibration_coordinate_descent

import (
	"fmt"
	"log"
	"math"
	"sync"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_factory"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/labeling_calibration_score"
)

type SpOptimizerLabelingCalibrationCoordinateDescent struct {
	superpixel_optimizer.SpOptimizerData
	score      float64
	bestParams map[superpixel.SpParameter]float64
}

func (o *SpOptimizerLabelingCalibrationCoordinateDescent) Run() error {
	if len(o.AnnotationPairs) == 0 {
		return fmt.Errorf("No image and ground truth paths provided for optimization.")
	}

	// Resolve the implementation name so we can create fresh copies during evaluation.
	implName, err := superpixel_factory.ImplementationName(o.SpImplementation)
	if err != nil {
		return fmt.Errorf("failed to get implementation name: %w", err)
	}

	// Use SetRecommendedParameters() as the starting point.
	tempImpl, err := superpixel_factory.NewSuperpixelImplementationByName(implName)
	if err != nil {
		return fmt.Errorf("failed to create superpixel implementation: %w", err)
	}
	tempImpl.SetRecommendedParameters()

	supportedParams := o.OptimizationTargetParameters
	initialParams := make(map[superpixel.SpParameter]float64)
	for _, param := range supportedParams {
		if v, err := tempImpl.GetParameter(param); err == nil {
			initialParams[param] = v
		}
	}

	bestParams := copyParams(initialParams)
	bestScore := evaluate(o.SpImplementation, o.AnnotationPairs, bestParams)
	log.Printf("Initial score: %.4f, params: %v", bestScore, bestParams)

	const maxPasses = 8

	for pass := 0; pass < maxPasses; pass++ {
		improved := false

		for _, param := range supportedParams {
			// Evaluate candidate values for this parameter while holding all others fixed.
			for _, candidate := range probeValues(bestParams[param], initialParams[param]) {
				if candidate == bestParams[param] {
					continue
				}
				trialParams := copyParams(bestParams)
				trialParams[param] = candidate

				if score := evaluate(o.SpImplementation, o.AnnotationPairs, trialParams); score > bestScore {
					bestScore = score
					bestParams[param] = candidate
					improved = true
					log.Printf("New best score: %.4f with param %s=%.4f", bestScore, param, candidate)
				} else {
					log.Printf("Trial score: %.4f with param %s=%.4f", score, param, candidate)
				}

			}
			log.Printf("Pass %d, param %s: best score=%.4f, best value=%.4f", pass+1, param, bestScore, bestParams[param])
		}

		log.Printf("Pass %d: score=%.4f, params=%v", pass+1, bestScore, bestParams)
		if !improved {
			break
		}
	}

	o.score = bestScore
	o.bestParams = bestParams
	return nil
}

func (o *SpOptimizerLabelingCalibrationCoordinateDescent) GetResults() (float64, map[superpixel.SpParameter]float64, error) {
	if o.bestParams == nil {
		return 0, nil, fmt.Errorf("optimization has not been run yet")
	}
	return o.score, o.bestParams, nil
}

// probeValues generates candidate values for a single parameter.
// It samples an exponential grid anchored at initialValue and finer steps around currentValue.
func probeValues(currentValue, initialValue float64) []float64 {
	base := initialValue
	if base == 0 {
		base = 1
	}

	seen := make(map[float64]bool)
	var values []float64

	add := func(v float64) {
		if v > 0 && !seen[v] {
			seen[v] = true
			values = append(values, v)
		}
	}

	// Exponential grid: base * 2^exp for exp in [-4, 4]
	for exp := -4; exp <= 4; exp++ {
		add(base * math.Pow(2, float64(exp)))
	}

	// Finer probing around the current best value
	if currentValue > 0 {
		for _, factor := range []float64{0.5, 0.75, 0.875, 1.0, 1.125, 1.25, 1.5, 2.0} {
			add(currentValue * factor)
		}
	}

	return values
}

func copyParams(params map[superpixel.SpParameter]float64) map[superpixel.SpParameter]float64 {
	result := make(map[superpixel.SpParameter]float64, len(params))
	for k, v := range params {
		result[k] = v
	}
	return result
}

func evaluate(impl superpixel.SpImplementation, annotationPairs []superpixel_optimizer.SpOptimizerImagePair, params map[superpixel.SpParameter]float64) float64 {
	if len(annotationPairs) == 0 {
		return 0
	}

	implName, err := superpixel_factory.ImplementationName(impl)
	if err != nil {
		return 0
	}

	scores := make(chan float64, len(annotationPairs))
	var wg sync.WaitGroup

	for _, sample := range annotationPairs {
		wg.Add(1)
		go func(sample superpixel_optimizer.SpOptimizerImagePair) {
			defer wg.Done()
			currentImpl, err := superpixel_factory.NewSuperpixelImplementationByName(implName)
			if err != nil {
				return
			}
			currentImg := io_utils.LoadImage(sample.ImagePath)
			currentGt := io_utils.ToGray16(io_utils.LoadImage(sample.GroundTruthPath))
			currentImpl.SetImage(currentImg)
			currentImpl.SetRecommendedParameters()

			// Apply the candidate parameters
			for param, value := range params {
				currentImpl.SetParameter(param, value)
			}

			if err := currentImpl.Run(); err != nil {
				return
			}

			// Labeling Calibration Score
			labelingCalibrationEvaluator := labeling_calibration_score.SpMetricLabelingCalibrationScore{}
			labelingCalibrationEvaluator.SetSpResult(currentImpl.GetResult())
			labelingCalibrationEvaluator.SetGroundTruthSegmentation(currentGt)
			labelingCalibrationEvaluator.SetParameter(labeling_calibration_score.MaxIslands, 5)
			labelingCalibrationScore, _ := labelingCalibrationEvaluator.GetScore()

			scores <- labelingCalibrationScore
		}(sample)
	}

	wg.Wait()
	close(scores)

	totalScore := 0.0
	for score := range scores {
		totalScore += score
	}

	return totalScore / float64(len(annotationPairs))
}
