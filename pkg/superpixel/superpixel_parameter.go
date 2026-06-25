package superpixel

import "fmt"

type SpParameter interface {
	fmt.Stringer
	Title() string
	Description() string
}
