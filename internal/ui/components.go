package ui

import (
	"fmt"
	"image"
	"image/color"
	"path/filepath"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/text"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/struffel/annotation-tool/internal/core"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

// drawProjectLoadError renders the error from a failed project load/creation attempt,
// shown under the New/Open Project buttons on the start screen.
func drawProjectLoadError(gtx layout.Context, appState *core.AppState) layout.Dimensions {

	if appState.Settings == nil || appState.Settings.Err == nil {
		return layout.Dimensions{}
	}

	label := material.Body2(&appState.Theme.GioTheme, appState.Settings.Err.Error())
	label.Color = color.NRGBA{R: 220, G: 80, B: 80, A: 255}
	label.Alignment = text.Start
	return label.Layout(gtx)
}

func drawSavedImageInfo(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	if appState.ActiveImageMetadata == nil {
		return layout.Dimensions{}
	}

	annotator := appState.ActiveImageMetadata.Annotator
	if annotator == "" {
		annotator = "Unknown"
	}
	return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				label := material.Body2(&appState.Theme.GioTheme, "Annotator: "+annotator)
				label.Color = appState.Theme.GioTheme.Palette.Fg
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				buttonGtx := gtx
				if appState.SelectionDiffersVersionOnDisk {
					HandleUiButtonClick(appState, gtx, &appState.RevertSavedMaskButton)
				} else {
					buttonGtx = gtx.Disabled()
				}
				return drawBigButton(buttonGtx, &appState.RevertSavedMaskButton, appState.Theme, "Revert to saved mask", 14)
			}),
		)
	})
}

// drawIconButton draws a button with an icon and handles its click event.
func drawIconButton(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, icon *widget.Icon, description string, filled bool) layout.Dimensions {
	style := material.IconButton(&theme.GioTheme, btn, icon, description)
	style.Size = unit.Dp(20)
	style.Inset = layout.UniformInset(unit.Dp(4))
	if filled {
		style.Color = theme.GioTheme.Palette.ContrastFg
		style.Background = theme.GioTheme.Palette.ContrastBg
	} else {
		style.Color = theme.GioTheme.Palette.Fg
		style.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 0}
	}
	return style.Layout(gtx)
}

func drawBigButton(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, txt string, size float32) layout.Dimensions {
	buttonStyle := material.Button(&theme.GioTheme, btn, txt)
	buttonStyle.TextSize = unit.Sp(size)
	return buttonStyle.Layout(gtx)
}

func drawMenuBarButton(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, icon *widget.Icon, txt string) layout.Dimensions {

	buttonStyle := material.ButtonLayout(&theme.GioTheme, btn)
	buttonStyle.CornerRadius = 2
	buttonStyle.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 0}

	menuButtonInset := layout.Inset{
		Top:    unit.Dp(app_theme.MenuBarVerticalSpacing),
		Bottom: unit.Dp(app_theme.MenuBarVerticalSpacing),
		Left:   unit.Dp(app_theme.MenuBarHorizontalSpacing),
		Right:  unit.Dp(app_theme.MenuBarHorizontalSpacing),
	}

	return buttonStyle.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return menuButtonInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Start}.Layout(gtx,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					if icon == nil {
						return layout.Dimensions{}
					}
					iconInset := layout.Inset{
						Right: unit.Dp(app_theme.MenuBarIconSpacing),
					}
					return iconInset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						gtx.Constraints.Min.X = gtx.Dp(unit.Dp(app_theme.MenuBarIconSize))
						gtx.Constraints.Min.Y = gtx.Dp(unit.Dp(app_theme.MenuBarIconSize))
						return icon.Layout(gtx, theme.GioTheme.Palette.Fg)
					})
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					label := material.Body2(&theme.GioTheme, txt)
					label.Color = theme.GioTheme.Palette.ContrastFg
					return label.Layout(gtx)
				}),
			)
		})
	})
}

func drawSidebarSectionHeader(gtx layout.Context, btn *widget.Clickable, theme *app_theme.AppTheme, title string, open bool) layout.Dimensions {
	buttonStyle := material.ButtonLayout(&theme.GioTheme, btn)
	buttonStyle.CornerRadius = 2
	buttonStyle.Background = color.NRGBA{R: 0, G: 0, B: 0, A: 0}

	return buttonStyle.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		label := material.Body2(&theme.GioTheme, title)
		label.Color = theme.GioTheme.Palette.ContrastFg
		label.Alignment = text.Start

		iconLabel := material.Body2(&theme.GioTheme, "")
		if open {
			iconLabel.Text = "▾"
		} else {
			iconLabel.Text = "▸"
		}
		iconLabel.Color = theme.GioTheme.Palette.ContrastFg
		iconLabel.Alignment = text.End

		return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
			layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return iconLabel.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(6)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return label.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(2)}.Layout),
			layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
				return drawDivider(gtx, theme, unit.Dp(2))
			}),
			layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		)
	})
}

