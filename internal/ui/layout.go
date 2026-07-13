package ui

import (
	"fmt"
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
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

func Layout(appState *app_config.AppState, gtx layout.Context) {

	// This is the size of the window
	size := gtx.Constraints.Max

	// We set up a clipping area that covers the entire window.
	// This ensures that all drawing operations are confined to the window bounds, preventing any rendering outside the visible area.
	bgClip := clip.Rect{Max: size}.Push(gtx.Ops)

	// Fill the background with a solid color (dark gray)
	paint.ColorOp{Color: appState.Theme.GioTheme.Palette.Bg}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	bgClip.Pop()

	// Entire application layout
	layout.Flex{Axis: layout.Vertical}.Layout(gtx,

		// Top menu bar
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			// Disable while loading
			if appState.CurrentlyLoadingProject {
				gtx = gtx.Disabled()
			}
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					HandleUiButtonClick(appState, gtx, &appState.NewProjectButton)
					return drawMenuBarButton(gtx, &appState.NewProjectButton, appState.Theme, app_theme.IconNew, "New")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					HandleUiButtonClick(appState, gtx, &appState.OpenProjectButton)
					return drawMenuBarButton(gtx, &appState.OpenProjectButton, appState.Theme, app_theme.IconOpen, "Open")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					HandleUiButtonClick(appState, gtx, &appState.OptimizeButton)
					return drawMenuBarButton(gtx, &appState.OptimizeButton, appState.Theme, app_theme.IconOptimize, "Optimize")
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					HandleUiButtonClick(appState, gtx, &appState.SaveButton)
					return drawMenuBarButton(gtx, &appState.SaveButton, appState.Theme, app_theme.IconSave, "Save")
				}),
			)
		}),

		// Main content window
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
				// Sidebar
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {

					// Set fixed width for the sidebar
					gtx.Constraints.Min.X = gtx.Dp(app_theme.SidebarWidth)
					gtx.Constraints.Max.X = gtx.Dp(app_theme.SidebarWidth)

					// Draw sidebar background
					sidebarClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
					paint.ColorOp{Color: appState.Theme.GioTheme.Palette.Bg}.Add(gtx.Ops)
					//paint.ColorOp{Color: color.NRGBA{R: 25, G: 25, B: 25, A: 255}}.Add(gtx.Ops)
					paint.PaintOp{}.Add(gtx.Ops)
					sidebarClip.Pop()

					// Layout sliders inside the sidebar
					inset := layout.UniformInset(unit.Dp(8))
					return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						var children []layout.FlexChild

						if appState.Settings != nil {

							parameterInfo, err := superpixel.ListParameterInfo(appState.Settings.Superpixel.SpAlgorithmId)
							if err != nil {
								fmt.Printf("Error listing parameter info: %v\n", err)
								return layout.Dimensions{}
							}

							for _, paramInfo := range parameterInfo {
								children = append(children,
									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										label := material.Body1(&appState.Theme.GioTheme, paramInfo.Title)
										label.Color = appState.Theme.GioTheme.Palette.Fg
										return label.Layout(gtx)
									}),

									layout.Rigid(func(gtx layout.Context) layout.Dimensions {
										currentEditor := appState.SuperpixelParameterEditors[paramInfo.Id]
										if currentEditor == nil {
											fmt.Printf("Editor for parameter %s not found\n", paramInfo.Id)
											return layout.Dimensions{}
										}

										editorStyle := material.Editor(&appState.Theme.GioTheme, currentEditor, "value")
										editorStyle.Color = appState.Theme.GioTheme.Palette.Fg
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
									lbl := material.Body2(&appState.Theme.GioTheme, counter)
									lbl.Color = appState.Theme.GioTheme.ContrastFg
									return lbl.Layout(gtx)
								}))
							}

							// Spacer
							children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout))

							// Image navigation buttons.
							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return drawImageNavigationButtons(appState, gtx)
							}))
						}

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
									lbl := material.Body1(&appState.Theme.GioTheme, promptText)
									lbl.Color = appState.Theme.GioTheme.ContrastFg
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
									return drawBigButton(gtx, &appState.NewProjectButton, appState.Theme, "New Project", 16)
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
									return drawBigButton(gtx, &appState.OpenProjectButton, appState.Theme, "Open Project", 16)
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
							drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.HoverPathSpec, appState.Theme.HoverOutlineColor, appState.Theme.HoverStrokeWidth, 0)

							if appState.RenderSelectionOutlines {
								// Halo (wider, dark stroke drawn first)
								drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.SelectionPathSpec, appState.Theme.SelectionOutlineColor, appState.Theme.SelectionStrokeWidth, appState.Theme.StrokeHaloWidth)

								// Draw existing mask outline (orange) for any mask loaded from disk
								drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.ExistingMaskPathSpec, appState.Theme.ExistingMaskOutlineColor, appState.Theme.ExistingMaskStrokeWidth, 0)
							}
						}

					}

					canvasClip.Pop()
					return layout.Dimensions{Size: canvasSize}
				}),
			)
		}),
	)
}
