package coordinate_descent

import (
	"fmt"
	"log"
	"math"
	"sort"
	"sync"

	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
)

func init() {
	parameterInfo := map[string]superpixel_optimizer.SpOptimizerParameterInfo{
		"passes": {
			Kind:        superpixel_optimizer.SpOptimizerParameterKindInt,
			Title:       "Passes",
			Description: "Number of passes to perform over the parameters. Each pass will attempt to optimize each parameter in turn.",
			Default:     3,
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

func runWithParams(imagePairs []superpixel_optimizer.SpOptimizerInputPair, metric string, algorithms []string, optimizer_params map[string]any) (best_score float64, best_score_algorithm_params map[string]any, err error) {
	return Run(imagePairs, metric, algorithms, optimizer_params["passes"].(int))
}

func Run(imagePairs []superpixel_optimizer.SpOptimizerInputPair, metric string, algorithms []string, passes int) (best_score float64, best_score_algorithm_params map[string]any, err error) {

	if len(imagePairs) == 0 {
		return 0, nil, fmt.Errorf("No image and ground truth paths provided for optimization.")
	}

	allAlgorithms := superpixel.ListAlgorithms()
	for _, algorithm := range algorithms {
		if !contains(allAlgorithms, algorithm) {
			return 0, nil, fmt.Errorf("Unknown superpixel algorithm %s", algorithm)
		}
	}

	allMetrics := superpixel_metrics.ListSpMetrics()
	if !contains(allMetrics, metric) {
		return 0, nil, fmt.Errorf("Unknown superpixel metric %s", metric)
	}

	best_score = -math.MaxFloat64

	for _, algorithm := range algorithms {
		score, params, err := optimizeAlgorithm(imagePairs, metric, algorithm, passes)
		if err != nil {
			return 0, nil, err
		}
		if score > best_score {
			best_score = score
			best_score_algorithm_params = params
			best_score_algorithm_params["algorithm"] = algorithm
		}
	}

	return best_score, best_score_algorithm_params, nil
}

func optimizeAlgorithm(
	imagePairs []superpixel_optimizer.SpOptimizerInputPair,
	metric string,
	algorithm string,
	passes int) (float64, map[string]any, error) {

	// Get default parameters for the algorithm
	defaultParams, err := superpixel.GetDefaultParameters(algorithm)
	if err != nil {
		return 0, nil, err
	}

	// Work internally with float64 for optimization arithmetic.
	initialFloatParams := paramsToFloat(defaultParams)

	// Sort param names for deterministic iteration order.
	paramNames := make([]string, 0, len(initialFloatParams))
	for k := range initialFloatParams {
		paramNames = append(paramNames, k)
	}
	sort.Strings(paramNames)

	bestParams := copyParams(initialFloatParams)
	bestScore := evaluate(algorithm, metric, imagePairs, bestParams)
	log.Printf("Initial score: %.4f, params: %v", bestScore, bestParams)

	for pass := 0; pass < passes; pass++ {
		improved := false

		for _, paramName := range paramNames {
			for _, candidate := range probeValues(bestParams[paramName], initialFloatParams[paramName]) {
				if candidate == bestParams[paramName] {
					continue
				}
				trialParams := copyParams(bestParams)
				trialParams[paramName] = candidate

				if score := evaluate(algorithm, metric, imagePairs, trialParams); score > bestScore {
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

	runnableParams, err := toRunnableParams(algorithm, bestParams)
	if err != nil {
		return 0, nil, err
	}
	return bestScore, runnableParams, nil
}

func evaluate(algorithm string, metric string, imagePairs []superpixel_optimizer.SpOptimizerInputPair, floatParams map[string]float64) float64 {
	runnableParams, err := toRunnableParams(algorithm, floatParams)
	if err != nil {
		return 0
	}

	metricParams, err := superpixel_metrics.GetDefaultSpMetricParameters(metric)
	if err != nil {
		return 0
	}

	scores := make(chan float64, len(imagePairs))
	var wg sync.WaitGroup

	for _, pair := range imagePairs {
		wg.Add(1)
		go func(pair superpixel_optimizer.SpOptimizerInputPair) {
			defer wg.Done()
			result, err := superpixel.Run(algorithm, pair.Image, runnableParams)
			if err != nil {
				return
			}
			score, err := superpixel_metrics.Run(metric, result, pair.Image, pair.GroundTruth, metricParams)
			if err != nil {
				return
			}
			scores <- score
		}(pair)
	}

	wg.Wait()
	close(scores)

	total := 0.0
	count := 0
	for score := range scores {
		total += score
		count++
	}
	if count == 0 {
		return 0
	}
	return total / float64(count)
}

// paramsToFloat converts a map[string]any (where values may be int or float64) to map[string]float64
// for use in optimization arithmetic.
func paramsToFloat(params map[string]any) map[string]float64 {
	result := make(map[string]float64, len(params))
	for k, v := range params {
		switch val := v.(type) {
		case float64:
			result[k] = val
		case int:
			result[k] = float64(val)
		case bool:
			if val {
				result[k] = 1.0
			} else {
				result[k] = 0.0
			}
		}
	}
	return result
}

// toRunnableParams converts float64 optimization params back to map[string]any,
// rounding int parameters to the nearest integer.
func toRunnableParams(algorithmId string, floatParams map[string]float64) (map[string]any, error) {
	_, _, paramInfos, err := superpixel.GetAlgorithmInfo(algorithmId)
	if err != nil {
		return nil, err
	}
	result := make(map[string]any, len(floatParams))
	for k, v := range floatParams {
		if info, ok := paramInfos[k]; ok && info.Kind == superpixel.SpParameterKindInt {
			result[k] = int(math.Round(v))
		} else {
			result[k] = v
		}
	}
	return result, nil
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
