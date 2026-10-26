package components

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/internal/core"
)

type EditorState int

const (
	EditorStateNormal EditorState = iota
	EditorStateUncommitedChange
	EditorStateOutOfRange
)

func DrawEditor(gtx layout.Context, appState *core.AppState, editor *widget.Editor, editorState EditorState) layout.Dimensions {
	editorStyle := material.Editor(&appState.Theme.GioTheme, editor, "value")
	editorStyle.Color = appState.Theme.GioTheme.Palette.Fg

	return (layout.Flex{Axis: layout.Vertical}).Layout(gtx,
		layout.Rigid(editorStyle.Layout),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {

			if editorState == EditorStateUncommitedChange {
				return DrawEditorUnderline(gtx, appState.Theme, appState.Theme.GioTheme.ContrastBg)
			}
			if editorState == EditorStateOutOfRange {
				return DrawEditorUnderline(gtx, appState.Theme, color.NRGBA{R: 255, G: 0, B: 0, A: 255})
			}

			return DrawEditorUnderline(gtx, appState.Theme, appState.Theme.GioTheme.Fg)
		}),
	)

}
