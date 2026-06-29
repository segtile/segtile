package superpixel_optimizer

import "github.com/struffel/annotation-tool/pkg/superpixel"

type SpOptimizerImagePair struct {
	ImagePath       string
	GroundTruthPath string
}

type SpOptimizer interface {

	// Set a map of image paths and corresponding ground truth selection paths to be used for optimization.
	SetImageAndGroundTruthPaths([]SpOptimizerImagePair)

	// Set a superpixel implementation to be evaluated and optimized.
	SetImplementation(implementation superpixel.SpImplementation)

	// Sets the parameters that will be optimized during the optimization process.
	SetOptimizationTargetParameters([]superpixel.SpParameter)

	// Gets the parameters that will be optimized during the optimization process.
	GetOptimizationTargetParameters() []superpixel.SpParameter

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
	AnnotationPairs              []SpOptimizerImagePair
	SpImplementation             superpixel.SpImplementation
	OptimizationTargetParameters []superpixel.SpParameter
}

func (o *SpOptimizerData) SetImageAndGroundTruthPaths(paths []SpOptimizerImagePair) {
	o.AnnotationPairs = paths
}

func (o *SpOptimizerData) SetImplementation(implementation superpixel.SpImplementation) {
	o.SpImplementation = implementation
}

func (o *SpOptimizerData) SetOptimizationTargetParameters(params []superpixel.SpParameter) {
	o.OptimizationTargetParameters = params
}

func (o *SpOptimizerData) GetOptimizationTargetParameters() []superpixel.SpParameter {
	return o.OptimizationTargetParameters
}
