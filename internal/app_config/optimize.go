package app_config

import (
	"image"
	"log"
	"math/rand/v2"
	"os"

	"github.com/d4l3k/go-bayesopt"
	"github.com/struffel/annotation-tool/internal/evaluation"
	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_factory"
)

const optimizeSampleCount = 16

type gtSample struct {
	image *image.NRGBA
	mask  *image.Gray16
}

// collectGTSamples assembles up to n image+GT-mask pairs to use during optimization.
// The current image with its active selection is always the first entry (if a selection exists).
// Additional samples are drawn randomly from images that already have a saved mask on disk.
func collectGTSamples(appState *AppState, n int) []gtSample {
	var samples []gtSample

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
		img := io_utils.ToNRGBA(io_utils.LoadImage(appState.ImageFiles[i]))
		samples = append(samples, gtSample{image: img, mask: loadedMask})
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

	// Prepare the parameters for the optimizer.
	bayesParameters := []bayesopt.Param{}

	/*for _, param := range spImpl.GetSupportedParameters() {
		bayesParameters = append(bayesParameters, bayesopt.UniformParam{
			Name: string(param),
			Min:  param.Min(),
			Max:  param.Max(),
		})
	}*/
	bayesParameters = append(bayesParameters, bayesopt.UniformParam{
		Name: string(superpixel.Count),
		Min:  superpixel.Count.Min(),
		Max:  superpixel.Count.Max(),
	})
	bayesParameters = append(bayesParameters, bayesopt.UniformParam{
		Name: string(superpixel.Compactness),
		Min:  superpixel.Compactness.Min(),
		Max:  superpixel.Compactness.Max(),
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
	log.Printf("Optimization complete: best score=%.4f, params=%v", score, parameterMap)
}

func evaluate(appState *AppState, samples []gtSample, params map[bayesopt.Param]float64) float64 {
	if len(samples) == 0 {
		return 0
	}

	implName, err := superpixel_factory.ImplementationName(appState.SuperpixelImplementation)
	if err != nil {
		return 0
	}

	totalScore := 0.0
	for _, sample := range samples {
		newImpl, err := superpixel_factory.NewSuperpixelImplementationByName(implName)
		if err != nil {
			continue
		}
		newImpl.SetImage(sample.image)
		newImpl.SetRecommendedParameters()

		// Apply the candidate parameters
		for param, value := range params {
			newImpl.SetParameter(superpixel.SpParameter(param.GetName()), value)
		}

		if err := newImpl.Run(); err != nil {
			continue
		}

		spResult := newImpl.GetResult()
		score, _, _, _, _, _, _ := evaluation.SuperpixelEvaluation(sample.mask, spResult, 10)
		totalScore += score
	}

	return totalScore / float64(len(samples))
}
