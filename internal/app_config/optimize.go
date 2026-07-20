package app_config

import (
	"context"
	"errors"
	"log"
	"math/rand/v2"
	"os"

	"github.com/struffel/annotation-tool/internal/io_utils"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/labeling_calibration_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_optimizer"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_optimizer/coordinate_descent"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

const optimizeSampleCount = 16

// OptimizationResult carries the outcome of a parameter optimization.
type OptimizationResult struct {
	Score      float64
	Algorithm  string
	Parameters map[string]float64
	Err        error
}

// collectGTSamples assembles up to maxImageCount image+GT-mask pairs to use during optimization.
// The current image with its active selection is always the first entry (if a selection exists).
// Additional samples are drawn randomly from images that already have a saved mask on disk.
func collectGTSamples(appState *AppState, maxImageCount int) []superpixel_optimizer.SpOptimizerInputPair {
	var samples []superpixel_optimizer.SpOptimizerInputPair

	indices := make([]int, len(appState.ImageFiles))
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	for _, i := range indices {
		if len(samples) >= maxImageCount {
			break
		}
		maskPath := appState.maskPathForIndex(i)
		if _, err := os.Stat(maskPath); err != nil {
			continue
		}
		loadedMask := io_utils.ToGray16(io_utils.LoadImage(maskPath))
		if !superpixel_utils.HasNonZeroPixels(loadedMask) {
			continue
		}
		img := io_utils.LoadImage(appState.ImageFiles[i])
		samples = append(samples, superpixel_optimizer.SpOptimizerInputPair{Image: img, GroundTruth: loadedMask})
	}

	return samples
}

func (appState *AppState) OptimizeSpParameters(ctx context.Context, progress superpixel_optimizer.SpOptimizerProgressIndicator) OptimizationResult {

	// TODO: Configured here for now, but could be made user-configurable in the future.
	optimizerName := "coordinate_descent"

	// Collect GT samples once; all optimizer rounds will use the same set.
	samples := collectGTSamples(appState, optimizeSampleCount)
	if len(samples) == 0 {
		return OptimizationResult{Err: errors.New("no ground-truth samples available for optimization")}
	}
	log.Printf("Optimizing using %d GT sample(s)", len(samples))

	optimizerParams, err := superpixel_optimizer.GetDefaultSpOptimizerParameters(optimizerName)
	if err != nil {
		return OptimizationResult{Err: err}
	}

	score, bestAlgorithm, bestParams, err := superpixel_optimizer.Run(
		ctx,
		optimizerName,
		samples,
		"labeling_calibration_score",
		[]string{appState.Settings.Superpixel.SpAlgorithmId},
		optimizerParams,
		progress,
	)
	if err != nil {
		return OptimizationResult{Err: err}
	}
	return OptimizationResult{
		Score:      score,
		Algorithm:  bestAlgorithm,
		Parameters: bestParams,
	}
}

// ApplyOptimizationResult applies a completed optimization on the Gio event loop.
func (appState *AppState) ApplyOptimizationResult(result OptimizationResult) {
	appState.CurrentlyOptimizing = false
	if errors.Is(result.Err, context.Canceled) {
		log.Println("Optimization canceled.")
		return
	}
	if result.Err != nil {
		log.Printf("Optimization failed: %v", result.Err)
		return
	}
	log.Printf("Optimization completed: score=%.4f, algorithm=%s, params=%v", result.Score, result.Algorithm, result.Parameters)

	appState.Settings.Superpixel.SpAlgorithmId = result.Algorithm
	appState.Settings.Superpixel.SpAlgorithmParameters = result.Parameters
	appState.RebuildEditors()

	for paramName, newValue := range result.Parameters {
		log.Printf("Set parameter %s to %v", paramName, newValue)
	}
	if err := appState.RecalculateSuperpixels(true, true); err != nil {
		log.Printf("Failed to recalculate superpixels: %v", err)
	}
	appState.Window.Invalidate()
}
