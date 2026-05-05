package superpixel

import "strings"

type SpParameter int

const (
	Count SpParameter = iota
	Compactness
	Iterations
	InitialPerturbationNeighborhoodRadius
)

// Parse from string
var spParameterNameToValue = map[string]SpParameter{
	"count":       Count,
	"compactness": Compactness,
	"iterations":  Iterations,
	"initial_perturbation_neighborhood_radius": InitialPerturbationNeighborhoodRadius,
}

func ParseSpParameter(name string) (SpParameter, bool) {
	param, exists := spParameterNameToValue[strings.ToLower(name)]
	return param, exists
}

// Titles

var spConfigurationName = map[SpParameter]string{
	Count:                                 "Superpixel Count",
	Compactness:                           "Compactness",
	Iterations:                            "Iterations",
	InitialPerturbationNeighborhoodRadius: "Initial Perturbation Neighborhood Radius",
}

func (spc SpParameter) String() string {
	return spConfigurationName[spc]
}

// Descriptions

var spConfigurationDescription = map[SpParameter]string{
	Count:                                 "The desired number of superpixels.",
	Compactness:                           "The compactness factor balances color similarity and spatial proximity. Higher values lead to more compact superpixels.",
	Iterations:                            "The number of iterations the algorithm will run. More iterations can lead to better results but will take more time.",
	InitialPerturbationNeighborhoodRadius: "The neighborhood size (given as the radius) for initial perturbation of cluster centers.",
}

func (spc SpParameter) Description() string {
	return spConfigurationDescription[spc]
}
