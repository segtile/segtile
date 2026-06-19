package app_config

import (
	"log"

	"github.com/d4l3k/go-bayesopt"
)

func (appState *AppState) OptimizeSpParameters() {

	// Copy the existing SP implementation
	spImpl := appState.SuperpixelImplementation

	// Prepare the parameters for the optimizer.
	bayesParameters := []bayesopt.Param{}

	for _, param := range spImpl.GetSupportedParameters() {
		bayesParameters = append(bayesParameters, bayesopt.UniformParam{
			Name: string(param),
			Min:  param.Min(),
			Max:  param.Max(),
		})
	}

	// Create the optimizer with the parameters.
	optimizer := bayesopt.New(
		bayesParameters,
	)

	parameterMap, score, err := optimizer.Optimize(func(params map[bayesopt.Param]float64) float64 {
		return evaluate(appState, params)
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(parameterMap, score)
}

func evaluate(appState *AppState, params map[bayesopt.Param]float64) float64 {
	// TODO
	return 0
}
