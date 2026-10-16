package main

import (
	"testing"
	"time"

	"gioui.org/app"
	"gioui.org/io/system"
	"gioui.org/unit"
	"github.com/struffel/annotation-tool/internal/core"
)

func TestEmptyOpenClose(t *testing.T) {
	appWindow := new(app.Window)
	appWindow.Option(app.Size(unit.Dp(600), unit.Dp(600)))
	appState := core.NewAppState()
	appState.Window = appWindow

	done := make(chan struct{})
	go func() {
		applicationLoop(appState, appWindow)
		close(done)
	}()

	time.Sleep(time.Second) // Close after 1 second
	appWindow.Perform(system.ActionClose)
	<-done
}
