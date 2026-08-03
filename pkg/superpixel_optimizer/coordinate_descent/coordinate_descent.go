package coordinate_descent

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/struffel/annotation-tool/pkg/progress"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_optimizer"
)

func init() {
	parameterInfo := []superpixel_optimizer.SpOptimizerParameterInfo{
		{
			Id:          "passes",
			Kind:        superpixel_optimizer.SpOptimizerParameterKindInt,
			Title:       "Passes",
			Description: "Number of passes to perform over the parameters. Each pass will attempt to optimize each parameter in turn.",
			Default:     8,
			Min:         1,
			Max:         math.MaxInt,
		},
	}
	superpixel_optimizer.RegisterSpOptimizer(
		"coordinate_descent",
		"Coordinate Descent Optimizer",
		"Optimizes superpixel algorithm parameters using a coordinate descent approach.",
		runWithParams,
		parameterInfo,
	)
}

func runWithParams(
	ctx context.Context,
	imagePairs []superpixel_optimizer.SpOptimizerInputPair,
	metric string,
	algorithm string,
	algorithmParameterOptimizationTargets []string,
	algorithmParameterBaseValues map[string]float64,
	optimizerParams map[string]float64,
	progress progress.ProgressIndicator) (superpixel_optimizer.SpOptimizerResult, error) {
	result, err := Run(ctx, imagePairs, metric, algorithm, algorithmParameterOptimizationTargets, algorithmParameterBaseValues, int(optimizerParams["passes"]), progress)
	if err != nil {
		return superpixel_optimizer.SpOptimizerResult{}, err
	}
	return superpixel_optimizer.SpOptimizerResult{
		Score:      result.Score,
		Algorithm:  result.Algorithm,
		Parameters: result.Parameters,
	}, nil
}

func Run(ctx context.Context, imagePairs []superpixel_optimizer.SpOptimizerInputPair, metric string, algorithm string, algorithmParameterOptimizationTargets []string, algorithmParameterBaseValues map[string]float64, passes int, progressIndicator progress.ProgressIndicator) (result superpixel_optimizer.SpOptimizerResult, err error) {

	if len(imagePairs) == 0 {
		return superpixel_optimizer.SpOptimizerResult{}, fmt.Errorf("No image and ground truth paths provided for optimization.")
	}

	allAlgorithms := superpixel.ListAlgorithms()
	if !contains(allAlgorithms, algorithm) {
		return superpixel_optimizer.SpOptimizerResult{}, fmt.Errorf("Unknown superpixel algorithm %s", algorithm)
	}

	// Ensure that all target parameters are valid for the algorithm
	defaultParams, err := superpixel.GetDefaultParameterValues(algorithm)
	if err != nil {
		return superpixel_optimizer.SpOptimizerResult{}, err
	}
	for _, target := range algorithmParameterOptimizationTargets {
		if _, exists := defaultParams[target]; !exists {
			return superpixel_optimizer.SpOptimizerResult{}, fmt.Errorf("Unknown parameter %s for algorithm %s", target, algorithm)
		}
	}

	allMetrics := superpixel_metrics.ListSpMetrics()
	if !contains(allMetrics, metric) {
		return superpixel_optimizer.SpOptimizerResult{}, fmt.Errorf("Unknown superpixel metric %s", metric)
	}

	bestScore := -math.MaxFloat64
	var bestAlgorithm string
	var bestScoreAlgorithmParams map[string]float64

	if err := ctx.Err(); err != nil {
		return superpixel_optimizer.SpOptimizerResult{}, err
	}
	if err := progressIndicator.Update(fmt.Sprintf("Optimizing %s", algorithm), 0); err != nil {
		return superpixel_optimizer.SpOptimizerResult{}, err
	}
	score, params, err := optimizeAlgorithm(ctx, imagePairs, metric, algorithm, algorithmParameterOptimizationTargets, algorithmParameterBaseValues, passes, progressIndicator)
	if err != nil {
		return superpixel_optimizer.SpOptimizerResult{}, err
	}
	if score > bestScore {
		bestScore = score
		bestAlgorithm = algorithm
		bestScoreAlgorithmParams = params
	}

	return superpixel_optimizer.SpOptimizerResult{
		Score:      bestScore,
		Algorithm:  bestAlgorithm,
		Parameters: bestScoreAlgorithmParams,
	}, nil
}

