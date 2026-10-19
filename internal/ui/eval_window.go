package ui

import (
	"fmt"
	"image/color"
	"time"

	"gioui.org/app"
	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/internal/core"
)

// RunEvalWindow runs the event loop of the secondary ground-truth overlay
// window used in evaluation mode. It returns when the window is closed; the
// main application keeps running.
func RunEvalWindow(appState *core.AppState, window *app.Window) {

	// Use a separate material theme so the text shaper is not shared with the
	// main window's goroutine.
	theme := material.NewTheme()

	var ops op.Ops
	for {
		switch e := window.Event().(type) {
		case app.DestroyEvent:
			appState.Eval.DetachWindow(window)
			return
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)
			drawEvalOverlay(appState, theme, gtx)
			e.Frame(&ops)

			// Repaint continuously so the elapsed time keeps ticking and the
			// overlay color flips as soon as the threshold is crossed.
			window.Invalidate()
		}
	}
}

// drawEvalOverlay renders the original image with the GT overlay
// (red below the threshold, green above) plus a small status line.
func drawEvalOverlay(appState *core.AppState, theme *material.Theme, gtx layout.Context) layout.Dimensions {
	snap := appState.Eval.Snapshot()

	// Background
	bgClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
	paint.ColorOp{Color: color.NRGBA{R: 20, G: 20, B: 20, A: 255}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bgClip.Pop()

	if !snap.HasImage {
		label := material.Body1(theme, "Open a project to see the ground truth overlay.")
		label.Color = color.NRGBA{R: 220, G: 220, B: 220, A: 255}
		gtx.Constraints.Min = gtx.Constraints.Max
		return layout.Center.Layout(gtx, label.Layout)
	}

	// Fit the image into the window, centered.
	size := snap.ImageSize
	scaleX := float32(gtx.Constraints.Max.X) / float32(size.X)
	scaleY := float32(gtx.Constraints.Max.Y) / float32(size.Y)
	scale := min(scaleX, scaleY)
	offset := f32.Pt(
		(float32(gtx.Constraints.Max.X)-float32(size.X)*scale)/2,
		(float32(gtx.Constraints.Max.Y)-float32(size.Y)*scale)/2,
	)
	transformation := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(scale, scale)).Offset(offset)

	stack := op.Affine(transformation).Push(gtx.Ops)
	snap.Base.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	snap.Overlay.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	stack.Pop()

	// Status line at the top-left: dice vs threshold, elapsed time, mode.
	elapsed := time.Since(snap.Start).Round(time.Second)
	status := fmt.Sprintf("dice %.3f / %.2f  •  %s  •  mode: %s", snap.Dice, snap.Threshold, elapsed, snap.Mode)
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Stack{}.Layout(gtx,
			layout.Expanded(func(gtx layout.Context) layout.Dimensions {
				bg := clip.Rect{Max: gtx.Constraints.Min}.Push(gtx.Ops)
				paint.ColorOp{Color: color.NRGBA{A: 160}}.Add(gtx.Ops)
				paint.PaintOp{}.Add(gtx.Ops)
				bg.Pop()
				return layout.Dimensions{Size: gtx.Constraints.Min}
			}),
			layout.Stacked(func(gtx layout.Context) layout.Dimensions {
				label := material.Body1(theme, status)
				label.Color = color.NRGBA{R: 255, G: 255, B: 255, A: 255}
				return layout.UniformInset(unit.Dp(4)).Layout(gtx, label.Layout)
			}),
		)
	})
}
