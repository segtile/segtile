package components

import (
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
)

// DrawStroke adds the drawing operations for an outline to the graphics context
// The halo is an additional outline around the main line.
func DrawStroke(
	stack *op.TransformStack,
	gtx layout.Context,
	transformation f32.Affine2D,
	viewScale float32,
	pathSpec clip.PathSpec,
	strokeColor color.NRGBA,
	strokeWidth float32,
	haloColor color.NRGBA,
	haloWidth float32) {

	if haloWidth > 0 {
		// Halo/Outline
		*stack = op.Affine(transformation).Push(gtx.Ops)
		paint.FillShape(gtx.Ops,
			haloColor,
			clip.Stroke{
				Path:  pathSpec,
				Width: (strokeWidth + haloWidth) / viewScale,
			}.Op())
		stack.Pop()
	}
	if strokeWidth > 0 {
		// Colored stroke on top
		*stack = op.Affine(transformation).Push(gtx.Ops)
		paint.FillShape(gtx.Ops,
			strokeColor,
			clip.Stroke{
				Path:  pathSpec,
				Width: strokeWidth / viewScale,
			}.Op())
		stack.Pop()
	}
}
