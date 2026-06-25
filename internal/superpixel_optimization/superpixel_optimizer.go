package superpixel_optimization

import (
	"fmt"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/tiff"

	"github.com/struffel/annotation-tool/internal/io_utils"
	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/average_boundary_distance"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/density_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/f1_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_metrics/regularity_score"
	"github.com/struffel/annotation-tool/pkg/superpixel_utils"
)

// ImageEvaluationResult holds the evaluation outcome for a single image/ground-truth pair
// under a specific set of superpixel parameters.
type ImageEvaluationResult struct {
	ImagePath                  string
	GroundTruthPath            string
	ActualSpCount              int
	SelectedSpCount            int
	SelectedBorderingSpCount   int
	AverageBoundaryDistance    float64
	F1Score                    float64
	DensityCalibrationScore    float64
	RegularityCalibrationScore float64
	LabelingCalibrationScore   float64
}

type SpOptimizer interface {

	// Set a map of image paths and corresponding ground truth selection paths to be used for optimization.
	SetImageAndGroundTruthPaths(paths map[string]string)

	// Set a superpixel implementation to be evaluated and optimized.
	SetImplementation(implementation superpixel.SpImplementation)

	// Run the optimization process
	Run() error

	// Get the best parameters and the resulting score found by the optimization process.
	GetResults() (score float64, parameters map[superpixel.SpParameter]float64, err error)

	// GetDetailedResults returns per-image evaluation results from the last Run call.
	GetDetailedResults() []ImageEvaluationResult
}

type SpOptimizerData struct {
	Paths             map[string]string
	Implementation    superpixel.SpImplementation
	OptimalParameters map[superpixel.SpParameter]float64
	ImageResults      []ImageEvaluationResult
}

func (o *SpOptimizerData) SetImageAndGroundTruthPaths(paths map[string]string) {
	o.Paths = paths
}

func (o *SpOptimizerData) SetImplementation(implementation superpixel.SpImplementation) {
	o.Implementation = implementation
}

func (o *SpOptimizerData) GetDetailedResults() []ImageEvaluationResult {
	return o.ImageResults
}

// EvaluatePaths runs the superpixel implementation (with its current parameters) against
// every image/ground-truth pair in o.Paths and returns one ImageEvaluationResult per pair.
// Pairs with mismatched dimensions or an all-zero ground-truth mask are skipped with a warning.
func (o *SpOptimizerData) EvaluatePaths() ([]ImageEvaluationResult, error) {
	results := make([]ImageEvaluationResult, 0, len(o.Paths))

	for imgPath, gtPath := range o.Paths {
		img := io_utils.ToNRGBA(io_utils.LoadImage(imgPath))
		groundTruth := io_utils.ToGray16(io_utils.LoadImage(gtPath))

		if img.Bounds() != groundTruth.Bounds() {
			fmt.Printf("Warning: image and mask dimensions do not match for %s and %s. Skipping.\n", imgPath, gtPath)
			continue
		}

		if !superpixel_metrics.HasNonZeroPixels(groundTruth) {
			fmt.Printf("Warning: ground truth mask %s contains no non-zero pixels. Skipping.\n", gtPath)
			continue
		}

		o.Implementation.SetImage(img)
		if err := o.Implementation.Run(); err != nil {
			return nil, fmt.Errorf("error running superpixel algorithm on %s: %w", imgPath, err)
		}

		spResult := o.Implementation.GetResult()

		densityEval := density_score.SpMetricDensityScore{}
		densityEval.SetSpResult(spResult)
		densityEval.SetGroundTruthSegmentation(groundTruth)
		densityScore := densityEval.GetScore()

		f1Eval := f1_score.SpMetricF1Score{}
		f1Eval.SetSpResult(spResult)
		f1Eval.SetGroundTruthSegmentation(groundTruth)
		f1Score := f1Eval.GetScore()

		regularityEval := regularity_score.SpMetricRegularityScore{}
		regularityEval.SetSpResult(spResult)
		regularityEval.SetGroundTruthSegmentation(groundTruth)
		regularityScore := regularityEval.GetScore()

		abdEval := average_boundary_distance.SpMetricAverageBoundaryDistance{}
		abdEval.SetSpResult(spResult)
		abdEval.SetGroundTruthSegmentation(groundTruth)
		abd := abdEval.GetScore()

		selection := spResult.SelectFromMask(groundTruth, 0.5)
		selectedSpCount, selectedBorderingSpCount := superpixel_utils.CountBorderingSuperpixels(spResult.LabelMap, selection)

		results = append(results, ImageEvaluationResult{
			ImagePath:                  imgPath,
			GroundTruthPath:            gtPath,
			ActualSpCount:              spResult.SpCount,
			SelectedSpCount:            selectedSpCount,
			SelectedBorderingSpCount:   selectedBorderingSpCount,
			AverageBoundaryDistance:    abd,
			F1Score:                    f1Score,
			DensityCalibrationScore:    densityScore,
			RegularityCalibrationScore: regularityScore,
			LabelingCalibrationScore:   densityScore * f1Score * regularityScore,
		})
	}

	return results, nil
}
