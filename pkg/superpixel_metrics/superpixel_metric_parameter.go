package superpixel_metrics

import "fmt"

type SpMetricParameter interface {
	fmt.Stringer
	Title() string
	Description() string
}
