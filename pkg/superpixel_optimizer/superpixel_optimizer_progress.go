package superpixel_optimizer

type SpOptimizerProgressIndicator interface {
	Update(message string, percentComplete int) error
}

type SpOptimizerSilentProgressIndicator struct{}

func (p *SpOptimizerSilentProgressIndicator) Update(message string, percentComplete int) error {
	return nil
}
