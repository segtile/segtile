package components

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"github.com/segtile/segtile/internal/ui/app_theme"
)

// DrawIconButton draws a button with an icon and handles its click event.
func DrawIconButton(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, icon *widget.Icon, description string, filled bool) layout.Dimensions {
	style := material.IconButton(&theme.GioTheme, btn, icon, description)
	style.Size = unit.Dp(20)
	style.Inset = layout.UniformInset(unit.Dp(4))
	if filled {
		style.Color = theme.GioTheme.Palette.ContrastFg
		style.Background = theme.GioTheme.Palette.ContrastBg
	} else {
		style.Color = theme.GioTheme.Palette.Fg
		style.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	}
	return style.Layout(gtx)
}

func DrawBoldButton(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, txt string, size float32) layout.Dimensions {
	buttonStyle := material.Button(&theme.GioTheme, btn, txt)
	buttonStyle.TextSize = unit.Sp(size)
	buttonStyle.Inset = layout.UniformInset(unit.Dp(16))
	return buttonStyle.Layout(gtx)
}

func DrawSubtleButton(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, txt string, size float32) layout.Dimensions {
	buttonStyle := material.Button(&theme.GioTheme, btn, txt)
	buttonStyle.TextSize = unit.Sp(size)
	buttonStyle.Inset = layout.UniformInset(unit.Dp(4))
	return buttonStyle.Layout(gtx)
}

func DrawMenuBarButton(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, icon *widget.Icon, txt string) layout.Dimensions {

	buttonStyle := material.ButtonLayout(&theme.GioTheme, btn)
	buttonStyle.CornerRadius = 2
	buttonStyle.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 0}

	menuButtonInset := layout.Inset{
		Top:    unit.Dp(app_theme.MenuBarVerticalSpacing),
		Bottom: unit.Dp(app_theme.MenuBarVerticalSpacing),
		Left:   unit.Dp(app_theme.MenuBarHorizontalSpacing),
		Right:  unit.Dp(app_theme.MenuBarHorizontalSpacing),
	}

	return buttonStyle.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return menuButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if icon == nil {
						return layout.Dimensions{}
					}
					iconInset := layout.Inset{
						Right: unit.Dp(app_theme.MenuBarIconSpacing),
					}
					return iconInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(app_theme.MenuBarIconSize))
						gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(app_theme.MenuBarIconSize))
						return icon.Layout(gtx, theme.GioTheme.Palette.Fg)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.Body2(&theme.GioTheme, txt)
					label.Color = theme.GioTheme.Palette.ContrastFg
					return label.Layout(gtx)
				}),
			)
		})
	})
}
