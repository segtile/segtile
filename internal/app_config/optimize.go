package app_config

import (
	"log"
	"math/rand/v2"
	"os"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer/labeling_calibration"
)

const optimizeSampleCount = 16

// collectGTSamples assembles up to n image+GT-mask pairs to use during optimization.
// The current image with its active selection is always the first entry (if a selection exists).
// Additional samples are drawn randomly from images that already have a saved mask on disk.
func collectGTSamples(appState *AppState, n int) []superpixel_optimizer.SpOptimizerImagePair {
	var samples []superpixel_optimizer.SpOptimizerImagePair

	// Shuffle all image indices and pick up to n–1 additional saved masks.
	indices := make([]int, len(appState.ImageFiles))
	for i := range indices {
		indices[i] = i
	}
	rand.Shuffle(len(indices), func(i, j int) {
		indices[i], indices[j] = indices[j], indices[i]
	})

	for _, i := range indices {
		if len(samples) >= n {
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

	optimizer := labeling_calibration.SpOptimizerLabelingCalibration{}
	optimizer.SpOptimizerData.SetImplementation(appState.SuperpixelImplementation)
	optimizer.SpOptimizerData.SetImageAndGroundTruthPaths(samples)

	optimizer.Run()
	/*
		// Prepare the parameters for the optimizer.
		bayesParameters := []bayesopt.Param{}

		/*for _, param := range spImpl.GetSupportedParameters() {
			bayesParameters = append(bayesParameters, bayesopt.UniformParam{
				Name: string(param),
				Min:  param.Min(),
				Max:  param.Max(),
			})
		} //
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
			result := evaluate(appState, samples, params)
			log.Printf("Round %d: score=%.4f, params=%v", optimizer.Rounds(), result, params)
			return result
		})
		if err != nil {
			log.Fatal(err)
		}
		log.Printf("Optimization complete: best score=%.4f, params=%v", score, parameterMap)*/
}
