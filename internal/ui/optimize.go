package ui

import (
	"context"
	"errors"
	"log"

	"github.com/ncruces/zenity"
	"github.com/struffel/annotation-tool/internal/core"
)

func triggerOptimization(appState *core.AppState) {

	// Create a progress indicator and start it
	progress := &ZenityProgressIndicator{}
	if err := progress.Start(); err != nil {
		log.Printf("Failed to show optimization progress: %v", err)
		return
	}

	// Indicate that the optimization is running. This will be reset when the result is processed in the main loop.
	appState.CurrentlyOptimizing = true

	// Create the context for the optimization, which can be canceled if the user closes the progress dialog
	ctx, cancelFunction := context.WithCancel(context.Background())

	// First goroutine which blocks until either the progress dialog is closed or the optimization is done, and cancels the context if the progress dialog is closed.
	go func() {
		select {
		case <-progress.Done():
			cancelFunction()
		case <-ctx.Done():
		}
	}()

	// Second goroutine which runs the actual optimization and writes the result to the channel
	go func() {

		defer cancelFunction()

		result, err := appState.OptimizeSpParameters(ctx, progress)
		if errors.Is(err, zenity.ErrCanceled) {
			err = context.Canceled
		}

		if err == nil {
			err = progress.Finish()
		} else {
			_ = progress.Close()
		}

		result.Err = err
		appState.OptimizationResult <- result
		appState.Window.Invalidate()
	}()

}
