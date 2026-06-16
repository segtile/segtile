package superpixel_factory

import (
	"fmt"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel/slic"
)

const (
	// Algorithm names
	nameSlic = "slic"
)

func NewSuperpixelImplementationByName(name string) (superpixel.SpImplementation, error) {
	switch name {
	case nameSlic:
		return &slic.SpSlic{}, nil
	default:
		return nil, fmt.Errorf("Unknown algorithm: %s", name)
	}
}

func ImplementationName(impl superpixel.SpImplementation) (string, error) {
	switch impl.(type) {
	case *slic.SpSlic:
		return nameSlic, nil
	default:
		return "unknown", fmt.Errorf("Unknown implementation: %T", impl)
	}
}
