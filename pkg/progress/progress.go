package progress

type ProgressIndicator interface {
	Start() error
	Update(msg string, percentComplete int) error
	Finish() error
	Close() error
	Done() <-chan struct{}
}

// Indicator which does nothing
type NullProgressIndicator struct{}

func (p NullProgressIndicator) Start() error                                 { return nil }
func (p NullProgressIndicator) Update(msg string, percentComplete int) error { return nil }
func (p NullProgressIndicator) Finish() error                                { return nil }
func (p NullProgressIndicator) Close() error                                 { return nil }
func (p NullProgressIndicator) Done() <-chan struct{}                        { return nil }
