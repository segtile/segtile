package main

import (
	"flag"
	_ "image/jpeg"
	"log"
	"os"
	"time"

	"gioui.org/app"
	"gioui.org/op"
	"gioui.org/unit"
	"github.com/struffel/annotation-tool/internal/core"
	"github.com/struffel/annotation-tool/internal/evaluation"
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

	// Evaluation mode flags (the project config path remains the positional argument).
	evalEnabled := flag.Bool("evaluation", false, "run in evaluation mode: existing masks are treated as ground truth, nothing is saved, annotation time is measured")
	evalThreshold := flag.Float64("eval-threshold", 0.95, "dice score threshold at which the ground truth overlay turns green")
	flag.Parse()

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

		// In evaluation mode, spin up the secondary ground-truth overlay window.
		if *evalEnabled {
			appState.Eval = evaluation.NewState(*evalThreshold)
			evalWindow := new(app.Window)
			evalWindow.Option(
				app.Title("Ground Truth (Evaluation)"),
				app.Size(unit.Dp(800), unit.Dp(800)),
			)
			appState.Eval.AttachWindow(evalWindow)
			go ui.RunEvalWindow(appState, evalWindow)
		}

		// If a config was provided on the CLI, load the project immediately.
		if flag.NArg() >= 1 {
			configPath := flag.Arg(0)
			appState.CurrentlyLoadingProject = true
			result := core.LoadProjectSettings(configPath)
			appState.ProjectLoadResult <- result
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

			if appState.Eval != nil {
				// Evaluation mode: record the final image, print the summary, and
				// never save anything.
				appState.Eval.EndImage()
				appState.Eval.PrintSummary()
				appState.Eval.CloseWindow()
			} else if appState.SelectionMaskDiffersVersionOnDisk {
				// Save the current mask before exiting, if it was actually changed.
				appState.SaveCurrentMask(false)
			}

			return e.Err
		case app.FrameEvent:
			frameStart := time.Now()

			// Apply a project if is has finished loading
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

			// Rebuild the processed selection mask if brush strokes have modified it
			if appState.SelectionMaskDirty {
				appState.ReprocessSelection()
			}

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
