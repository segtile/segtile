package ui

import "github.com/ncruces/zenity"

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

// Wrapper for a Zenity Dialog
type ZenityProgressIndicator struct {
	zenityDialog zenity.ProgressDialog
}

func (p *ZenityProgressIndicator) Start() error {
	zenityDialog, err := zenity.Progress(zenity.AutoClose())
	if err != nil {
		return err
	}
	p.zenityDialog = zenityDialog
	return p.Update("Preparing...", 0)
}
func (p *ZenityProgressIndicator) Update(msg string, percentComplete int) error {
	err := p.zenityDialog.Value(percentComplete)
	if err != nil {
		return err
	}
	err = p.zenityDialog.Text(msg)
	if err != nil {
		return err
	}
	return nil

}
func (p *ZenityProgressIndicator) Finish() error {
	return p.zenityDialog.Value(p.zenityDialog.MaxValue())
}
func (p *ZenityProgressIndicator) Close() error {
	return p.zenityDialog.Close()
}
func (p *ZenityProgressIndicator) Done() <-chan struct{} {
	return p.zenityDialog.Done()
}
