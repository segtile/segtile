package superpixel_metrics

import "github.com/struffel/annotation-tool/pkg/superpixel"

type SpMetric interface {
	// Load a superpixel result to be evaluated.
	SetSpResult(spResult *superpixel.SpResult)

	//
	GetSpResult() *superpixel.SpResult

	// Returns the score for the current superpixel result
	GetScore() float64

	// Returns the score that is considered "perfect" for this metric
	PerfectScore() float64
}
