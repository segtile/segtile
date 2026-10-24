package components

import (
	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/internal/core"
)

func DrawEditor(gtx layout.Context, appState *core.AppState, editor *widget.Editor, uncommitedChange bool) layout.Dimensions {
	editorStyle := material.Editor(&appState.Theme.GioTheme, editor, "value")
	editorStyle.Color = appState.Theme.GioTheme.Palette.Fg

	return (layout.Flex{Axis: layout.Vertical}).Layout(gtx,
		layout.Rigid(editorStyle.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return DrawEditorUnderline(gtx, appState.Theme, uncommitedChange)
		}),
	)

}
