package cli

import (
	"fmt"

	"github.com/segtile/segtile/pkg/superpixel"
)

func LoadDefaultSuperpixelParameters(algorithmName string) (map[string]float64, error) {
	if _, _, _, err := superpixel.GetAlgorithmInfo(algorithmName); err != nil {
		return nil, err
	}

	params, err := superpixel.GetDefaultParameterValues(algorithmName)
	if err != nil {
		return nil, err
	}

	return params, nil
}

func ApplySuperpixelParameterOverrides(algorithmName string, parameters map[string]float64, overrides map[string]float64) error {
	for name, value := range overrides {
		if _, err := superpixel.GetParameterInfo(algorithmName, name); err != nil {
			return fmt.Errorf("invalid parameter %s: %w", name, err)
		}
		parameters[name] = value
	}

	return nil
}

func LoadSuperpixelParametersFromCLI(algorithmName string, args []string) (map[string]float64, error) {
	parameters, err := LoadDefaultSuperpixelParameters(algorithmName)
	if err != nil {
		return nil, err
	}

	overrides, err := ParseParameterList(args)
	if err != nil {
		return nil, err
	}

	if err := ApplySuperpixelParameterOverrides(algorithmName, parameters, overrides); err != nil {
		return nil, err
	}

	return parameters, nil
}
