package superpixel_metrics

import (
	"fmt"
	"image"

	"github.com/struffel/annotation-tool/pkg/superpixel"
)

type SpMetricParameterKind int

const (
	SpMetricParameterKindInt SpMetricParameterKind = iota
	SpMetricParameterKindFloat
	SpMetricParameterKindBool
)

type SpMetricParameterInfo struct {
	Kind        SpMetricParameterKind
	Title       string
	Description string
	Default     float64
	Min         float64
	Max         float64
}

type spMetricRegistryEntry struct {
	Title        string
	Description  string
	OptimalScore float64
	Run          func(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]any) (score float64, err error)
	Parameters   map[string]SpMetricParameterInfo
}

var spMetricRegistry = make(map[string]spMetricRegistryEntry)

func RegisterSpMetric(
	id string,
	title string,
	description string,
	optimalScore float64,
	run func(result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]any) (score float64, err error),
	parameters map[string]SpMetricParameterInfo) {
	if _, exists := spMetricRegistry[id]; exists {
		panic("Superpixel metric with id " + id + " is already registered")
	}

	spMetricRegistry[id] = spMetricRegistryEntry{
		Title:        title,
		Description:  description,
		OptimalScore: optimalScore,
		Run:          run,
		Parameters:   parameters,
	}
}

func ListSpMetricAlgorithms() []string {
	algorithms := make([]string, 0, len(spMetricRegistry))
	for id := range spMetricRegistry {
		algorithms = append(algorithms, id)
	}
	return algorithms
}

func GetSpMetricInfo(id string) (title string, description string, parameters map[string]SpMetricParameterInfo, err error) {
	entry, exists := spMetricRegistry[id]
	if !exists {
		return "", "", nil, fmt.Errorf("Superpixel metric with id %s is not registered", id)
	}
	return entry.Title, entry.Description, entry.Parameters, nil
}

// Run executes the registered superpixel metric with the given id.
// It takes the superpixel result, the original image, the ground truth mask, and a map of parameters as input.
// Not all algorithms may use all parameters.
func Run(id string, result superpixel.SpResult, img image.Image, groundTruth *image.Gray16, params map[string]any) (score float64, err error) {
	entry, exists := spMetricRegistry[id]
	if !exists {
		return 0, fmt.Errorf("Superpixel metric with id %s is not registered", id)
	}
	return entry.Run(result, img, groundTruth, params)
}
