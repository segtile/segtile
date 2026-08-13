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
)

func Layout(appState *app_config.AppState, gtx layout.Context) {

	// Disable the entire UI if a background operation owns project state.
	if appState.CurrentlyLoadingProject || appState.CurrentlyOptimizing {
		gtx = gtx.Disabled()
	}

	// Determine if a project is loaded
	projectLoaded := appState.Settings != nil

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
	layout.Stack{Alignment: layout.S}.Layout(gtx,

		// Main content layout
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

				// Top menu bar
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					// Do not render menu bar if no project is loaded
					if !projectLoaded {
						return layout.Dimensions{}
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
							HandleUiButtonClick(appState, gtx, &appState.SaveButton)
							return drawMenuBarButton(gtx, &appState.SaveButton, appState.Theme, app_theme.IconSave, "Save")
						}),
					)
				}),

				// Main content window (sidebar + canvas)
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,

						// Sidebar
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {

							// Do not render sidebar if no project is loaded
							if !projectLoaded {
								return layout.Dimensions{}
							}

							// Set fixed width for the sidebar
							gtx.Constraints.Min.X = gtx.Dp(app_theme.SidebarWidth)
							gtx.Constraints.Max.X = gtx.Dp(app_theme.SidebarWidth)

							// Draw sidebar background
							sidebarClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
							paint.ColorOp{Color: appState.Theme.GioTheme.Palette.Bg}.Add(gtx.Ops)
							//paint.ColorOp{Color: color.NRGBA{R: 25, G: 25, B: 25, A: 255}}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							sidebarClip.Pop()

							// Layout buttons and editors
							inset := layout.UniformInset(unit.Dp(8))
							return inset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								var children []layout.FlexChild

								// Image counter label.
								if len(appState.ImageFiles) > 0 {
									counter := fmt.Sprintf("%d / %d  —  %s",
										appState.ImageFilesIndex+1,
										len(appState.ImageFiles),
										filepath.Base(appState.ImageFiles[appState.ImageFilesIndex]),
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

								// Spacer between param editors and navigation controls.
								children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout))

								// Draw the superpixel parameter editors
								children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return drawSpSettingsEditors(gtx, appState)
								}))

								// Spacer between param editors and navigation controls.
								children = append(children, layout.Rigid(layout.Spacer{Height: unit.Dp(12)}.Layout))

								// Auto flood fill checkbox
								children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return material.CheckBox(&appState.Theme.GioTheme, &appState.EnableAutoFloodFill, "Enable Auto Flood Fill").Layout(gtx)
								}))

								return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
							})

						}),
						// Main canvas
						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {

							// Set clipping area to the canvas size and register it for events
							canvasClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)

							// Register the canvas area to receive pointer and keyboard events
							event.Op(gtx.Ops, &appState.CanvasTag)
							event.Op(gtx.Ops, &appState.KeyboardTag)
							key.InputHintOp{Tag: &appState.KeyboardTag}.Add(gtx.Ops)

							// Set initial keyboard focus so the F key works without requiring a click first
							if !appState.InitialFocusDone {
								gtx.Execute(key.FocusCmd{Tag: &appState.KeyboardTag})
								appState.InitialFocusDone = true
							}

							if !projectLoaded {
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

											gtx.Constraints.Min.X = gtx.Dp(220)
											return drawBigButton(gtx, &appState.NewProjectButton, appState.Theme, "New Project", 16)
										}),
										layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
										layout.Rigid(func(gtx layout.Context) layout.Dimensions {
											if appState.OpenProjectButton.Clicked(gtx) {
												triggerOpenProject(appState)
											}

											gtx.Constraints.Min.X = gtx.Dp(220)
											return drawBigButton(gtx, &appState.OpenProjectButton, appState.Theme, "Open Project", 16)
										}),
									)
								})
							} else {
								// Project loaded: draw the image.
								if appState.ActiveImage != nil {

									// Fit the image to the window if requested
									if appState.FitToWindow {
										appState.FitImageToWindow(gtx.Constraints.Max)
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
										// Draw selection outline for the current selection
										drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.SelectionPathSpec, appState.Theme.SelectionOutlineColor, appState.Theme.SelectionStrokeWidth, appState.Theme.StrokeHaloWidth)

										if !appState.SelectionMade() {

											// Draw existing mask outline for any mask loaded from disk
											drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.ExistingMaskPathSpec, appState.Theme.ExistingMaskOutlineColor, appState.Theme.ExistingMaskStrokeWidth, 0)
										}
									}
								}

							}

							canvasClip.Pop()
							return layout.Dimensions{Size: gtx.Constraints.Max}
						}),
					)
				}),
			)
		}),

		// Toasts
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return appState.Toasts.Layout(gtx, &appState.Theme.GioTheme)
		}),
	)
}
