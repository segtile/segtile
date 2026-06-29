package app_config

import (
	"log"
	"math/rand/v2"
	"os"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer/labeling_calibration_coordinate_descent"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel/slic"
)

const optimizeSampleCount = 16

// collectGTSamples assembles up to maxImageCount image+GT-mask pairs to use during optimization.
// The current image with its active selection is always the first entry (if a selection exists).
// Additional samples are drawn randomly from images that already have a saved mask on disk.
func collectGTSamples(appState *AppState, maxImageCount int) []superpixel_optimizer.SpOptimizerImagePair {
	var samples []superpixel_optimizer.SpOptimizerImagePair

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
		if !isMaskNonEmpty(loadedMask) {
			continue
		}
		samples = append(samples, superpixel_optimizer.SpOptimizerImagePair{ImagePath: appState.ImageFiles[i], GroundTruthPath: maskPath})
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

	optimizer := labeling_calibration_coordinate_descent.SpOptimizerLabelingCalibrationCoordinateDescent{}
	optimizer.SpOptimizerData.SetImplementation(appState.SuperpixelImplementation)
	optimizer.SpOptimizerData.SetOptimizationTargetParameters([]superpixel.SpParameter{slic.Count, slic.Compactness})
	optimizer.SpOptimizerData.SetImageAndGroundTruthPaths(samples)

	optimizer.Run()

}
