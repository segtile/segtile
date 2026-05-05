package superpixel_factory

import (
	"fmt"

	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel/slic"
)

func GetSuperpixelImplementationByName(name string) (superpixel.SpImplementation, error) {
	switch name {
	case "slic":
		return &slic.SpSlic{}, nil
	default:
		return nil, fmt.Errorf("Unknown algorithm: %s", name)
	}
}
