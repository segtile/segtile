package superpixel_optimizer

import "fmt"

type SpOptimizerParameter interface {
	fmt.Stringer
	Title() string
	Description() string
}
