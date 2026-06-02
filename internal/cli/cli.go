package cli

import (
	"fmt"
	"strconv"
	"strings"
)

type ParameterRange struct {
	Name  string
	Start float64
	End   float64
	Step  float64
}

func (this ParameterRange) Values() []float64 {
	var values []float64
	for v := this.Start; v <= this.End; v += this.Step {
		values = append(values, v)
	}
	return values
}

// Parses a list of parameters in the format "parameter=value" into a map.
func ParseParameterList(args []string) (map[string]float64, error) {
	parameters := make(map[string]float64)
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid parameter format: %s", arg)
		}
		value, err := strconv.ParseFloat(parts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid parameter value: %s", parts[1])
		}
		parameters[parts[0]] = value
	}
	return parameters, nil
}

func ParseParameterRanges(args []string) ([]ParameterRange, error) {
	parameterRanges := make([]ParameterRange, 0)
	for _, arg := range args {
		parts := strings.SplitN(arg, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("Invalid parameter range format: %s", arg)
		}
		name := parts[0]
		rangeParts := strings.Split(parts[1], ",")
		if len(rangeParts) != 3 {
			return nil, fmt.Errorf("Invalid parameter range format: %s", arg)

		}
		start, err := strconv.ParseFloat(rangeParts[0], 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid start value: %s", rangeParts[0])
		}
		end, err := strconv.ParseFloat(rangeParts[1], 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid end value: %s", rangeParts[1])
		}
		step, err := strconv.ParseFloat(rangeParts[2], 64)
		if err != nil {
			return nil, fmt.Errorf("Invalid step value: %s", rangeParts[2])
		}
		parameterRanges = append(parameterRanges, ParameterRange{Name: name, Start: start, End: end, Step: step})
	}
	return parameterRanges, nil
}
