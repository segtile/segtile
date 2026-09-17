package ui

import (
	"log"

	"gioui.org/layout"
	"gioui.org/widget"
	"github.com/struffel/annotation-tool/internal/app_config"
)

// Next/Prev handling, both for buttons and keyboard shortcuts
func HandleUiButtonClick(appState *app_config.AppState, gtx layout.Context, button *widget.Clickable) {

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

	if button == &appState.EditExistingMaskButton {
		appState.RecreateExistingMaskFromDisk()
		return
	}

	if button == &appState.PrevUnlabeledButton {
		appState.LoadPreviousUnlabeledImage()
		return
	}

	if button == &appState.PrevButton {
		appState.LoadPreviousImage()
		return
	}

	if button == &appState.NextUnlabeledButton {
		appState.LoadNextUnlabeledImage()
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
		if appState.SelectionMade() || showSaveEmptyMaskWarning() {
			appState.SaveCurrentMask(true, true)
		}

		return
	}

	if button == &appState.ShrinkGrowAmountReset {
		appState.ShrinkGrowAmount.Value = 0.5
		appState.Settings.ShrinkGrowAmount = app_config.SliderToSignedShrinkGrow(appState.ShrinkGrowAmount.Value)
		appState.ReprocessSelection()
		if err := app_config.SaveProjectSettings(appState.Settings); err != nil {
			log.Println("Warning: could not save config:", err)
		}
		appState.Window.Invalidate()
		return
	}

	if button == &appState.SmoothAmountReset {
		appState.SmoothAmount.Value = 0
		appState.Settings.SmoothAmount = 0
		appState.ReprocessSelection()
		if err := app_config.SaveProjectSettings(appState.Settings); err != nil {
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
