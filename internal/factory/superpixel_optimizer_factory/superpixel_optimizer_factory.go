package superpixel_optimizer_factory

import (
	"fmt"

	"github.com/struffel/annotation-tool/internal/superpixel_optimizer"
	"github.com/struffel/annotation-tool/internal/superpixel_optimizer/coordinate_descent"
)

const (
	// Algorithm names
	nameCoordinateDescent = "coordinate_descent"
)

func NewSuperpixelOptimizerByName(name string) (superpixel_optimizer.SpOptimizer, error) {
	switch name {
	case nameCoordinateDescent:
		return &coordinate_descent.SpOptimizerCoordinateDescent{}, nil
	default:
		return nil, fmt.Errorf("Unknown algorithm name: %s", name)
	}
}

func GetOptimizerName(impl superpixel_optimizer.SpOptimizer) (string, error) {
	switch impl.(type) {
	case *coordinate_descent.SpOptimizerCoordinateDescent:
		return nameCoordinateDescent, nil
	default:
		return "unknown", fmt.Errorf("Unknown implementation: %T", impl)
	}
}
