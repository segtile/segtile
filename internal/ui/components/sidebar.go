package components

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
)

func DrawSidebarSectionHeader(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, title string, open bool, dividerThickness unit.Dp) layout.Dimensions {
	buttonStyle := material.ButtonLayout(&theme.GioTheme, btn)
	buttonStyle.CornerRadius = 2
	buttonStyle.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 0}

	return buttonStyle.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		label := material.Body2(&theme.GioTheme, title)
		label.Color = theme.GioTheme.Palette.ContrastFg
		label.Alignment = text.Start

		icon := app_theme.IconSectionClosed
		if open {
			icon = app_theme.IconSectionOpen
		}

		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				size := gtx.Dp(16)
				gtx.Constraints = layout.Exact(gtx.Constraints.Constrain(image.Pt(size, size)))
				return icon.Layout(gtx, theme.GioTheme.Palette.ContrastFg)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return DrawDivider(gtx, theme, unit.Dp(dividerThickness))
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		)
	})
}
