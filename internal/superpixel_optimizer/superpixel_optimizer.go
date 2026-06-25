package superpixel_optimizer

import "github.com/struffel/annotation-tool/pkg/superpixel"

type SpOptimizer interface {

	// Set a map of image paths and corresponding ground truth selection paths to be used for optimization.
	SetImageAndGroundTruthPaths(paths map[string]string)

	// Set a superpixel implementation to be evaluated and optimized.
	SetImplementation(implementation superpixel.SpImplementation)

	// Get available optimizer parameters
	GetSupportedParameters() []SpOptimizerParameter

	// Set a parameter value for the optimizer.
	SetParameter(parameter SpOptimizerParameter, value float64) error

	// Run the optimization process
	Run() error

	// Get the best superpixel parameters and the resulting score found by the optimization process.
	GetResults() (score float64, parameters map[superpixel.SpParameter]float64, err error)
}

type SpOptimizerData struct {
	AnnotationPairs  map[string]string
	SpImplementation superpixel.SpImplementation
}

func (o *SpOptimizerData) SetImageAndGroundTruthPaths(paths map[string]string) {
	o.AnnotationPairs = paths
}

func (o *SpOptimizerData) SetImplementation(implementation superpixel.SpImplementation) {
	o.SpImplementation = implementation
}
