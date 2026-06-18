package theme

import (
	"image/color"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"golang.org/x/exp/shiny/materialdesign/icons"
)

// AppTheme defines the colors and styles used throughout the application.
// It encapsulates the Gio material theme and adds its own stuff on top.
type AppTheme struct {
	GioTheme                 material.Theme
	HoverOutlineColor        color.NRGBA
	SelectionOutlineColor    color.NRGBA
	ExistingMaskOutlineColor color.NRGBA

	SelectionStrokeWidth    float32
	HoverStrokeWidth        float32
	ExistingMaskStrokeWidth float32
	StrokeHaloWidth         float32
}

// golang.org/x/exp/shiny/materialdesign/icons does not provide a "play reversed arrow" icon, so we define it ourselves by mirroring the existing play arrow icon.
var aVPlayReverseArrow = []byte{
	0x89, 0x49, 0x56, 0x47, 0x02, 0x0a, 0x00, 0x50, 0x50, 0xb0, 0xb0, 0xc0, 0x90, 0x64, 0xe9, 0xb8,
	0x20, 0x54, 0x64, 0xe1,
}

var (
	IconFirst         = mustIcon(widget.NewIcon(icons.AVSkipPrevious))
	IconPrevUnlabeled = mustIcon(widget.NewIcon(icons.AVFastRewind))
	IconPrev          = mustIcon(widget.NewIcon(aVPlayReverseArrow))
	IconNext          = mustIcon(widget.NewIcon(icons.AVPlayArrow))
	IconNextUnlabeled = mustIcon(widget.NewIcon(icons.AVFastForward))
	IconLast          = mustIcon(widget.NewIcon(icons.AVSkipNext))
)

func mustIcon(ic *widget.Icon, err error) *widget.Icon {
	if err != nil {
		panic(err)
	}
	return ic
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
	// Material theme for Gio widgets
	GioTheme: gioTheme,

	// Colors for outlines
	HoverOutlineColor:        color.NRGBA{R: 0, G: 200, B: 255, A: 220},
	SelectionOutlineColor:    color.NRGBA{R: 255, G: 200, B: 0, A: 240},
	ExistingMaskOutlineColor: color.NRGBA{R: 255, G: 140, B: 0, A: 160},

	// Stroke widths
	SelectionStrokeWidth:    0.75,
	HoverStrokeWidth:        0.75,
	ExistingMaskStrokeWidth: 1.0,
	StrokeHaloWidth:         0.75,
}

func (theme *AppTheme) DefaultButton(btn *widget.Clickable, txt string, size float32) material.ButtonStyle {
	buttonStyle := material.Button(&theme.GioTheme, btn, txt)
	buttonStyle.TextSize = unit.Sp(size)
	return buttonStyle
}

func (theme *AppTheme) IconButton(btn *widget.Clickable, icon *widget.Icon, description string) material.IconButtonStyle {
	style := material.IconButton(&theme.GioTheme, btn, icon, description)
	style.Size = unit.Dp(20)
	style.Inset = layout.UniformInset(unit.Dp(4))
	return style
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
