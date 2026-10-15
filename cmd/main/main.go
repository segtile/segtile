package main

import (
	_ "image/jpeg"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/struffel/annotation-tool/internal/core"
	"github.com/struffel/annotation-tool/internal/ui"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/rss"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/scale_adaptive"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"

	//_ "github.com/struffel/annotation-tool/pkg/superpixel/snic"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/vcell"

	"net/http"
	_ "net/http/pprof"
)

func main() {

	// Start profiling
	go func() {
		log.Println(http.ListenAndServe("localhost:6060", nil))
	}()

	go func() {

		// Initialize application state and window.
		appWindow := new(app.Window)
		appWindow.Option(
			app.Title("Annotation Tool"),
			app.Size(unit.Dp(1200), unit.Dp(800)),
		)

		appState := core.NewAppState()
		appState.Window = appWindow

		// If a config was provided on the CLI, load the project immediately.
		if len(os.Args) >= 2 {
			configPath := os.Args[1]
			go func() {
				result := core.LoadProjectSettings(configPath)
				appState.ProjectLoadResult <- result
			}()
		}

		// Main app loop
		if err := applicationLoop(appState, appWindow); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()

	// Start the Gio application main loop.
	app.Main()
}

func applicationLoop(appState *core.AppState, appWindow *app.Window) error {

	var ops op.Ops

	for {
		switch e := appWindow.Event().(type) {
		case app.DestroyEvent:

			// Save the current mask before exiting, but only if it was actually changed.
			if appState.SelectionModified {
				appState.SaveCurrentMask(false, false)
			}

			return e.Err
		case app.FrameEvent:
			frameStart := time.Now()
			// Apply loaded projects
			select {
			case result := <-appState.ProjectLoadResult:
				appState.ApplyProjectSettings(result)
			default:
			}

			// Apply optimizer results
			select {
			case result := <-appState.OptimizationResult:
				appState.ApplyOptimizationResult(result)
				appState.CurrentlyOptimizing = false
			default:
			}

			// New context for this frame
			gtx := app.NewContext(&ops, e)

			// Rebuild the selection mask at most once per frame if brush strokes are pending
			appState.FlushPendingManualFixes()

			// Handle interactions
			ui.HandlePointer(appState, gtx)
			ui.HandleEditors(appState, gtx)
			ui.HandleKeyboard(appState, gtx)

			// Build UI layout
			ui.Layout(appState, gtx)

			// Draw!
			e.Frame(&ops)

			// Measure frame time
			frameEnd := time.Now()
			frameTime := frameEnd.Sub(frameStart)

			// Warn if the frame time no longer allows 60 FPS (16.67 ms per frame)
			if frameTime > 16*time.Millisecond {
				log.Printf("Frame time was %v > 16ms", frameTime)
			}

		}
	}
}
