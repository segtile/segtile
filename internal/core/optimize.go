package core

import (
	"context"
	"errors"
	"fmt"
	"log"
	"math/rand/v2"
	"os"

	"github.com/struffel/annotation-tool/pkg/progress"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_map"

	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/labeling_calibration_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_optimizer"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_optimizer/coordinate_descent"
)

const optimizeSampleCount = 16

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
		loadedMask := superpixel_map.NewMaskFromImage(LoadImage(maskPath))
		if loadedMask.IsEmpty() {
			continue
		}
		img := LoadImage(appState.ImageFiles[i])
		samples = append(samples, superpixel_optimizer.SpOptimizerInputPair{Image: img, GroundTruth: loadedMask})
	}

	return samples
}

func (appState *AppState) OptimizeSpParameters(ctx context.Context, progressIndicator progress.ProgressIndicator) (result superpixel_optimizer.SpOptimizerResult, err error) {

	// TODO: Configured here for now, but could be made user-configurable in the future.
	optimizerName := "coordinate_descent"

	// Collect GT samples once; all optimizer rounds will use the same set.
	samples := collectGTSamples(appState, optimizeSampleCount)
	if len(samples) == 0 {
		return superpixel_optimizer.SpOptimizerResult{}, fmt.Errorf("No ground truth samples available for optimization. Please save at least one mask before optimizing.")
	}
	log.Printf("Optimizing using %d GT sample(s)", len(samples))

	parametersToOptimize := appState.SelectedOptimizeParameters()
	if len(parametersToOptimize) == 0 {
		return superpixel_optimizer.SpOptimizerResult{}, fmt.Errorf("No parameters selected for optimization. Please check at least one parameter.")
	}

	optimizerParams, err := superpixel_optimizer.GetDefaultSpOptimizerParameters(optimizerName)
	if err != nil {
		return superpixel_optimizer.SpOptimizerResult{}, err
	}

	optResult, err := superpixel_optimizer.Run(
		ctx,
		optimizerName,
		samples,
		"labeling_calibration_score",
		appState.ActiveSettings.Superpixel.SpAlgorithmId,
		parametersToOptimize,
		appState.ActiveSettings.Superpixel.SpAlgorithmParameters,
		optimizerParams,
		progressIndicator,
	)
	if err != nil {
		return superpixel_optimizer.SpOptimizerResult{}, err
	}
	return optResult, nil
}

// ApplyOptimizationResult applies a completed optimization on the Gio event loop.
func (appState *AppState) ApplyOptimizationResult(result superpixel_optimizer.SpOptimizerResult) {

	// Check for errors
	if errors.Is(result.Err, context.Canceled) {
		log.Println("Optimization canceled.")
	} else if result.Err != nil {
		log.Printf("Optimization failed: %v", result.Err)
	} else {

		// Validate that the chosen SP algorithm exists and the parameters are valid.
		_, _, paramInfos, err := superpixel.GetAlgorithmInfo(result.Algorithm)
		if err != nil {
			log.Printf("Optimizer returned an invalid superpixel algorithm")
			return
		}
		for paramName := range result.Parameters {
			found := false
			for _, paramInfo := range paramInfos {
				if paramInfo.Id == paramName {
					found = true
					break
				}
			}
			if !found {
				log.Printf("Optimizer returned an invalid parameter %s for algorithm %s", paramName, result.Algorithm)
				return
			}
		}

		// Apply the algorithm and parameters to the app state, rebuild editors, and recalculate superpixels.
		log.Printf("Optimization completed: score=%.4f, algorithm=%s, params=%v", result.Score, result.Algorithm, result.Parameters)

		appState.ActiveSettings.Superpixel.SpAlgorithmId = result.Algorithm
		appState.ActiveSettings.Superpixel.SpAlgorithmParameters = result.Parameters
		appState.RebuildEditors()

		for paramName, newValue := range result.Parameters {
			log.Printf("Set parameter %s to %v", paramName, newValue)
		}
		if err := appState.RecalculateSuperpixels(true); err != nil {
			log.Printf("Failed to recalculate superpixels: %v", err)
		}
	}

	appState.Window.Invalidate()
}
