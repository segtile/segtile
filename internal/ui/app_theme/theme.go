package app_theme

import (
	"image/color"

	"gioui.org/widget/material"
)

// AppTheme defines the colors and styles used throughout the application.
// It encapsulates the Gio material theme and adds its own stuff on top.
type AppTheme struct {
	GioTheme                 material.Theme
	HoverOutlineColor        color.NRGBA
	HoverHaloColor           color.NRGBA
	SelectionOutlineColor    color.NRGBA
	SelectionHaloColor       color.NRGBA
	ExistingMaskOutlineColor color.NRGBA
	ExistingMaskHaloColor    color.NRGBA

	SelectionStrokeWidth    float32
	SelectionHaloWidth      float32
	HoverStrokeWidth        float32
	HoverHaloWidth          float32
	ExistingMaskStrokeWidth float32
	ExistingMaskHaloWidth   float32
}

// Menu bar
const (
	MenuBarHorizontalSpacing = 6
	MenuBarVerticalSpacing   = 2
	MenuBarIconSpacing       = 3
	MenuBarIconSize          = 18
	SidebarWidth             = 280
)

// Theme configuration for the application
var palette = material.Palette{
	Bg:         color.NRGBA{R: 35, G: 35, B: 35, A: 255},    // Used as background for entire application
	Fg:         color.NRGBA{R: 220, G: 220, B: 220, A: 255}, // Used for text and icons on top of the background
	ContrastFg: color.NRGBA{R: 255, G: 255, B: 255, A: 255}, // Used for text and icons on top of buttons
	ContrastBg: color.NRGBA{R: 63, G: 81, B: 181, A: 255},   // Used for buttons and other interactive elements
}
var gioTheme = material.NewTheme().WithPalette(palette)

// Global theme instance used throughout the application
var Theme = &AppTheme{
	// Material theme for Gio widgets
	GioTheme: gioTheme,

	// Colors for outlines
	HoverOutlineColor: color.NRGBA{R: 0, G: 200, B: 255, A: 220},
	HoverHaloColor:    color.NRGBA{R: 0, G: 0, B: 0, A: 255},

	SelectionOutlineColor: color.NRGBA{R: 255, G: 200, B: 0, A: 255},
	SelectionHaloColor:    color.NRGBA{R: 0, G: 0, B: 0, A: 255},

	ExistingMaskOutlineColor: color.NRGBA{R: 255, G: 140, B: 0, A: 160},
	ExistingMaskHaloColor:    color.NRGBA{R: 0, G: 0, B: 0, A: 255},

	// Stroke widths
	SelectionStrokeWidth: 1,
	SelectionHaloWidth:   0.75,

	HoverStrokeWidth: 0.75,
	HoverHaloWidth:   0.5,

	ExistingMaskStrokeWidth: 1.0,
	ExistingMaskHaloWidth:   1,
}
