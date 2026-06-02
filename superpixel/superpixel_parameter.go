package superpixel

import (
	"fmt"
	"strings"
)

type SpParameter int

const (
	Count SpParameter = iota
	Compactness
	Iterations
	InitialPerturbationNeighborhoodRadius
)

// Titles
func (this SpParameter) Title() string {
	switch this {
	case Count:
		return "Superpixel Count"
	case Compactness:
		return "Compactness"
	case Iterations:
		return "Iterations"
	case InitialPerturbationNeighborhoodRadius:
		return "Initial Perturbation Neighborhood Radius"
	default:
		return fmt.Sprintf("Unnamed Superpixel Parameter (%d)", this)
	}
}

// Descriptions
func (this SpParameter) Description() string {
	switch this {
	case Count:
		return "The desired number of superpixels."
	case Compactness:
		return "The compactness factor balances color similarity and spatial proximity. Higher values lead to more compact superpixels."
	case Iterations:
		return "The number of iterations the algorithm will run. More iterations can lead to better results but will take more time."
	case InitialPerturbationNeighborhoodRadius:
		return "The neighborhood size (given as the radius) for initial perturbation of cluster centers."
	default:
		return fmt.Sprintf("No description for this parameter. (%d)", this)
	}
}

// Keys
var spParameterKey = map[SpParameter]string{
	Count:                                 "count",
	Compactness:                           "compactness",
	Iterations:                            "iterations",
	InitialPerturbationNeighborhoodRadius: "initial_perturbation_neighborhood_radius",
}

// Key returns the lowercase serialization key for use in config files.
func (this SpParameter) Key() string {
	if key, exists := spParameterKey[this]; exists {
		return key
	}
	return fmt.Sprintf("unknown_key_%d", this)
}

func ParseSpParameter(name string) (SpParameter, error) {
	for param, key := range spParameterKey {
		if key == strings.ToLower(name) {
			return param, nil
		}
	}
	return 0, fmt.Errorf("Invalid superpixel parameter: %s", name)
}
