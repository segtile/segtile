package ui

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"

	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/internal/app_config"
)

func Layout(appState *app_config.AppState, gtx layout.Context) {

	// This is the size of the window
	size := gtx.Constraints.Max

	// We set up a clipping area that covers the entire window.
	// This ensures that all drawing operations are confined to the window bounds, preventing any rendering outside the visible area.
	bgClip := clip.Rect{Max: size}.Push(gtx.Ops)

	// Fill the background with a solid color (dark gray)
	paint.ColorOp{Color: color.NRGBA{R: 35, G: 35, B: 35, A: 255}}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bgClip.Pop()

	const sidebarWidth = 220

	// Entire application layout
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,

		// Top menu bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Draw menu bar background
			menuBarHeight := gtx.Dp(32)
			menuBarClip := clip.Rect{Max: image.Point{X: size.X, Y: menuBarHeight}}.Push(gtx.Ops)
			paint.ColorOp{Color: color.NRGBA{R: 25, G: 25, B: 25, A: 255}}.Add(gtx.Ops)
			paint.PaintOp{}.Add(gtx.Ops)
			menuBarClip.Pop()
			return layout.Dimensions{Size: image.Point{X: size.X, Y: menuBarHeight}}
		}),

		// Main content window
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				// Sidebar
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {

					// Set fixed width for the sidebar
					gtx.Constraints.Min.X = sidebarWidth
					gtx.Constraints.Max.X = sidebarWidth

					// Draw sidebar background
					sidebarClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
					paint.ColorOp{Color: color.NRGBA{R: 25, G: 25, B: 25, A: 255}}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					sidebarClip.Pop()

					// Layout sliders inside the sidebar
					inset := layout.UniformInset(unit.Dp(8))
					return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						var children []layout.FlexChild

						for _, param := range appState.SuperpixelImplementation.GetSupportedParameters() {
							children = append(children,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									label := material.Body1(appState.Theme, param.Title())
									label.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
									return label.Layout(gtx)
								}),

								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									currentEditor := appState.SuperpixelParameterEditors[param]

									editorStyle := material.Editor(appState.Theme, currentEditor, "value")
									editorStyle.Color = color.NRGBA{R: 200, G: 200, B: 200, A: 255}
									return editorStyle.Layout(gtx)
								}),
							)
						}

						// Spacer between param editors and navigation controls.
						children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout))

						// Image counter label.
						if len(appState.ImageFiles) > 0 {
							counter := fmt.Sprintf("%d / %d  —  %s",
								appState.CurrentIndex+1,
								len(appState.ImageFiles),
								filepath.Base(appState.ImageFiles[appState.CurrentIndex]),
							)
							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								lbl := material.Body2(appState.Theme, counter)
								lbl.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
								return lbl.Layout(gtx)
							}))
						}

						// Spacer
						children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))

						// Prev / Next navigation buttons.
						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									// Handle possible click
									HandleUiButtonClick(appState, gtx, &appState.PrevButton)

									// Disable the button if we're at the beginning of the list
									if appState.CurrentIndex == 0 {
										gtx = gtx.Disabled()
									}
									return material.Button(appState.Theme, &appState.PrevButton, "← Prev").Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
								layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
									// Handle possible click
									HandleUiButtonClick(appState, gtx, &appState.NextButton)

									// Disable the button if we're at the end of the list
									if appState.CurrentIndex >= len(appState.ImageFiles)-1 {
										gtx = gtx.Disabled()
									}
									return material.Button(appState.Theme, &appState.NextButton, "Next →").Layout(gtx)
								}),
							)
						}))

						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
					})
				}),
				// Main canvas
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					canvasSize := gtx.Constraints.Max

					// Set clipping area to the canvas size and register it for events
					canvasClip := clip.Rect{Max: canvasSize}.Push(gtx.Ops)

					// Register the canvas area to receive pointer and keyboard events
					event.Op(gtx.Ops, &appState.CanvasTag)
					event.Op(gtx.Ops, &appState.KeyboardTag)
					key.InputHintOp{Tag: &appState.KeyboardTag}.Add(gtx.Ops)

					// Set initial keyboard focus so the F key works without requiring a click first
					if !appState.InitialFocusDone {
						gtx.Execute(key.FocusCmd{Tag: &appState.KeyboardTag})
						appState.InitialFocusDone = true
					}

					if appState.Settings == nil {
						// No project loaded: show centered New Project / Open Project buttons.
						promptText := "Create a new project or open an existing one."
						if appState.CurrentlyLoadingProject {
							promptText = "Loading project…"
						}
						layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									lbl := material.Body1(appState.Theme, promptText)
									lbl.Color = color.NRGBA{R: 160, G: 160, B: 160, A: 255}
									return lbl.Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if appState.NewProjectButton.Clicked(gtx) {
										triggerNewProject(appState)
									}
									if appState.CurrentlyLoadingProject {
										gtx = gtx.Disabled()
									}
									gtx.Constraints.Min.X = gtx.Dp(220)
									return material.Button(appState.Theme, &appState.NewProjectButton, "New Project").Layout(gtx)
								}),
								layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									if appState.OpenProjectButton.Clicked(gtx) {
										triggerOpenProject(appState)
									}
									if appState.CurrentlyLoadingProject {
										gtx = gtx.Disabled()
									}
									gtx.Constraints.Min.X = gtx.Dp(220)
									return material.Button(appState.Theme, &appState.OpenProjectButton, "Open Project").Layout(gtx)
								}),
							)
						})
					} else {
						// Project loaded: draw the image.
						if appState.CurrentImage != nil {
							imageWidth := float32(appState.CurrentImage.Bounds().Dx())
							imageHeight := float32(appState.CurrentImage.Bounds().Dy())
							canvasWidth := float32(canvasSize.X)
							canvasHeight := float32(canvasSize.Y)

							// Fit image to window on first render or when requested
							if appState.FitToWindow {
								scaleX := canvasWidth / imageWidth
								scaleY := canvasHeight / imageHeight
								appState.ViewScale = scaleX
								if scaleY < appState.ViewScale {
									appState.ViewScale = scaleY
								}
								appState.ViewOffsetX = (canvasWidth - imageWidth*appState.ViewScale) / 2
								appState.ViewOffsetY = (canvasHeight - imageHeight*appState.ViewScale) / 2
								appState.FitToWindow = false
							}

							// Build the image-to-canvas affine transform
							transformation := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(appState.ViewScale, appState.ViewScale)).Offset(f32.Pt(appState.ViewOffsetX, appState.ViewOffsetY))

							// Draw the base image using the cached ImageOp
							stack := op.Affine(transformation).Push(gtx.Ops)
							appState.ImgBaseOp.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							stack.Pop()

							// Draw hover outline using cached path spec
							stack = op.Affine(transformation).Push(gtx.Ops)
							paint.FillShape(gtx.Ops, color.NRGBA{R: 0, G: 200, B: 255, A: 220}, clip.Stroke{Path: appState.HoverPathSpec, Width: appState.HoverStrokeWidth / appState.ViewScale}.Op())
							stack.Pop()

							if appState.RenderSelectionOutlines {
								// Draw selection outline using cached path spec
								stack = op.Affine(transformation).Push(gtx.Ops)
								paint.FillShape(gtx.Ops, color.NRGBA{R: 255, G: 200, B: 0, A: 220}, clip.Stroke{Path: appState.SelectionPathSpec, Width: appState.SelectionStrokeWidth / appState.ViewScale}.Op())
								stack.Pop()
							}
						}

						// Small New Project / Open Project buttons overlaid at the top-left corner.
						layout.NW.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
							return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								if appState.CurrentlyLoadingProject {
									gtx = gtx.Disabled()
								}
								return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if appState.NewProjectButton.Clicked(gtx) {
											triggerNewProject(appState)
										}
										return material.Button(appState.Theme, &appState.NewProjectButton, "New Project").Layout(gtx)
									}),
									layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										if appState.OpenProjectButton.Clicked(gtx) {
											triggerOpenProject(appState)
										}
										return material.Button(appState.Theme, &appState.OpenProjectButton, "Open Project").Layout(gtx)
									}),
								)
							})
						})
					}

					canvasClip.Pop()
					return layout.Dimensions{Size: canvasSize}
				}),
			)
		}),
	)
}
