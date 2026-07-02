package superpixel

import (
	"fmt"
	"image"
)

type SpParameterKind int

const (
	SpParameterKindInt SpParameterKind = iota
	SpParameterKindFloat
	SpParameterKindBool
)

type SpParameterInfo struct {
	Kind        SpParameterKind
	Title       string
	Description string
	Default     float64
	Min         float64
	Max         float64
}

type spRegistryEntry struct {
	Title       string
	Description string
	Run         func(img image.Image, params map[string]any) (result SpResult, err error)
	Parameters  map[string]SpParameterInfo
}

var spRegistry = make(map[string]spRegistryEntry)

func RegisterSpAlgorithm(id string, title string, description string, run func(img image.Image, params map[string]any) (result SpResult, err error), parameters map[string]SpParameterInfo) {
	if _, exists := spRegistry[id]; exists {
		panic("Superpixel algorithm with id " + id + " is already registered")
	}

	spRegistry[id] = spRegistryEntry{
		Title:       title,
		Description: description,
		Run:         run,
		Parameters:  parameters,
	}
}

func ListParameterInfo(algorithmId string) (map[string]SpParameterInfo, error) {
	entry, exists := spRegistry[algorithmId]
	if !exists {
		return nil, fmt.Errorf("Superpixel algorithm with id %s is not registered", algorithmId)
	}
	return entry.Parameters, nil
}

func GetDefaultParameters(id string) (params map[string]any, err error) {
	entry, exists := spRegistry[id]
	if !exists {
		return nil, fmt.Errorf("Superpixel algorithm with id %s is not registered", id)
	}
	params = make(map[string]any)
	for paramName, paramInfo := range entry.Parameters {
		params[paramName] = paramInfo.Default
	}
	return params, nil
}

func ListAlgorithms() []string {
	algorithms := make([]string, 0, len(spRegistry))
	for id := range spRegistry {
		algorithms = append(algorithms, id)
	}
	return algorithms
}

func GetAlgorithmInfo(id string) (title string, description string, parameters map[string]SpParameterInfo, err error) {
	entry, exists := spRegistry[id]
	if !exists {
		return "", "", nil, fmt.Errorf("Superpixel algorithm with id %s is not registered", id)
	}
	return entry.Title, entry.Description, entry.Parameters, nil
}

func Run(id string, img image.Image, params map[string]any) (result SpResult, err error) {
	entry, exists := spRegistry[id]
	if !exists {
		return SpResult{}, fmt.Errorf("Superpixel algorithm with id %s is not registered", id)
	}
	return entry.Run(img, params)
}
