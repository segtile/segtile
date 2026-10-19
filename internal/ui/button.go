package ui

import (
	"log"
	"time"

	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/reiver/go-giotoast"
	"github.com/struffel/annotation-tool/internal/core"
)

// Next/Prev handling, both for buttons and keyboard shortcuts
func HandleUiButtonClick(appState *core.AppState, gtx layout.Context, button *widget.Clickable) {

	if !button.Clicked(gtx) {
		return
	}
	if appState.CurrentlyOptimizing {
		return
	}

	if button == &appState.ImagesSection.Button {
		appState.ImagesSection.Open = !appState.ImagesSection.Open
		return
	}

	if button == &appState.SuperpixelSection.Button {
		appState.SuperpixelSection.Open = !appState.SuperpixelSection.Open
		return
	}

	if button == &appState.MiscSection.Button {
		appState.MiscSection.Open = !appState.MiscSection.Open
		return
	}

	if button == &appState.ChangeSpAlgorithmButton {
		appState.SpAlgorithmMenuOpen = !appState.SpAlgorithmMenuOpen
		return
	}

	if button == &appState.RevertSavedMaskButton {
		if err := appState.RevertToSavedMask(); err != nil {
			log.Printf("Failed to revert to saved mask: %v", err)
			appState.Toasts.ShowType(time.Now(), giotoast.TypeError, err.Error(), 3*time.Second)
		}
		return
	}

	if button == &appState.PrevUnlabeledButton {
		// In evaluation mode every image has a mask on disk (the ground truth),
		// so "unlabeled" navigation would never find anything; fall back to plain navigation.
		if appState.Eval != nil {
			appState.LoadPreviousImage()
		} else {
			appState.LoadPreviousUnlabeledImage()
		}
		return
	}

	if button == &appState.PrevButton {
		appState.LoadPreviousImage()
		return
	}

	if button == &appState.NextUnlabeledButton {
		if appState.Eval != nil {
			appState.LoadNextImage()
		} else {
			appState.LoadNextUnlabeledImage()
		}
		return
	}

	if button == &appState.NextButton {
		appState.LoadNextImage()
		return
	}

	if button == &appState.OpenProjectButton {
		triggerOpenProject(appState)
		return
	}

	if button == &appState.NewProjectButton {
		triggerNewProject(appState)
		return
	}

	if button == &appState.OptimizeButton {
		// Optimizing superpixels against the masks on disk would tune them towards
		// the ground truth, which defeats the purpose of the evaluation.
		if appState.Eval != nil {
			appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "Optimization is disabled in evaluation mode.", 2*time.Second)
			return
		}
		triggerOptimization(appState)
		return
	}

	if button == &appState.FirstButton {
		appState.LoadFirstImage()
		return
	}

	if button == &appState.LastButton {
		appState.LoadLastImage()
		return
	}

	if button == &appState.SaveButton {
		// Evaluation mode never saves anything.
		if appState.Eval != nil {
			return
		}
		if appState.SelectionMade() || showSaveEmptyMaskWarning() {
			appState.SaveCurrentMask(true)
		}

		return
	}

	if button == &appState.ShrinkGrowAmountReset {
		appState.ShrinkGrowAmount.Value = 0.5
		newValue := core.SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value)
		if newValue != appState.ActiveSettings.ShrinkGrowAmount {
			appState.SelectionMaskDiffersVersionOnDisk = true
		}
		appState.ActiveSettings.ShrinkGrowAmount = newValue
		appState.ReprocessSelection()
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
		appState.Window.Invalidate()
		return
	}

	if button == &appState.SmoothAmountReset {
		appState.SmoothAmount.Value = 0
		if appState.ActiveSettings.SmoothAmount != 0 {
			appState.SelectionMaskDiffersVersionOnDisk = true
		}
		appState.ActiveSettings.SmoothAmount = 0
		appState.ReprocessSelection()
		if err := appState.SaveProjectSettings(); err != nil {
			log.Println("Warning: could not save config:", err)
		}
		appState.Window.Invalidate()
		return
	}

	for i := range appState.SpAlgorithmMenuEntries {

		if button != &appState.SpAlgorithmMenuEntries[i].Button {
			continue
		}

		err := appState.SetSpAlgorithm(appState.SpAlgorithmMenuEntries[i].AlgorithmId)
		if err != nil {
			log.Printf("Failed to change superpixel algorithm: %v", err)
			return
		}
		appState.SpAlgorithmMenuOpen = false

		if appState.Window != nil {
			appState.Window.Invalidate()
		}
		return
	}

}
