package slic

import "fmt"

type SlicParameter string

const (
	Count                                 SlicParameter = "count"
	Compactness                           SlicParameter = "compactness"
	Iterations                            SlicParameter = "iterations"
	InitialPerturbationNeighborhoodRadius SlicParameter = "initial_perturbation_neighborhood_radius"
)

func (this SlicParameter) String() string {
	return string(this)
}

// Titles
func (this SlicParameter) Title() string {
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
		return fmt.Sprintf("Unnamed Superpixel Parameter (%s)", this)
	}
}

// Descriptions
func (this SlicParameter) Description() string {
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
		return fmt.Sprintf("Parameter without description. (%s)", this)
	}
}
