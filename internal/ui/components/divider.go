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
