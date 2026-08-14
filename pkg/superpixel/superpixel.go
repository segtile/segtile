package superpixel

import (
	"fmt"
	"image"
	"sort"
)

type SpParameterKind int

const (
	SpParameterKindInt SpParameterKind = iota
	SpParameterKindFloat
	SpParameterKindBool
)

type SpParameterInfo struct {
	Id          string
	Kind        SpParameterKind
	Title       string
	Description string
	Default     float64
	Min         float64
	Max         float64
}

type spRegistryEntry struct {
	Title          string
	Description    string
	Run            func(img image.Image, params map[string]float64) (result SpResult, err error)
	Parameters     []SpParameterInfo
	ParameterIndex map[string]int // Maps parameter id to its index in the Parameters slice
}

var spRegistry = make(map[string]spRegistryEntry)

func RegisterSpAlgorithm(id string, title string, description string, run func(img image.Image, params map[string]float64) (result SpResult, err error), parameters []SpParameterInfo) {

	// Do not allow multiple registrations on the same id
	if _, exists := spRegistry[id]; exists {
		panic("Attempted to register algorithm with duplicate id " + id)
	}

	// Ensure that no two parameters have the same id
	paramIds := make(map[string]bool)
	for _, param := range parameters {
		if _, exists := paramIds[param.Id]; exists {
			panic("Attempted to register duplicate parameter id " + param.Id + " for superpixel algorithm " + id)
		}
		paramIds[param.Id] = true
	}

	// Build the parameter index for quick lookup by id
	parameterIndex := make(map[string]int)
	for i, param := range parameters {
		parameterIndex[param.Id] = i
	}

	spRegistry[id] = spRegistryEntry{
		Title:          title,
		Description:    description,
		Run:            run,
		Parameters:     parameters,
		ParameterIndex: parameterIndex,
	}
}

func ListParameterInfo(algorithmId string) ([]SpParameterInfo, error) {
	entry, exists := spRegistry[algorithmId]
	if !exists {
		return nil, fmt.Errorf("Superpixel algorithm with id %s is not registered", algorithmId)
	}
	return entry.Parameters, nil
}

func GetParameterInfo(algorithmId string, paramName string) (SpParameterInfo, error) {

	// Ensure algorithm exists
	entry, exists := spRegistry[algorithmId]
	if !exists {
		return SpParameterInfo{}, fmt.Errorf("Superpixel algorithm with id %s is not registered", algorithmId)
	}

	// Ensure parameter exists
	index, exists := entry.ParameterIndex[paramName]
	if !exists {
		return SpParameterInfo{}, fmt.Errorf("Parameter %s is not registered for superpixel algorithm with id %s", paramName, algorithmId)
	}

	return entry.Parameters[index], nil
}

func GetDefaultParameterValues(id string) (params map[string]float64, err error) {
	entry, exists := spRegistry[id]
	if !exists {
		return nil, fmt.Errorf("Superpixel algorithm with id %s is not registered", id)
	}
	params = make(map[string]float64)
	for _, param := range entry.Parameters {
		params[param.Id] = param.Default
	}
	return params, nil
}

func ListAlgorithms() []string {
	algorithms := make([]string, 0, len(spRegistry))
	for id := range spRegistry {
		algorithms = append(algorithms, id)
	}
	sort.Strings(algorithms)
	return algorithms
}

func GetAlgorithmInfo(id string) (title string, description string, parameters []SpParameterInfo, err error) {
	entry, exists := spRegistry[id]
	if !exists {
		return "", "", nil, fmt.Errorf("Superpixel algorithm with id %s is not registered", id)
	}
	return entry.Title, entry.Description, entry.Parameters, nil
}

func Run(id string, img image.Image, params map[string]float64) (result SpResult, err error) {
	entry, exists := spRegistry[id]
	if !exists {
		return SpResult{}, fmt.Errorf("Superpixel algorithm with id %s is not registered", id)
	}
	result, err = entry.Run(img, params)
	result.EnforceContiguousIndexing()
	return result, err
}
