package f1_match

import (
	"fmt"

	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
)

type SpOptimizerF1Match struct {
	superpixel_optimizer.SpOptimizerData
	desiredF1Score float64
}

func (o *SpOptimizerF1Match) GetSupportedParameters() []superpixel_optimizer.SpOptimizerParameter {
	return []superpixel_optimizer.SpOptimizerParameter{
		DesiredF1Score,
	}
}

func (o *SpOptimizerF1Match) SetParameter(parameter superpixel_optimizer.SpOptimizerParameter, value float64) error {
	switch parameter {
	case DesiredF1Score:
		if value < 0.0 || value > 1.0 {
			return fmt.Errorf("Desired F1 score must be between 0 and 1")
		}
		o.desiredF1Score = value
		return nil
	default:
		return fmt.Errorf("Unknown parameter for F1Match optimizer: %s", parameter)
	}
}

func (o *SpOptimizerF1Match) Run() error {

	// Start with the recommended settings

	// Make the mask approximation

	/*for imagePath, groundTruthPath := range o.AnnotationPairs {

	}*/
	return nil
}
