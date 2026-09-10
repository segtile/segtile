package superpixel

import "fmt"

// Unknown superpixel algorithm

type UnknownSpAlgorithmError struct {
	algorithmId string
}

func (e UnknownSpAlgorithmError) Error() string {
	return fmt.Sprintf("Unknown superpixel algorithm: %s", e.algorithmId)
}
