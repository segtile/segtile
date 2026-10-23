package components

import (
	"image"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/internal/core"
)

func DrawInfoRow(gtx layout.Context, appState *core.AppState, title string, value string, icon *widget.Icon) layout.Dimensions {
	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Start}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					gtx.Constraints.Max = image.Pt(gtx.Dp(16), gtx.Dp(16))
					return icon.Layout(gtx, appState.Theme.GioTheme.Palette.Fg)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Spacer{Width: unit.Dp(2)}.Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return material.Caption(&appState.Theme.GioTheme, title).Layout(gtx)
				}),
			)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return material.Body2(&appState.Theme.GioTheme, value).Layout(gtx)
		}),
	)
}
