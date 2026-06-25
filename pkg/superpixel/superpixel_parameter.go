package superpixel

import "fmt"

type SpParameter interface {
	fmt.Stringer
	Title() string
	Description() string
	Min() float64
	Max() float64
}
