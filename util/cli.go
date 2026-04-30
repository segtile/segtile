package util

import (
	"fmt"
	"strconv"
	"strings"
)

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
