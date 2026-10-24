package components

import (
	"image"
	"image/color"

	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
)

// DrawDivider draws a horizontal divider with the specified thickness and theme.
func DrawDivider(gtx layout.Context, theme *app_theme.AppTheme, thickness unit.Dp) layout.Dimensions {
	w := gtx.Dp(thickness)
	size := image.Pt(gtx.Constraints.Max.X, w)

	// Alpha = 128 -> 50 % opacity
	paint.FillShape(gtx.Ops, color.NRGBA{R: theme.GioTheme.Fg.R, G: theme.GioTheme.Fg.G, B: theme.GioTheme.Fg.B, A: 128}, clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}

// DrawEditorUnderline draws a horizontal underline for an editor with the specified thickness and theme.
func DrawEditorUnderline(gtx layout.Context, theme *app_theme.AppTheme, uncommitedChange bool) layout.Dimensions {
	w := gtx.Dp(0.5)
	size := image.Pt(gtx.Constraints.Max.X, w)

	// Alpha = 128 -> 50 % opacity
	if uncommitedChange {
		paint.FillShape(gtx.Ops, color.NRGBA{R: theme.GioTheme.ContrastBg.R, G: theme.GioTheme.ContrastBg.G, B: theme.GioTheme.ContrastBg.B, A: 128}, clip.Rect{Max: size}.Op())
	} else {
		paint.FillShape(gtx.Ops, color.NRGBA{R: theme.GioTheme.Fg.R, G: theme.GioTheme.Fg.G, B: theme.GioTheme.Fg.B, A: 128}, clip.Rect{Max: size}.Op())
	}
	return layout.Dimensions{Size: size}
}
