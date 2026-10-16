package app_theme

import (
	"fmt"
	"image/color"
	"strconv"
	"strings"

	"gioui.org/widget/material"
)

// AppTheme defines the colors and styles used throughout the application.
// It encapsulates the Gio material theme and adds its own stuff on top.
// The outline colors and stroke widths are persisted in the project settings.
type AppTheme struct {
	GioTheme                   material.Theme `yaml:"-"`
	HoverOutlineColor          color.NRGBA    `yaml:"hover_outline_color"`
	SelectionOutlineColor      color.NRGBA    `yaml:"selection_outline_color"`
	SavedSelectionOutlineColor color.NRGBA    `yaml:"saved_selection_outline_color"`

	SelectionStrokeWidth float32 `yaml:"selection_stroke_width"`
	HoverStrokeWidth     float32 `yaml:"hover_stroke_width"`
}

// MinStrokeWidth is the minimum outline stroke width; the UI sliders cannot go below it.
const MinStrokeWidth = 0.2

// haloColor is the static color used for all halo outlines.
var haloColor = color.NRGBA{R: 0, G: 0, B: 0, A: 200}

// Halo colors are static and shared by all outlines.
func (t *AppTheme) HoverHaloColor() color.NRGBA          { return haloColor }
func (t *AppTheme) SelectionHaloColor() color.NRGBA      { return haloColor }
func (t *AppTheme) SavedSelectionHaloColor() color.NRGBA { return haloColor }

// Halo widths are 75% of their stroke width.
func (t *AppTheme) SelectionHaloWidth() float32 { return t.SelectionStrokeWidth * 0.75 }
func (t *AppTheme) HoverHaloWidth() float32     { return t.HoverStrokeWidth * 0.75 }

// Clone returns a deep copy of the theme.
func (t *AppTheme) Clone() *AppTheme {
	if t == nil {
		return nil
	}
	cloned := *t
	return &cloned
}

// Normalize initializes the non-persisted Gio theme and fills in default values
// for any unset (zero) fields, e.g. when loading a settings file without a theme section.
func (t *AppTheme) Normalize() {
	defaults := Default()
	t.GioTheme = defaults.GioTheme
	if t.HoverOutlineColor.A == 0 {
		t.HoverOutlineColor = defaults.HoverOutlineColor
	}
	if t.SelectionOutlineColor.A == 0 {
		t.SelectionOutlineColor = defaults.SelectionOutlineColor
	}
	if t.SavedSelectionOutlineColor.A == 0 {
		t.SavedSelectionOutlineColor = defaults.SavedSelectionOutlineColor
	}
	if t.SelectionStrokeWidth <= 0 {
		t.SelectionStrokeWidth = defaults.SelectionStrokeWidth
	}
	if t.SelectionStrokeWidth < MinStrokeWidth {
		t.SelectionStrokeWidth = MinStrokeWidth
	}
	if t.HoverStrokeWidth <= 0 {
		t.HoverStrokeWidth = defaults.HoverStrokeWidth
	}
	if t.HoverStrokeWidth < MinStrokeWidth {
		t.HoverStrokeWidth = MinStrokeWidth
	}
}

// FormatHexColor formats a color as "#RRGGBB" for display in the theme editors.
func FormatHexColor(c color.NRGBA) string {
	return fmt.Sprintf("#%02X%02X%02X", c.R, c.G, c.B)
}

// ParseHexColor parses a color in "#RRGGBB" or "RRGGBB" format.
func ParseHexColor(s string) (color.NRGBA, error) {
	s = strings.TrimPrefix(strings.TrimSpace(s), "#")
	if len(s) != 6 {
		return color.NRGBA{}, fmt.Errorf("invalid color %q: expected 6 hex digits (RRGGBB)", s)
	}
	value, err := strconv.ParseUint(s, 16, 32)
	if err != nil {
		return color.NRGBA{}, fmt.Errorf("invalid color %q: %w", s, err)
	}
	return color.NRGBA{
		R: uint8(value >> 16),
		G: uint8(value >> 8),
		B: uint8(value),
		A: 255,
	}, nil
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

// Default returns a new AppTheme instance with the default colors and stroke widths.
func Default() *AppTheme {
	return &AppTheme{
		// Material theme for Gio widgets
		GioTheme: material.NewTheme().WithPalette(palette),

		// Colors for outlines
		HoverOutlineColor:          color.NRGBA{R: 0, G: 200, B: 255, A: 220},
		SelectionOutlineColor:      color.NRGBA{R: 255, G: 200, B: 0, A: 255},
		SavedSelectionOutlineColor: color.NRGBA{R: 255, G: 140, B: 0, A: 255},

		// Stroke widths
		SelectionStrokeWidth: 1,
		HoverStrokeWidth:     0.75,
	}
}
