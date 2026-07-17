package superpixel_optimizer

type SpOptimizerProgressIndicator interface {
	Update(message string, percentComplete int)
	Done()
}

type SpOptimizerSilentProgressIndicator struct{}

func (p *SpOptimizerSilentProgressIndicator) Update(message string, percentComplete int) {}
func (p *SpOptimizerSilentProgressIndicator) Done()                                      {}
