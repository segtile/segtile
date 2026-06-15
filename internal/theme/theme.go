package theme

import (
	"image/color"

	"gioui.org/widget/material"
)

// AppTheme defines the colors and styles used throughout the application.
// It encapsulates the Gio material theme and adds its own stuff on top.
type AppTheme struct {
	GioTheme              material.Theme
	HoverOutlineColor     color.NRGBA
	SelectionOutlineColor color.NRGBA
}

var palette = material.Palette{
	Bg:         color.NRGBA{R: 35, G: 35, B: 35, A: 255},
	Fg:         color.NRGBA{R: 220, G: 220, B: 220, A: 255},
	ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255},
	ContrastBg: color.NRGBA{R: 25, G: 25, B: 25, A: 255},
}
var gioTheme = material.NewTheme().WithPalette(palette)

var Theme = &AppTheme{
	GioTheme:              gioTheme,
	HoverOutlineColor:     color.NRGBA{R: 0, G: 200, B: 255, A: 220},
	SelectionOutlineColor: color.NRGBA{R: 255, G: 200, B: 0, A: 220},
}
