package app_config

import (
	"log"
	"math/rand/v2"
	"os"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
	_ "github.com/struffel/annotation-tool/internal/superpixel_optimizer/coordinate_descent"
	_ "github.com/struffel/annotation-tool/pkg/superpixel_metrics/labeling_calibration_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
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
		loadedMask := io_utils.ToGray16(io_utils.LoadImage(maskPath))
		if !superpixel_utils.HasNonZeroPixels(loadedMask) {
			continue
		}
		img := io_utils.LoadImage(appState.ImageFiles[i])
		samples = append(samples, superpixel_optimizer.SpOptimizerInputPair{Image: img, GroundTruth: loadedMask})
	}

	return samples
}

func (appState *AppState) OptimizeSpParameters() {

	// Collect GT samples once; all optimizer rounds will use the same set.
	samples := collectGTSamples(appState, optimizeSampleCount)
	if len(samples) == 0 {
		log.Println("Optimization aborted: no GT samples available (make a selection first).")
		return
	}
	log.Printf("Optimizing using %d GT sample(s)", len(samples))

	optimizerParams, err := superpixel_optimizer.GetDefaultSpOptimizerParameters("coordinate_descent")
	if err != nil {
		log.Printf("Optimization failed: %v", err)
		return
	}

	score, bestAlgorithm, bestParams, err := superpixel_optimizer.Run(
		"coordinate_descent",
		samples,
		"labeling_calibration_score",
		[]string{appState.Settings.Superpixel.SpAlgorithmId},
		optimizerParams,
	)
	if err != nil {
		log.Printf("Optimization failed: %v", err)
		return
	}
	log.Printf("Optimization completed: score=%.4f, algorithm=%s, params=%v", score, bestAlgorithm, bestParams)

	appState.Settings.Superpixel.SpAlgorithmId = bestAlgorithm

	// TODO: Actually rebuild properly
	for paramName, newValue := range bestParams {
		if _, ok := appState.Settings.Superpixel.SpAlgorithmParameters[paramName]; ok {
			appState.Settings.Superpixel.SpAlgorithmParameters[paramName] = newValue
			log.Printf("Set parameter %s to %v", paramName, newValue)
		}
	}

	appState.SyncEditors()
	appState.RecalculateSuperpixels()
	appState.Window.Invalidate()
}
