package main

import (
	_ "image/jpeg"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/struffel/annotation-tool/internal/app_config"
	"github.com/struffel/annotation-tool/internal/ui"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/slic"
	_ "github.com/struffel/annotation-tool/pkg/superpixel/snic"
)

func main() {

	go func() {

		// Initialize application state and window.
		appWindow := new(app.Window)
		appWindow.Option(
			app.Title("Annotation Tool"),
			app.Size(unit.Dp(1200), unit.Dp(800)),
		)

		appState := app_config.NewAppState()
		appState.Window = appWindow

		// If a config was provided on the CLI, load the project immediately.
		if len(os.Args) >= 2 {
			configPath := os.Args[1]
			cfg, err := app_config.Load(configPath)
			if err != nil {
				log.Printf("Warning: failed to load config %q: %v", configPath, err)
			} else {
				appState.ApplyProject(cfg)
			}
		}

		// Main app loop
		if err := applicationLoop(appState, appWindow); err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

func applicationLoop(appState *app_config.AppState, appWindow *app.Window) error {

	var ops op.Ops

	for {
		switch e := appWindow.Event().(type) {
		case app.DestroyEvent:

			// Save the current mask before exiting.
			appState.SaveCurrentMask(false, false)

			return e.Err
		case app.FrameEvent:
			// Apply loaded projects
			select {
			case result := <-appState.ProjectLoadResult:
				appState.ApplyProjectLoadingResult(result)
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

			// Handle interactions
			ui.HandlePointer(appState, gtx)
			ui.HandleEditors(appState, gtx)
			ui.HandleKeyboard(appState, gtx)

			// Build UI layout
			ui.Layout(appState, gtx)

			// Draw!
			e.Frame(&ops)
		}
	}
}
