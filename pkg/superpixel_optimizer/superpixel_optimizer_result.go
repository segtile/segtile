package superpixel_optimizer

// SpOptimizerResult carries the outcome of a parameter optimization.
type SpOptimizerResult struct {
	Score      float64
	Algorithm  string
	Parameters map[string]float64
	Err        error
}
