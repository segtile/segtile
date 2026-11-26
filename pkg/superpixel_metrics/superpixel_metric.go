package superpixel_metrics

import (
	"fmt"
	"image"

	"github.com/segtile/segtile/pkg/superpixel"
	"github.com/segtile/segtile/pkg/superpixel_map"
)

type SpMetricParameterKind int

const (
	SpMetricParameterKindInt SpMetricParameterKind = iota
	SpMetricParameterKindFloat
	SpMetricParameterKindBool
)

type SpMetricParameterInfo struct {
	Id          string
	Kind        SpMetricParameterKind
	Title       string
	Description string
	Default     float64
	Min         float64
	Max         float64
}

type spMetricRegistryEntry struct {
	Title          string
	Description    string
	OptimalScore   float64
	Run            func(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error)
	Parameters     []SpMetricParameterInfo
	ParameterIndex map[string]int // Maps parameter id to its index in the Parameters slice
}

var spMetricRegistry = make(map[string]spMetricRegistryEntry)

func RegisterSpMetric(
	id string,
	title string,
	description string,
	optimalScore float64,
	run func(result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error),
	parameters []SpMetricParameterInfo) {
	if _, exists := spMetricRegistry[id]; exists {
		panic("Superpixel metric with id " + id + " is already registered")
	}

	paramIds := make(map[string]bool)
	for _, param := range parameters {
		if _, exists := paramIds[param.Id]; exists {
			panic("Attempted to register duplicate parameter id " + param.Id + " for superpixel metric " + id)
		}
		paramIds[param.Id] = true
	}

	parameterIndex := make(map[string]int)
	for i, param := range parameters {
		parameterIndex[param.Id] = i
	}

	spMetricRegistry[id] = spMetricRegistryEntry{
		Title:          title,
		Description:    description,
		OptimalScore:   optimalScore,
		Run:            run,
		Parameters:     parameters,
		ParameterIndex: parameterIndex,
	}
}

func ListSpMetrics() []string {
	metrics := make([]string, 0, len(spMetricRegistry))
	for id := range spMetricRegistry {
		metrics = append(metrics, id)
	}
	return metrics
}

func GetSpMetricInfo(id string) (title string, description string, parameters []SpMetricParameterInfo, err error) {
	entry, exists := spMetricRegistry[id]
	if !exists {
		return "", "", nil, fmt.Errorf("Superpixel metric with id %s is not registered", id)
	}
	return entry.Title, entry.Description, entry.Parameters, nil
}

func GetDefaultSpMetricParameters(id string) (params map[string]float64, err error) {
	entry, exists := spMetricRegistry[id]
	if !exists {
		return nil, fmt.Errorf("Superpixel metric with id %s is not registered", id)
	}
	params = make(map[string]float64)
	for _, paramInfo := range entry.Parameters {
		params[paramInfo.Id] = paramInfo.Default
	}
	return params, nil
}

// Run executes the registered superpixel metric with the given id.
// It takes the superpixel result, the original image, the ground truth mask, and a map of parameters as input.
// Not all algorithms may use all parameters.
func Run(id string, result superpixel.SpResult, img image.Image, groundTruth *superpixel_map.Mask, params map[string]float64) (score float64, err error) {
	entry, exists := spMetricRegistry[id]
	if !exists {
		return 0, fmt.Errorf("Superpixel metric with id %s is not registered", id)
	}
	return entry.Run(result, img, groundTruth, params)
}
