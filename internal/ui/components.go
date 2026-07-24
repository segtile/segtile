package ui

import (
	"fmt"
	"image/color"

	"gioui.org/f32"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/struffel/annotation-tool/internal/app_config"
	"github.com/struffel/annotation-tool/internal/ui/app_theme"
	"github.com/struffel/annotation-tool/pkg/superpixel"
)

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
	buttonStyle.CornerRadius = 0
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

func drawSpSettingsEditors(gtx layout.Context, appState *app_config.AppState) layout.Dimensions {

	if appState.Settings == nil {
		return layout.Dimensions{}
	}
	HandleUiButtonClick(appState, gtx, &appState.ChangeSpAlgorithmButton)
	if appState.SpAlgorithmMenuOpen {
		for i := range appState.SpAlgorithmMenuButtons {
			HandleUiButtonClick(appState, gtx, &appState.SpAlgorithmMenuButtons[i])
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

					children := make([]layout.FlexChild, 0, len(appState.SpAlgorithmMenuAlgorithmIds))
					for i := range appState.SpAlgorithmMenuAlgorithmIds {

						button := &appState.SpAlgorithmMenuButtons[i]

						children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							HandleUiButtonClick(appState, gtx, button)
							return component.MenuItem(&appState.Theme.GioTheme, button, appState.SpAlgoritmMenuTitles[i]).Layout(gtx)
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

		for i := range appState.SuperpixelParameterEditors {
			title := appState.SuperpixelParameterEditorTitles[i]
			editor := &appState.SuperpixelParameterEditors[i]
			checkbox := &appState.OptimizeParameterCheckboxes[i]
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
	}

	return layout.Flex{Axis: layout.Vertical, Spacing: layout.SpaceEnd}.Layout(gtx, children...)

}

func drawImageNavigationButtons(appState *app_config.AppState, gtx layout.Context) layout.Dimensions {
	atFirstImage := appState.CurrentIndex == 0
	atLastImage := appState.CurrentIndex == len(appState.ImageFiles)-1

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
	haloWidth float32) {

	if haloWidth > 0 {
		// Halo/Outline
		*stack = op.Affine(transformation).Push(gtx.Ops)
		paint.FillShape(gtx.Ops,
			color.NRGBA{R: 0, G: 0, B: 0, A: 200},
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
