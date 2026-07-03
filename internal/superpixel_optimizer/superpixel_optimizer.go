package superpixel_optimizer

import (
	"fmt"
	"image"
)

type SpOptimizerInputPair struct {
	Image       image.Image
	GroundTruth *image.Gray16
}

type SpOptimizerParameterKind int

const (
	SpOptimizerParameterKindInt SpOptimizerParameterKind = iota
	SpOptimizerParameterKindFloat
	SpOptimizerParameterKindBool
)

type SpOptimizerParameterInfo struct {
	Id          string
	Kind        SpOptimizerParameterKind
	Title       string
	Description string
	Default     float64
	Min         float64
	Max         float64
}

type spOptimizerRegistryEntry struct {
	Title          string
	Description    string
	Run            func(imagePairs []SpOptimizerInputPair, metric string, algorithms []string, optimizer_params map[string]float64) (best_score float64, best_algorithm string, best_score_algorithm_params map[string]float64, err error)
	Parameters     []SpOptimizerParameterInfo
	ParameterIndex map[string]int // Maps parameter id to its index in the Parameters slice
}

var spOptimizerRegistry = make(map[string]spOptimizerRegistryEntry)

func RegisterSpOptimizer(id string, title string, description string, run func(imagePairs []SpOptimizerInputPair, metric string, algorithms []string, optimizer_params map[string]float64) (best_score float64, best_algorithm string, best_score_algorithm_params map[string]float64, err error), parameters []SpOptimizerParameterInfo) {
	if _, exists := spOptimizerRegistry[id]; exists {
		panic("Superpixel optimizer with id " + id + " is already registered")
	}

	paramIds := make(map[string]bool)
	for _, param := range parameters {
		if _, exists := paramIds[param.Id]; exists {
			panic("Attempted to register duplicate parameter id " + param.Id + " for superpixel optimizer " + id)
		}
		paramIds[param.Id] = true
	}

	parameterIndex := make(map[string]int)
	for i, param := range parameters {
		parameterIndex[param.Id] = i
	}

	spOptimizerRegistry[id] = spOptimizerRegistryEntry{
		Title:          title,
		Description:    description,
		Run:            run,
		Parameters:     parameters,
		ParameterIndex: parameterIndex,
	}
}

func ListOptimizers() []string {
	optimizers := make([]string, 0, len(spOptimizerRegistry))
	for id := range spOptimizerRegistry {
		optimizers = append(optimizers, id)
	}
	return optimizers
}

func GetDefaultSpOptimizerParameters(id string) (params map[string]float64, err error) {
	entry, exists := spOptimizerRegistry[id]
	if !exists {
		return nil, fmt.Errorf("Superpixel optimizer with id %s is not registered", id)
	}
	params = make(map[string]float64)
	for _, paramInfo := range entry.Parameters {
		params[paramInfo.Id] = paramInfo.Default
	}
	return params, nil
}

func GetOptimizerInfo(id string) (title string, description string, parameters []SpOptimizerParameterInfo, err error) {
	entry, exists := spOptimizerRegistry[id]
	if !exists {
		return "", "", nil, fmt.Errorf("Superpixel optimizer with id %s is not registered", id)
	}
	return entry.Title, entry.Description, entry.Parameters, nil
}

func Run(id string, imagePairs []SpOptimizerInputPair, metric string, algorithms []string, optimizer_params map[string]float64) (best_score float64, best_algorithm string, best_score_algorithm_params map[string]float64, err error) {
	entry, exists := spOptimizerRegistry[id]
	if !exists {
		return 0, "", nil, fmt.Errorf("Superpixel optimizer with id %s is not registered", id)
	}
	return entry.Run(imagePairs, metric, algorithms, optimizer_params)
}
