package superpixel

import "fmt"

// Unknown superpixel algorithm

type UnknownSpAlgorithmError struct {
	algorithmId string
}

func (e UnknownSpAlgorithmError) Error() string {
	return fmt.Sprintf("Unknown superpixel algorithm: %s", e.algorithmId)
}

// Is reports a match by type, ignoring the dynamic algorithmId field.
/*func (e *UnknownSpAlgorithmError) Is(target error) bool {
	_, ok := target.(*UnknownSpAlgorithmError)
	return ok
}*/
