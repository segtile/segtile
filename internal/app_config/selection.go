package app_config

import (
	"time"

	"github.com/reiver/go-giotoast"
)

func (appState *AppState) SelectionMade() bool {
	for _, selected := range appState.SelectionIndex {
		if selected {
			return true
		}
	}
	return false
}

func (appState *AppState) ClearSelection(showToast bool) {
	for i := range appState.SelectionIndex {
		appState.SelectionIndex[i] = false
	}
	appState.RecalculateProcessedSelection()
	appState.Window.Invalidate()
	if showToast {
		appState.Toasts.ShowType(time.Now(), giotoast.TypeWarning, "Cleared selection.", 1*time.Second)
	}
}
