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
	Threshold
	Beta
	TargetRegionCount
	HistogramBins
	SinkhornEpsilon
	SinkhornIterations
)

// Parse from string
var spParameterNameToValue = map[string]SpParameter{
	"count":       Count,
	"compactness": Compactness,
	"iterations":  Iterations,
	"initial_perturbation_neighborhood_radius": InitialPerturbationNeighborhoodRadius,
	"threshold":           Threshold,
	"beta":                Beta,
	"target_region_count": TargetRegionCount,
	"histogram_bins":      HistogramBins,
	"sinkhorn_epsilon":    SinkhornEpsilon,
	"sinkhorn_iterations": SinkhornIterations,
}

func ParseSpParameter(name string) (SpParameter, error) {
	param, exists := spParameterNameToValue[strings.ToLower(name)]
	if !exists {
		return 0, fmt.Errorf("Invalid superpixel parameter: %s", name)
	}
	return param, nil
}

// Titles

var spConfigurationName = map[SpParameter]string{
	Count:                                 "Superpixel Count",
	Compactness:                           "Compactness",
	Iterations:                            "Iterations",
	InitialPerturbationNeighborhoodRadius: "Initial Perturbation Neighborhood Radius",
	Threshold:                             "Threshold",
	Beta:                                  "Beta",
	TargetRegionCount:                     "Target Region Count",
	HistogramBins:                         "Histogram Bins",
	SinkhornEpsilon:                       "Sinkhorn Epsilon",
	SinkhornIterations:                    "Sinkhorn Iterations",
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
	Threshold:                             "Path distance threshold T controlling superpixel size: larger values produce fewer, larger superpixels.",
	Beta:                                  "Color-distance decay factor β in [0,1]: higher values improve boundary adherence at the cost of more superpixels.",
	TargetRegionCount:                     "Target number of regions after superpixel merging.",
	HistogramBins:                         "Number of representative color bins used for region histograms.",
	SinkhornEpsilon:                       "Entropic regularization parameter for Sinkhorn OT distance.",
	SinkhornIterations:                    "Number of iterations for the Sinkhorn solver.",
}

func (spc SpParameter) Description() string {
	return spConfigurationDescription[spc]
}