// drawSidebarSection draws a collapsible sidebar section header followed by its body when open.
func drawSidebarSection(gtx layout.Context, appState *core.AppState, title string, open bool, button *widget.Clickable, body func(layout.Context) layout.Dimensions) layout.Dimensions {
	HandleUiButtonClick(appState, gtx, button)

	return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return drawSidebarSectionHeader(gtx, button, appState.Theme, title, open)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			if !open {
				return layout.Dimensions{}
			}
			return layout.UniformInset(unit.Dp(8)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return body(gtx)
			})
		}),
	)
}

func drawImagesSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	return drawSidebarSection(gtx, appState, "Images", appState.ImagesSection.Open, &appState.ImagesSection.Button, func(gtx layout.Context) layout.Dimensions {
		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,

			// Image name and counter
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(appState.ImageFiles) == 0 {
					return layout.Dimensions{}
				}
				counter := fmt.Sprintf("%d / %d",
					appState.ImageFilesIndex+1,
					len(appState.ImageFiles),
				)
				lbl := material.Caption(&appState.Theme.GioTheme, counter)
				lbl.Color = appState.Theme.GioTheme.ContrastFg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				if len(appState.ImageFiles) == 0 {
					return layout.Dimensions{}
				}
				lbl := material.Body2(&appState.Theme.GioTheme, filepath.Base(appState.ImageFiles[appState.ImageFilesIndex]))
				lbl.Color = appState.Theme.GioTheme.ContrastFg
				return lbl.Layout(gtx)
			}),
			layout.Rigid(layout.Spacer{Height: unit.Dp(4)}.Layout),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return drawImageNavigationButtons(gtx, appState)
			}),
		)
	})
}

func drawSuperpixelSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	return drawSidebarSection(gtx, appState, "Superpixel", appState.SuperpixelSection.Open, &appState.SuperpixelSection.Button, func(gtx layout.Context) layout.Dimensions {
		if appState.Settings == nil {
			return layout.Dimensions{}
		}
		HandleUiButtonClick(appState, gtx, &appState.ChangeSpAlgorithmButton)
		if appState.SpAlgorithmMenuOpen {
			for i := range appState.SpAlgorithmMenuEntries {
				HandleUiButtonClick(appState, gtx, &appState.SpAlgorithmMenuEntries[i].Button)
			}
		}

		var children []layout.FlexChild

		// Algorithm Name and change button
		children = append(children,
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {

								superpixelTitle, _, _, err := superpixel.GetAlgorithmInfo(appState.Settings.Superpixel.SpAlgorithmId)
								if err != nil {
									superpixelTitle = fmt.Sprintf("Untitled (%s)", appState.Settings.Superpixel.SpAlgorithmId)
								}

								label := material.Body1(&appState.Theme.GioTheme, superpixelTitle)
								label.Color = appState.Theme.GioTheme.Palette.Fg
								return label.Layout(gtx)
							}),
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								return drawIconButton(gtx, &appState.ChangeSpAlgorithmButton, appState.Theme, app_theme.IconChangeAlgorithm, "Change Superpixel Algorithm", false)
							}),
						)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						if !appState.SpAlgorithmMenuOpen {
							return layout.Dimensions{}
						}

						children := make([]layout.FlexChild, 0, len(appState.SpAlgorithmMenuEntries))
						for i := range appState.SpAlgorithmMenuEntries {

							button := &appState.SpAlgorithmMenuEntries[i].Button
							title := appState.SpAlgorithmMenuEntries[i].Title

							children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								HandleUiButtonClick(appState, gtx, button)
								return component.MenuItem(&appState.Theme.GioTheme, button, title).Layout(gtx)
							}))
						}
						return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
					}),
				)
			}),
		)

		if !appState.SpAlgorithmMenuOpen {

			// Spacer
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)

			// Individual parameter editors
			for i := range appState.SuperpixelParameterEditors {
				title := appState.SuperpixelParameterEditors[i].Title
				editor := &appState.SuperpixelParameterEditors[i].Editor
				checkbox := &appState.SuperpixelParameterEditors[i].OptimizeCheckbox
				children = append(children,
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						return layout.Flex{Axis: layout.Horizontal, Alignment: layout.End}.Layout(gtx,
							layout.Rigid(func(gtx layout.Context) layout.Dimensions {
								checkboxStyle := material.CheckBox(&appState.Theme.GioTheme, checkbox, "")
								checkboxStyle.IconColor = appState.Theme.GioTheme.Palette.Fg
								checkboxStyle.Size = unit.Dp(16)
								return checkboxStyle.Layout(gtx)
							}),
							layout.Flexed(3, func(gtx layout.Context) layout.Dimensions {
								label := material.Body1(&appState.Theme.GioTheme, title)
								label.Color = appState.Theme.GioTheme.Palette.Fg
								return label.Layout(gtx)
							}),

							layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
								editorStyle := material.Editor(&appState.Theme.GioTheme, editor, "value")
								editorStyle.Color = appState.Theme.GioTheme.Palette.Fg
								return editorStyle.Layout(gtx)
							}),
						)
					}),
				)
			}

			// Optimize button
			children = append(children,
				layout.Rigid(layout.Spacer{Height: unit.Dp(8)}.Layout),
			)
			children = append(children,
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					HandleUiButtonClick(appState, gtx, &appState.OptimizeButton)
					return drawMenuBarButton(gtx, &appState.OptimizeButton, appState.Theme, app_theme.IconOptimize, "Suggest Parameters")
				}),
			)
		}

		return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx, children...)
	})
}

