package cli

type ParameterRange struct {
	Name  string
	Start float64
	End   float64
	Step  float64
}

func (this ParameterRange) Values() []float64 {
	var values []float64
	for v := this.Start; v <= this.End; v += this.Step {
		values = append(values, v)
	}
	return values
}
