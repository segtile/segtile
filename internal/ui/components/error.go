package components

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/widget/material"
	"github.com/segtile/segtile/internal/core"
)

// drawProjectLoadError renders the error from a failed project load/creation attempt,
// shown under the New/Open Project buttons on the start screen.
func DrawProjectLoadError(gtx layout.Context, appState *core.AppState) layout.Dimensions {

	if appState.ActiveSettings == nil || appState.ActiveSettings.Err == nil {
		return layout.Dimensions{}
	}

	label := material.Body2(&appState.Theme.GioTheme, appState.ActiveSettings.Err.Error())
	label.Color = color.NRGBA{R: 220, G: 80, B: 80, A: 255}
	label.Alignment = text.Start
	return label.Layout(gtx)
}
