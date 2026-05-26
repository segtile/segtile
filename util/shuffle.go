package util

import "math/rand/v2"

func ShuffleInUnison[T any](a, b []T) {
	if len(a) != len(b) {
		panic("Slices must have the same length")
	}
	for i := len(a) - 1; i > 0; i-- {
		j := rand.IntN(i + 1)
		a[i], a[j] = a[j], a[i]
		b[i], b[j] = b[j], b[i]
	}
}
