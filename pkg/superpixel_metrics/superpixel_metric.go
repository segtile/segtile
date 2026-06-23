package superpixel_metrics

import (
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
)

type SpMetric interface {
	// Load a superpixel result to be evaluated.
	SetSpResult(spResult *superpixel.SpResult)

	// Get the superpixel result that is currently loaded.
	GetSpResult() *superpixel.SpResult

	// Set the ground truth mask to be used for evaluation.
	SetGroundTruthSegmentation(groundTruthMask *image.Gray16)

	// Get the ground truth mask that is currently loaded.
	GetGroundTruthSegmentation() *image.Gray16

	// Returns the score for the current superpixel result
	GetScore() float64

	// Returns the score that is considered "perfect" for this metric
	PerfectScore() float64
}

type SpMetricsData struct {
	spResult                *superpixel.SpResult
	groundTruthSegmentation *image.Gray16
}

func (this *SpMetricsData) SetSpResult(spResult *superpixel.SpResult) {
	this.spResult = spResult
}

func (this *SpMetricsData) GetSpResult() *superpixel.SpResult {
	return this.spResult
}

func (this *SpMetricsData) SetGroundTruthSegmentation(groundTruthMask *image.Gray16) {
	this.groundTruthSegmentation = groundTruthMask
}

func (this *SpMetricsData) GetGroundTruthSegmentation() *image.Gray16 {
	return this.groundTruthSegmentation
}
