package theme

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

// AppTheme defines the colors and styles used throughout the application.
// It encapsulates the Gio material theme and adds its own stuff on top.
type AppTheme struct {
	GioTheme              material.Theme
	HoverOutlineColor     color.NRGBA
	SelectionOutlineColor color.NRGBA
}

// Constants
const MenuBarSpacing = 6
const SidebarWidth = 220

var transparent = color.NRGBA{R: 0, G: 0, B: 0, A: 0}

var palette = material.Palette{
	Bg:         color.NRGBA{R: 35, G: 35, B: 35, A: 255},
	Fg:         color.NRGBA{R: 220, G: 220, B: 220, A: 255},
	ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	ContrastBg: color.NRGBA{R: 63, G: 81, B: 181, A: 255},
}
var gioTheme = material.NewTheme().WithPalette(palette)

var Theme = &AppTheme{
	GioTheme:              gioTheme,
	HoverOutlineColor:     color.NRGBA{R: 0, G: 200, B: 255, A: 220},
	SelectionOutlineColor: color.NRGBA{R: 255, G: 200, B: 0, A: 220},
}

func (theme *AppTheme) DefaultButton(btn *widget.Clickable, txt string) material.ButtonStyle {
	return material.Button(&theme.GioTheme, btn, txt)
}

func (theme *AppTheme) MenuBarButton(btn *widget.Clickable, txt string) material.ButtonStyle {
	buttonStyle := material.Button(&theme.GioTheme, btn, txt)
	buttonStyle.CornerRadius = 0
	buttonStyle.Inset = layout.UniformInset(unit.Dp(2))
	buttonStyle.Inset.Left = unit.Dp(MenuBarSpacing)
	buttonStyle.Inset.Right = unit.Dp(MenuBarSpacing)
	buttonStyle.Background = transparent
	return buttonStyle
}