func drawSelectionSection(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	return drawSidebarSection(gtx, appState, "Selection", appState.MiscSection.Open, &appState.MiscSection.Button, func(gtx layout.Context) layout.Dimensions {
		children := []layout.FlexChild{
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.CheckBox(&appState.Theme.GioTheme, &appState.EnableAutoFloodFill, "Enable Auto Flood Fill").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.CheckBox(&appState.Theme.GioTheme, &appState.UpdateSelectionOnSpChange, "Snap selection when SP change").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Label(&appState.Theme.GioTheme, unit.Sp(12), "Shrink / Grow").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {

				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Slider(&appState.Theme.GioTheme, &appState.ShrinkGrowAmount).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						HandleUiButtonClick(appState, gtx, &appState.ShrinkGrowAmountReset)
						return drawIconButton(gtx, &appState.ShrinkGrowAmountReset, appState.Theme, app_theme.IconAvReplay, "Reset Shrink/Grow", false)
					}),
				)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {
				return material.Label(&appState.Theme.GioTheme, unit.Sp(12), "Smooth").Layout(gtx)
			}),
			layout.Rigid(func(gtx layout.Context) layout.Dimensions {

				return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
					layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
						return material.Slider(&appState.Theme.GioTheme, &appState.SmoothAmount).Layout(gtx)
					}),
					layout.Rigid(func(gtx layout.Context) layout.Dimensions {
						HandleUiButtonClick(appState, gtx, &appState.SmoothAmountReset)
						return drawIconButton(gtx, &appState.SmoothAmountReset, appState.Theme, app_theme.IconAvReplay, "Reset Smooth", false)
					}),
				)
			}),
		}

		return layout.Flex{Axis: layout.Vertical}.Layout(gtx,
			children...,
		)
	})
}

func drawImageNavigationButtons(gtx layout.Context, appState *core.AppState) layout.Dimensions {
	atFirstImage := appState.ImageFilesIndex == 0
	atLastImage := appState.ImageFilesIndex == len(appState.ImageFiles)-1

	return layout.Flex{Axis: layout.Horizontal}.Layout(gtx,
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if atFirstImage {
				gtx = gtx.Disabled()
			} else {
				HandleUiButtonClick(appState, gtx, &appState.FirstButton)
			}
			return drawIconButton(gtx, &appState.FirstButton, appState.Theme, app_theme.IconFirst, "First", true)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if atFirstImage {
				gtx = gtx.Disabled()
			} else {
				HandleUiButtonClick(appState, gtx, &appState.PrevUnlabeledButton)
			}
			return drawIconButton(gtx, &appState.PrevUnlabeledButton, appState.Theme, app_theme.IconPrevUnlabeled, "Previous Unlabeled", true)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if atFirstImage {
				gtx = gtx.Disabled()
			} else {
				HandleUiButtonClick(appState, gtx, &appState.PrevButton)
			}
			return drawIconButton(gtx, &appState.PrevButton, appState.Theme, app_theme.IconPrev, "Previous", true)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {

			if atLastImage {
				gtx = gtx.Disabled()
			} else {
				HandleUiButtonClick(appState, gtx, &appState.NextButton)
			}
			return drawIconButton(gtx, &appState.NextButton, appState.Theme, app_theme.IconNext, "Next", true)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if atLastImage {
				gtx = gtx.Disabled()
			} else {
				HandleUiButtonClick(appState, gtx, &appState.NextUnlabeledButton)
			}
			return drawIconButton(gtx, &appState.NextUnlabeledButton, appState.Theme, app_theme.IconNextUnlabeled, "Next Unlabeled", true)
		}),
		layout.Rigid(layout.Spacer{Width: unit.Dp(4)}.Layout),
		layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
			if atLastImage {
				gtx = gtx.Disabled()
			} else {
				HandleUiButtonClick(appState, gtx, &appState.LastButton)
			}
			return drawIconButton(gtx, &appState.LastButton, appState.Theme, app_theme.IconLast, "Last", true)
		}),
	)
}

// drawStroke adds the drawing operations for an outline to the graphics context
// The halo is an additional outline around the main line.
func drawStroke(
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

func drawDivider(gtx layout.Context, theme *app_theme.AppTheme, thickness unit.Dp) layout.Dimensions {
	w := gtx.Dp(thickness)
	size := image.Pt(gtx.Constraints.Max.X, w)

	// Alpha = 128 -> 50 % opacity
	paint.FillShape(gtx.Ops, color.NRGBA{R: theme.GioTheme.Fg.R, G: theme.GioTheme.Fg.G, B: theme.GioTheme.Fg.B, A: 128}, clip.Rect{Max: size}.Op())
	return layout.Dimensions{Size: size}
}
