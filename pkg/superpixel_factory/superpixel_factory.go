package superpixel_factory

import (
	"fmt"

	"github.com/struffel/annotation-tool/pkg/superpixel"
	"github.com/struffel/annotation-tool/pkg/superpixel/slic"
	"github.com/struffel/annotation-tool/superpixel"
	"github.com/struffel/annotation-tool/superpixel/scale_adaptive"
	"github.com/struffel/annotation-tool/superpixel/slic"
	"github.com/struffel/annotation-tool/superpixel/wasserstein_sp"
)

func GetSuperpixelImplementationByName(name string) (superpixel.SpImplementation, error) {
	switch name {
	case "slic":
		return &slic.SpSlic{}, nil
	case "scale_adaptive":
		return &scale_adaptive.SpScaleAdaptive{}, nil
	case "wasserstein_sp":
		return &wasserstein_sp.SpWasserstein{}, nil
	default:
		return nil, fmt.Errorf("Unknown algorithm: %s", name)
	}
}
