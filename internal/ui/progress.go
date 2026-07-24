package ui

import "github.com/ncruces/zenity"

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
	return p.Update("Starting...", 0)
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
