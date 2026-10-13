package ui

import (
	"gioui.org/f32"
	"gioui.org/io/event"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget/material"
	"github.com/struffel/annotation-tool/internal/core"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
)

func Layout(appState *core.AppState, gtx layout.Context) {

	// Disable the entire UI if a background operation owns project state.
	if appState.UiLocked() {
		gtx = gtx.Disabled()
	}

	// Determine if a project is loaded
	hasActiveProject := appState.Settings != nil && appState.Settings.Err == nil
	isLoadingProject := appState.LoadingState.CurrentlyLoadingProject

	// We set up a clipping area that covers the entire window.
	// This ensures that all drawing operations are confined to the window bounds, preventing any rendering outside the visible area.
	bgClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)

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
					if !hasActiveProject {
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
							if !hasActiveProject {
								return layout.Dimensions{}
							}

							// Set fixed width for the sidebar
							gtx.Constraints.Min.X = gtx.Dp(app_theme.SidebarWidth)
							gtx.Constraints.Max.X = gtx.Dp(app_theme.SidebarWidth)

							// Draw sidebar background
							sidebarClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
							paint.ColorOp{Color: appState.Theme.GioTheme.Palette.Bg}.Add(gtx.Ops)
							paint.PaintOp{}.Add(gtx.Ops)
							sidebarClip.Pop()

							// Draw sidebar sections.
							return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return drawImagesSection(gtx, appState)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return drawSuperpixelSection(gtx, appState)
								}),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return drawSelectionSection(gtx, appState)
								}),
								layout.Flexed(1, layout.Spacer{Height: unit.Dp(12)}.Layout),
								layout.Rigid(func(gtx layout.Context) layout.Dimensions {
									return drawExistingMaskWarning(gtx, appState)
								}),
							)
						}),

						layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
							canvasClip := clip.Rect{Max: gtx.Constraints.Max}.Push(gtx.Ops)
							event.Op(gtx.Ops, &appState.CanvasTag)
							event.Op(gtx.Ops, &appState.KeyboardTag)
							key.InputHintOp{Tag: &appState.KeyboardTag}.Add(gtx.Ops)

							if !appState.InitialFocusDone {
								gtx.Execute(key.FocusCmd{Tag: &appState.KeyboardTag})
								appState.InitialFocusDone = true
							}

							if !hasActiveProject {

								gtx.Constraints.Min = gtx.Constraints.Max
								layout.Center.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									if isLoadingProject {
										return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												lbl := material.Body1(&appState.Theme.GioTheme, "Loading project…")
												lbl.Color = appState.Theme.GioTheme.ContrastFg
												return lbl.Layout(gtx)
											}))
									} else {
										return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle}.Layout(gtx,
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												lbl := material.Body1(&appState.Theme.GioTheme, "Create a new project or open an existing one.")
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
											}), layout.Rigid(layout.Spacer{Height: unit.Dp(16)}.Layout),
											layout.Rigid(func(gtx layout.Context) layout.Dimensions {
												gtx.Constraints.Max.X = gtx.Dp(480)
												gtx.Constraints.Min.X = 0
												return drawProjectLoadError(gtx, appState)
											}))
									}
								})
							} else if appState.ActiveImage != nil {
								if appState.FitToWindow {
									appState.FitImageToWindow(gtx.Constraints.Max)
									appState.FitToWindow = false
								}

								transformation := f32.Affine2D{}.Scale(f32.Pt(0, 0), f32.Pt(appState.ViewScale, appState.ViewScale)).Offset(appState.ViewOffset)

								stack := op.Affine(transformation).Push(gtx.Ops)
								appState.ImgBaseOp.Add(gtx.Ops)
								paint.PaintOp{}.Add(gtx.Ops)
								stack.Pop()

								drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.HoverPathSpec, appState.Theme.HoverOutlineColor, appState.Theme.HoverStrokeWidth, appState.Theme.HoverHaloColor, appState.Theme.HoverHaloWidth)

								if appState.RenderSelectionOutlines {
									drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.SelectionPathSpec, appState.Theme.SelectionOutlineColor, appState.Theme.SelectionStrokeWidth, appState.Theme.SelectionHaloColor, appState.Theme.SelectionHaloWidth)
									if appState.AwaitingMaskDecision {
										drawStroke(&stack, gtx, transformation, appState.ViewScale, appState.ExistingMaskPathSpec, appState.Theme.ExistingMaskOutlineColor, appState.Theme.ExistingMaskStrokeWidth, appState.Theme.ExistingMaskHaloColor, appState.Theme.ExistingMaskHaloWidth)
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
		layout.Stacked(func(gtx layout.Context) layout.Dimensions {
			return appState.Toasts.Layout(gtx, &appState.Theme.GioTheme)
		}),
	)
}