func optimizeAlgorithm(
	ctx context.Context,
	imagePairs []superpixel_optimizer.SpOptimizerInputPair,
	metric string,
	algorithm string,
	targetParams []string,
	fixedParams map[string]float64,
	passes int,
	progressIndicator progress.ProgressIndicator) (float64, map[string]float64, error) {

	// Get default parameters for the algorithm and set all the given fixed params
	baseParams, err := superpixel.GetDefaultParameterValues(algorithm)
	if err != nil {
		return 0, nil, err
	}
	for k, v := range fixedParams {
		baseParams[k] = v
	}

	// Sort param names for deterministic iteration order.
	paramNames := make([]string, 0, len(targetParams))
	for _, k := range targetParams {
		paramNames = append(paramNames, k)
	}
	sort.Strings(paramNames)

	bestParams := copyParams(baseParams)
	bestScore, err := evaluate(ctx, algorithm, metric, imagePairs, bestParams)
	if err != nil {
		return 0, nil, err
	}
	log.Printf("Initial score: %.4f, params: %v", bestScore, bestParams)
	completedTrials := 1
	estimatedTrials := 1 + passes*len(paramNames)*17

	for pass := 0; pass < passes; pass++ {
		if err := ctx.Err(); err != nil {
			return 0, nil, err
		}
		improved := false

		for _, paramName := range paramNames {
			for _, candidate := range probeValues(bestParams[paramName], baseParams[paramName]) {
				if err := ctx.Err(); err != nil {
					return 0, nil, err
				}
				if candidate == bestParams[paramName] {
					continue
				}
				trialParams := copyParams(bestParams)
				trialParams[paramName] = candidate

				completedTrials++
				percentComplete := min(99, completedTrials*100/estimatedTrials)
				if err := progressIndicator.Update(fmt.Sprintf("Pass %d, %s=%.2f", pass+1, paramName, bestParams[paramName]), percentComplete); err != nil {
					return 0, nil, err
				}
				score, err := evaluate(ctx, algorithm, metric, imagePairs, trialParams)
				if err != nil {
					return 0, nil, err
				}
				if score > bestScore {
					bestScore = score
					bestParams[paramName] = candidate
					improved = true
					log.Printf("New best score: %.4f with param %s=%.4f", bestScore, paramName, candidate)
				} else {
					log.Printf("Trial score: %.4f with param %s=%.4f", score, paramName, candidate)
				}
			}
			log.Printf("Pass %d, param %s: best score=%.4f, best value=%.4f", pass+1, paramName, bestScore, bestParams[paramName])
		}

		log.Printf("Pass %d: score=%.4f, params=%v", pass+1, bestScore, bestParams)
		if !improved {
			break
		}
	}

	return bestScore, bestParams, nil
}

func evaluate(ctx context.Context, algorithm string, metric string, imagePairs []superpixel_optimizer.SpOptimizerInputPair, params map[string]float64) (float64, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	metricParams, err := superpixel_metrics.GetDefaultSpMetricParameters(metric)
	if err != nil {
		return 0, err
	}

	scores := make(chan float64, len(imagePairs))
	var wg sync.WaitGroup

	for _, pair := range imagePairs {
		wg.Add(1)
		go func(pair superpixel_optimizer.SpOptimizerInputPair) {
			defer wg.Done()
			if ctx.Err() != nil {
				return
			}
			result, err := superpixel.Run(algorithm, pair.Image, params)
			if err != nil {
				return
			}
			if ctx.Err() != nil {
				return
			}
			score, err := superpixel_metrics.Run(metric, result, pair.Image, pair.GroundTruth, metricParams)
			if err != nil {
				return
			}
			select {
			case scores <- score:
			case <-ctx.Done():
			}
		}(pair)
	}

	wg.Wait()
	close(scores)
	if err := ctx.Err(); err != nil {
		return 0, err
	}

	total := 0.0
	count := 0
	for score := range scores {
		total += score
		count++
	}
	if count == 0 {
		return 0, nil
	}
	return total / float64(count), nil
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
		for _, factor := range []float64{0.5, 0.75, 0.875, 0.95, 0.99, 1.0, 1.01, 1.05, 1.125, 1.25, 1.5, 2.0} {
			add(currentValue * factor)
		}
	}

	return values
}

func copyParams(params map[string]float64) map[string]float64 {
	result := make(map[string]float64, len(params))
	for k, v := range params {
		result[k] = v
	}
	return result
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
